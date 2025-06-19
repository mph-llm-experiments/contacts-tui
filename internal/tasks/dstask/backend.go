package dstask

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
	
	"github.com/pdxmph/contacts-tui/internal/tasks"
)

// dstaskTask represents a dstask task in its native JSON format
type dstaskTask struct {
	ID       int      `json:"id"`
	UUID     string   `json:"uuid"`
	Summary  string   `json:"summary"`
	Status   string   `json:"status"`
	Tags     []string `json:"tags"`
	Created  string   `json:"created"`
	Modified string   `json:"modified"`
	Due      string   `json:"due,omitempty"`
	Priority string   `json:"priority,omitempty"`
	Notes    string   `json:"notes,omitempty"`
}

// Backend implements the tasks.Backend interface for dstask
type Backend struct {
	enabled bool
}

// NewBackend creates a new dstask backend
func NewBackend() tasks.Backend {
	return &Backend{
		enabled: isDstaskAvailable(),
	}
}

// Name returns the backend identifier
func (b *Backend) Name() string {
	return "dstask"
}

// IsEnabled returns whether dstask integration is available
func (b *Backend) IsEnabled() bool {
	return b.enabled
}

// CreateContactTask creates a dstask task for a contact state change
func (b *Backend) CreateContactTask(contactName, state, label string) error {
	if !b.enabled {
		return fmt.Errorf("dstask not available")
	}

	if label == "" {
		return fmt.Errorf("contact must have a label to create dstask task")
	}

	// Format task description based on state
	description := formatTaskDescription(state, contactName)
	
	// Ensure label starts with @
	if !strings.HasPrefix(label, "@") {
		label = "@" + label
	}

	// Create the task with label and state as tags
	// Using -- to ensure we don't get filtered by current context
	args := []string{"add", "--", description, "+" + label, "+contact-" + state}
	
	cmd := exec.Command("dstask", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("creating task: %w (output: %s)", err, string(output))
	}

	return nil
}

// GetContactTasks retrieves all tasks for a contact by their label
func (b *Backend) GetContactTasks(label string) ([]tasks.Task, error) {
	if !b.enabled {
		return nil, fmt.Errorf("dstask not available")
	}

	if label == "" {
		return []tasks.Task{}, nil
	}

	// Ensure label starts with @
	if !strings.HasPrefix(label, "@") {
		label = "@" + label
	}

	// Use show-open to bypass context filtering and get all open tasks
	// Then we'll filter by our label tag
	args := []string{"show-open", "--json"}
	
	cmd := exec.Command("dstask", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("getting tasks: %w", err)
	}

	var allTasks []dstaskTask
	if len(output) > 0 && string(output) != "\n" {
		err = json.Unmarshal(output, &allTasks)
		if err != nil {
			return nil, fmt.Errorf("parsing task JSON: %w", err)
		}
	}

	// Filter tasks by label tag
	var filteredTasks []dstaskTask
	for _, task := range allTasks {
		for _, tag := range task.Tags {
			if tag == label {
				filteredTasks = append(filteredTasks, task)
				break
			}
		}
	}

	// Convert dstask tasks to generic tasks
	genericTasks := make([]tasks.Task, len(filteredTasks))
	for i, dtTask := range filteredTasks {
		genericTasks[i] = convertToGenericTask(dtTask)
	}

	return genericTasks, nil
}

// CompleteTask marks a task as completed
func (b *Backend) CompleteTask(taskID string, completionNote string) error {
	if !b.enabled {
		return fmt.Errorf("dstask not available")
	}

	// If there's a completion note, we'll add it to the task notes before completing
	if completionNote != "" {
		// Get the task first to append to existing notes
		showArgs := []string{"show-resolved", "--json", taskID}
		cmd := exec.Command("dstask", showArgs...)
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			var tasks []dstaskTask
			if err := json.Unmarshal(output, &tasks); err == nil && len(tasks) > 0 {
				// Append completion note to existing notes
				existingNotes := tasks[0].Notes
				if existingNotes != "" {
					existingNotes += "\n\n"
				}
				newNotes := existingNotes + "Completion: " + completionNote
				
				// Update notes
				notesArgs := []string{taskID, "note", newNotes}
				cmd = exec.Command("dstask", notesArgs...)
				cmd.Run() // Ignore errors, not critical
			}
		}
	}
	
	// Mark the task as done
	args := []string{taskID, "done"}
	
	cmd := exec.Command("dstask", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("completing task: %w (output: %s)", err, string(output))
	}

	return nil
}

// convertToGenericTask converts a dstask task to the generic Task type
func convertToGenericTask(dtTask dstaskTask) tasks.Task {
	task := tasks.Task{
		ID:          strconv.Itoa(dtTask.ID), // Use numeric ID as string
		Description: dtTask.Summary,
		Status:      mapDstaskStatus(dtTask.Status),
		Tags:        dtTask.Tags,
		Priority:    mapDstaskPriority(dtTask.Priority),
		Metadata: map[string]interface{}{
			"uuid": dtTask.UUID,
		},
	}

	// Parse timestamps
	if dtTask.Created != "" {
		if t, err := time.Parse(time.RFC3339, dtTask.Created); err == nil {
			task.Created = t
		}
	}
	
	if dtTask.Modified != "" {
		if t, err := time.Parse(time.RFC3339, dtTask.Modified); err == nil {
			task.Modified = t
		}
	}
	
	if dtTask.Due != "" {
		if t, err := time.Parse(time.RFC3339, dtTask.Due); err == nil {
			task.Due = &t
		}
	}

	return task
}

// mapDstaskStatus converts dstask status to generic status
func mapDstaskStatus(status string) string {
	// dstask uses: pending, active, done, etc.
	switch status {
	case "pending", "active":
		return "pending"
	case "done", "completed":
		return "completed"
	default:
		return status
	}
}

// mapDstaskPriority converts dstask priority to generic priority
func mapDstaskPriority(priority string) string {
	// dstask uses P0, P1, P2, P3 (P0 being highest)
	switch priority {
	case "P0":
		return "H"
	case "P1":
		return "M"
	case "P2", "P3":
		return "L"
	default:
		return priority
	}
}

// isDstaskAvailable checks if dstask is installed and configured
func isDstaskAvailable() bool {
	cmd := exec.Command("dstask", "help")
	err := cmd.Run()
	return err == nil
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

// Register the dstask backend
func init() {
	tasks.Register("dstask", func() tasks.Backend { return NewBackend() })
}
