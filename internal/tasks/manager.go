package tasks

import (
	"fmt"
)

// Manager handles task backend selection and operations
type Manager struct {
	backend Backend
}

// NewManager creates a new task manager with the specified backend
// If backendName is empty, it tries common backends in order of preference
func NewManager(backendName string) (*Manager, error) {
	var backend Backend
	var err error
	
	if backendName != "" {
		// Use specified backend
		backend, err = CreateBackend(backendName)
		if err != nil {
			return nil, fmt.Errorf("creating backend %s: %w", backendName, err)
		}
	} else {
		// Try backends in order of preference
		backendPreference := []string{"taskwarrior", "dstask", "noop"}
		
		for _, name := range backendPreference {
			b, err := CreateBackend(name)
			if err != nil {
				continue
			}
			
			if b.IsEnabled() {
				backend = b
				break
			}
		}
		
		// If no backend is enabled, use noop
		if backend == nil || !backend.IsEnabled() {
			backend, _ = CreateBackend("noop")
		}
	}
	
	return &Manager{backend: backend}, nil
}

// Backend returns the current backend
func (m *Manager) Backend() Backend {
	return m.backend
}

// Name returns the name of the current backend
func (m *Manager) Name() string {
	return m.backend.Name()
}

// IsEnabled returns whether the current backend is enabled
func (m *Manager) IsEnabled() bool {
	return m.backend.IsEnabled()
}
