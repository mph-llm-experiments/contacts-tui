package taskwarrior

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Task represents a TaskWarrior task
type Task struct {
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

// Client handles TaskWarrior CLI operations
type Client struct {
	enabled bool
}

// NewClient creates a new TaskWarrior client
func NewClient() *Client {
	return &Client{
		enabled: isTaskWarriorAvailable(),
	}
}

// IsEnabled returns whether TaskWarrior integration is available
func (c *Client) IsEnabled() bool {
	return c.enabled
}

// CreateContactTask creates a TaskWarrior task for a contact state change
func (c *Client) CreateContactTask(contactName, state, label string) error {
	if !c.enabled {
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
func (c *Client) GetContactTasks(label string) ([]Task, error) {
	if !c.enabled {
		return nil, fmt.Errorf("TaskWarrior not available")
	}

	if label == "" {
		return []Task{}, nil
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
			return []Task{}, nil
		}
		return nil, fmt.Errorf("getting tasks with command 'task %s': %w", strings.Join(args, " "), err)
	}

	var tasks []Task
	if len(output) > 0 {
		err = json.Unmarshal(output, &tasks)
		if err != nil {
			return nil, fmt.Errorf("parsing task JSON: %w", err)
		}
	}

	return tasks, nil
}

// CompleteTask marks a task as completed
func (c *Client) CompleteTask(taskUUID string) error {
	if !c.enabled {
		return fmt.Errorf("TaskWarrior not available")
	}

	args := []string{taskUUID, "done"}
	
	cmd := exec.Command("task", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("completing task: %w (output: %s)", err, string(output))
	}

	return nil
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
