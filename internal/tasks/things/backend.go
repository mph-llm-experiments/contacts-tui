package things

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
	
	"github.com/pdxmph/contacts-tui/internal/config"
	"github.com/pdxmph/contacts-tui/internal/tasks"
)

// thingsTask represents a Things task as returned by JXA
type thingsTask struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Tags        []string `json:"tags"`
	Notes       string   `json:"notes,omitempty"`
	DueDate     string   `json:"dueDate,omitempty"`
	CreatedDate string   `json:"createdDate,omitempty"`
	ModifiedDate string  `json:"modifiedDate,omitempty"`
}

// Backend implements the tasks.Backend interface for Things 3
type Backend struct {
	enabled   bool
	authToken string
}

// NewBackend creates a new Things backend
func NewBackend() tasks.Backend {
	backend := &Backend{
		enabled: isThingsAvailable(),
	}
	
	// Load auth token from config if available
	if cfg, err := config.Load(); err == nil {
		backend.authToken = cfg.Tasks.Things.AuthToken
	}
	
	return backend
}

// Name returns the backend identifier
func (b *Backend) Name() string {
	return "things"
}

// IsEnabled returns whether Things integration is available
func (b *Backend) IsEnabled() bool {
	return b.enabled
}

// CreateContactTask creates a Things task for a contact state change
func (b *Backend) CreateContactTask(contactName, state, label string) error {
	if !b.enabled {
		return fmt.Errorf("Things not available")
	}

	if b.authToken == "" {
		return fmt.Errorf("Things auth token not configured - see README for setup instructions")
	}

	if label == "" {
		return fmt.Errorf("contact must have a label to create Things task")
	}

	// Format task description based on state
	description := formatTaskDescription(state, contactName)
	
	// Ensure label starts with @
	if !strings.HasPrefix(label, "@") {
		label = "@" + label
	}

	// Build Things URL with auth token
	// Format: things:///add?title=TITLE&tags=TAG1,TAG2&auth-token=TOKEN
	params := url.Values{}
	params.Set("title", description)
	params.Set("tags", fmt.Sprintf("%s,contact-%s", label, state))
	params.Set("auth-token", b.authToken)
	
	thingsURL := "things:///add?" + params.Encode()
	
	// Open the URL to create the task
	cmd := exec.Command("open", thingsURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("creating task: %w (output: %s)", err, string(output))
	}

	return nil
}

// GetContactTasks retrieves all tasks for a contact by their label
func (b *Backend) GetContactTasks(label string) ([]tasks.Task, error) {
	if !b.enabled {
		return nil, fmt.Errorf("Things not available")
	}

	if label == "" {
		return []tasks.Task{}, nil
	}

	// Ensure label starts with @
	if !strings.HasPrefix(label, "@") {
		label = "@" + label
	}

	// JXA script to find tasks with the label tag
	jxaScript := fmt.Sprintf(`
		const things = Application('Things3');
		const todos = things.toDos();
		const result = [];
		
		for (let i = 0; i < todos.length; i++) {
			const todo = todos[i];
			const tags = todo.tags().map(t => t.name());
			
			if (tags.includes('%s')) {
				const createdDate = todo.creationDate();
				const modifiedDate = todo.modificationDate();
				const dueDate = todo.dueDate();
				
				result.push({
					id: todo.id(),
					name: todo.name(),
					status: todo.status(),
					tags: tags,
					notes: todo.notes(),
					createdDate: createdDate ? createdDate.toISOString() : null,
					modifiedDate: modifiedDate ? modifiedDate.toISOString() : null,
					dueDate: dueDate ? dueDate.toISOString() : null
				});
			}
		}
		
		JSON.stringify(result);
	`, label)

	cmd := exec.Command("osascript", "-l", "JavaScript", "-e", jxaScript)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("querying tasks: %w", err)
	}

	var thingsTasks []thingsTask
	if len(output) > 0 {
		err = json.Unmarshal(output, &thingsTasks)
		if err != nil {
			return nil, fmt.Errorf("parsing task JSON: %w", err)
		}
	}

	// Convert Things tasks to generic tasks
	genericTasks := make([]tasks.Task, len(thingsTasks))
	for i, tTask := range thingsTasks {
		genericTasks[i] = convertToGenericTask(tTask)
	}

	return genericTasks, nil
}

