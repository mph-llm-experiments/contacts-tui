package taskwarrior

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
	
	"github.com/pdxmph/contacts-tui/internal/tasks"
)

// taskWarriorTask represents a TaskWarrior task in its native format
type taskWarriorTask struct {
	ID          int      `json:"id"`
	UUID        string   `json:"uuid"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Tags        []string `json:"tags"`
	Entry       string   `json:"entry"`
	Modified    string   `json:"modified"`
	Due         string   `json:"due,omitempty"`
	Priority    string   `json:"priority,omitempty"`
}

// Backend implements the tasks.Backend interface for TaskWarrior
type Backend struct {
	enabled bool
}

// NewBackend creates a new TaskWarrior backend
func NewBackend() tasks.Backend {
	return &Backend{
		enabled: isTaskWarriorAvailable(),
	}
}

// Name returns the backend identifier
func (b *Backend) Name() string {
	return "taskwarrior"
}

// IsEnabled returns whether TaskWarrior integration is available
func (b *Backend) IsEnabled() bool {
	return b.enabled
}

// CreateContactTask creates a TaskWarrior task for a contact state change
func (b *Backend) CreateContactTask(contactName, state, label string) error {
	if !b.enabled {
		return fmt.Errorf("TaskWarrior not available")
	}

	if label == "" {
		return fmt.Errorf("contact must have a label to create TaskWarrior task")
	}

	// Format task description based on state
	description := formatTaskDescription(state, contactName)
	
	// Ensure label starts with @
	if !strings.HasPrefix(label, "@") {
		label = "@" + label
	}

	// Create the task with label as tag (+ means add tag, @chrisb is the tag name)
	args := []string{"add", description, "+" + label}
	
	cmd := exec.Command("task", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("creating task: %w (output: %s)", err, string(output))
	}

	return nil
}

// GetContactTasks retrieves all tasks for a contact by their label
func (b *Backend) GetContactTasks(label string) ([]tasks.Task, error) {
	if !b.enabled {
		return nil, fmt.Errorf("TaskWarrior not available")
	}

	if label == "" {
		return []tasks.Task{}, nil
	}

	// Ensure label starts with @
	if !strings.HasPrefix(label, "@") {
		label = "@" + label
	}

	// Export tasks with the contact's label tag - filter goes before export command
	args := []string{"tag:" + label, "status:pending", "export"}
	
	cmd := exec.Command("task", args...)
	output, err := cmd.Output()
	if err != nil {
		// If no tasks found, return empty slice
		if strings.Contains(string(output), "No matching tasks") {
			return []tasks.Task{}, nil
		}
		return nil, fmt.Errorf("getting tasks with command 'task %s': %w", strings.Join(args, " "), err)
	}

	var twTasks []taskWarriorTask
	if len(output) > 0 {
		err = json.Unmarshal(output, &twTasks)
		if err != nil {
			return nil, fmt.Errorf("parsing task JSON: %w", err)
		}
	}

	// Convert TaskWarrior tasks to generic tasks
	genericTasks := make([]tasks.Task, len(twTasks))
	for i, twTask := range twTasks {
		genericTasks[i] = convertToGenericTask(twTask)
	}

	return genericTasks, nil
}

// CompleteTask marks a task as completed
func (b *Backend) CompleteTask(taskID string, completionNote string) error {
	if !b.enabled {
		return fmt.Errorf("TaskWarrior not available")
	}

	// If there's a completion note, add it as an annotation first
	if completionNote != "" {
		// Add annotation to the task
		annotateArgs := []string{taskID, "annotate", completionNote}
		cmd := exec.Command("task", annotateArgs...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("adding annotation: %w (output: %s)", err, string(output))
		}
	}
	
	// Now mark the task as done
	args := []string{taskID, "done"}
	
	cmd := exec.Command("task", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("completing task: %w (output: %s)", err, string(output))
	}

	return nil
}

// convertToGenericTask converts a TaskWarrior task to the generic Task type
func convertToGenericTask(twTask taskWarriorTask) tasks.Task {
	task := tasks.Task{
		ID:          twTask.UUID, // Use UUID as the ID for better stability
		Description: twTask.Description,
		Status:      twTask.Status,
		Tags:        twTask.Tags,
		Priority:    twTask.Priority,
		Metadata: map[string]interface{}{
			"taskwarrior_id": twTask.ID,
		},
	}

	// Parse timestamps
	if twTask.Entry != "" {
		if t, err := time.Parse(time.RFC3339, twTask.Entry); err == nil {
			task.Created = t
		}
	}
	
	if twTask.Modified != "" {
		if t, err := time.Parse(time.RFC3339, twTask.Modified); err == nil {
			task.Modified = t
		}
	}
	
	if twTask.Due != "" {
		if t, err := time.Parse(time.RFC3339, twTask.Due); err == nil {
			task.Due = &t
		}
	}

	return task
}

// isTaskWarriorAvailable checks if TaskWarrior is installed and configured
func isTaskWarriorAvailable() bool {
	cmd := exec.Command("task", "version")
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

// Register the TaskWarrior backend
func init() {
	tasks.Register("taskwarrior", func() tasks.Backend { return NewBackend() })
}
