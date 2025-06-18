package tasks

import (
	"fmt"
	"sync"
)

// Registry manages available task backends
type Registry struct {
	mu       sync.RWMutex
	backends map[string]BackendFactory
}

// NewRegistry creates a new backend registry
func NewRegistry() *Registry {
	return &Registry{
		backends: make(map[string]BackendFactory),
	}
}

// Register adds a new backend factory to the registry
func (r *Registry) Register(name string, factory BackendFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.backends[name]; exists {
		return fmt.Errorf("backend %s already registered", name)
	}
	
	r.backends[name] = factory
	return nil
}

// Create instantiates a backend by name
func (r *Registry) Create(name string) (Backend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	factory, exists := r.backends[name]
	if !exists {
		return nil, fmt.Errorf("backend %s not registered", name)
	}
	
	return factory(), nil
}

// List returns all registered backend names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	names := make([]string, 0, len(r.backends))
	for name := range r.backends {
		names = append(names, name)
	}
	return names
}

// Global registry instance
var defaultRegistry = NewRegistry()

// Register adds a backend to the global registry
func Register(name string, factory BackendFactory) error {
	return defaultRegistry.Register(name, factory)
}

// CreateBackend creates a backend from the global registry
func CreateBackend(name string) (Backend, error) {
	return defaultRegistry.Create(name)
}

// ListBackends returns all registered backend names from the global registry
func ListBackends() []string {
	return defaultRegistry.List()
}
