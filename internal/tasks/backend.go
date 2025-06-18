package tasks

import "time"

// Task represents a task in any backend system
type Task struct {
	ID          string    // Backend-specific ID (could be int as string, UUID, etc.)
	Description string    
	Status      string    // pending, completed, etc.
	Tags        []string  
	Created     time.Time
	Modified    time.Time
	Due         *time.Time // Optional due date
	Priority    string     // Optional priority
	// Backend-specific data can be stored here
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Backend defines the interface that all task management backends must implement
type Backend interface {
	// Name returns the backend identifier (e.g., "taskwarrior", "dstask")
	Name() string
	
	// IsEnabled checks if the backend is available and properly configured
	IsEnabled() bool
	
	// CreateContactTask creates a task associated with a contact state change
	CreateContactTask(contactName, state, label string) error
	
	// GetContactTasks retrieves all tasks associated with a contact label
	GetContactTasks(label string) ([]Task, error)
	
	// CompleteTask marks a task as completed, optionally with a completion note
	CompleteTask(taskID string, completionNote string) error
}

// BackendFactory is a function that creates a new instance of a Backend
type BackendFactory func() Backend