// CompleteTask marks a task as completed
func (b *Backend) CompleteTask(taskID string, completionNote string) error {
	if !b.enabled {
		return fmt.Errorf("Things not available")
	}

	// Build the notes update part if completion note is provided
	notesUpdate := ""
	if completionNote != "" {
		// Escape the completion note for JavaScript
		escapedNote := strings.ReplaceAll(completionNote, `"`, `\"`)
		escapedNote = strings.ReplaceAll(escapedNote, "\n", "\\n")
		
		notesUpdate = fmt.Sprintf(`
			const currentNotes = todo.notes();
			const separator = currentNotes ? "\\n\\n" : "";
			todo.notes = currentNotes + separator + "Completed: %s";
		`, escapedNote)
	}

	// JXA script to complete the task
	jxaScript := fmt.Sprintf(`
		const things = Application('Things3');
		
		try {
			// Find task by ID
			const todos = things.toDos.whose({id: '%s'});
			
			if (todos.length === 0) {
				throw new Error("Task not found with ID: %s");
			}
			
			const todo = todos[0];
			
			// Add completion note if provided
			%s
			
			// Mark as completed
			todo.status = 'completed';
			
			JSON.stringify({success: true, taskName: todo.name()});
		} catch (e) {
			JSON.stringify({error: e.toString()});
		}
	`, taskID, taskID, notesUpdate)

	cmd := exec.Command("osascript", "-l", "JavaScript", "-e", jxaScript)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("completing task: %w", err)
	}

	// Parse the result
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("parsing completion result: %w", err)
	}

	if errMsg, ok := result["error"].(string); ok {
		return fmt.Errorf(errMsg)
	}

	return nil
}

// convertToGenericTask converts a Things task to the generic Task type
func convertToGenericTask(tTask thingsTask) tasks.Task {
	task := tasks.Task{
		ID:          tTask.ID,
		Description: tTask.Name,
		Status:      mapThingsStatus(tTask.Status),
		Tags:        tTask.Tags,
		Metadata: map[string]interface{}{
			"notes": tTask.Notes,
		},
	}

	// Parse timestamps
	if tTask.CreatedDate != "" {
		if t, err := time.Parse(time.RFC3339, tTask.CreatedDate); err == nil {
			task.Created = t
		}
	}
	
	if tTask.ModifiedDate != "" {
		if t, err := time.Parse(time.RFC3339, tTask.ModifiedDate); err == nil {
			task.Modified = t
		}
	}
	
	if tTask.DueDate != "" {
		if t, err := time.Parse(time.RFC3339, tTask.DueDate); err == nil {
			task.Due = &t
		}
	}

	return task
}

// mapThingsStatus converts Things status to generic status
func mapThingsStatus(status string) string {
	switch status {
	case "open":
		return "pending"
	case "completed":
		return "completed"
	case "canceled":
		return "canceled"
	default:
		return status
	}
}

// isThingsAvailable checks if Things 3 is installed (macOS only)
func isThingsAvailable() bool {
	// Only available on macOS
	if runtime.GOOS != "darwin" {
		return false
	}

	// Check if Things3.app exists
	thingsPath := "/Applications/Things3.app"
	if _, err := os.Stat(thingsPath); err == nil {
		return true
	}

	// Also check in user Applications
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userThingsPath := fmt.Sprintf("%s/Applications/Things3.app", homeDir)
		if _, err := os.Stat(userThingsPath); err == nil {
			return true
		}
	}

	return false
}

// formatTaskDescription creates a task description based on contact state
func formatTaskDescription(state, contactName string) string {
	switch strings.ToLower(state) {
	case "ping":
		return fmt.Sprintf("Ping %s", contactName)
	case "followup":
		return fmt.Sprintf("Follow up with %s", contactName)
	case "invite":
		return fmt.Sprintf("Send invitation to %s", contactName)
	case "write":
		return fmt.Sprintf("Write to %s", contactName)
	case "scheduled":
		return fmt.Sprintf("Meeting scheduled with %s", contactName)
	case "timeout":
		return fmt.Sprintf("Check timeout status for %s", contactName)
	default:
		return fmt.Sprintf("%s: %s", strings.Title(state), contactName)
	}
}

// Register the Things backend
func init() {
	tasks.Register("things", func() tasks.Backend { return NewBackend() })
}
