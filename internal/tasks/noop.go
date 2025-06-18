package tasks

import "fmt"

// NoopBackend is a backend that does nothing, used when no task system is configured
type NoopBackend struct{}

// NewNoopBackend creates a new no-op backend
func NewNoopBackend() Backend {
	return &NoopBackend{}
}

// Name returns the backend identifier
func (n *NoopBackend) Name() string {
	return "noop"
}

// IsEnabled always returns false for the noop backend
func (n *NoopBackend) IsEnabled() bool {
	return false
}

// CreateContactTask returns an error indicating no backend is available
func (n *NoopBackend) CreateContactTask(contactName, state, label string) error {
	return fmt.Errorf("no task backend configured")
}

// GetContactTasks returns empty list
func (n *NoopBackend) GetContactTasks(label string) ([]Task, error) {
	return []Task{}, nil
}

// CompleteTask returns an error indicating no backend is available
func (n *NoopBackend) CompleteTask(taskID string, completionNote string) error {
	return fmt.Errorf("no task backend configured")
}

// Register the noop backend
func init() {
	Register("noop", func() Backend { return NewNoopBackend() })
}
