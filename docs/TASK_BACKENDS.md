# Task Backend Implementation Guide

This document describes the task backend interface and implementation requirements for contacts-tui.

## Overview

The task backend system provides an abstraction layer between the contacts-tui application and external task management systems. Backends are responsible for creating, retrieving, and completing tasks associated with contacts.

## Interface Definition

All task backends must implement the `Backend` interface defined in `internal/tasks/backend.go`:

```go
type Backend interface {
    Name() string
    IsEnabled() bool
    CreateContactTask(contactName, state, label string) error
    GetContactTasks(label string) ([]Task, error)
    CompleteTask(taskID string, completionNote string) error
}
```

### Method Specifications

#### Name() string
Returns the backend identifier used for configuration and selection.

- Must return a consistent, lowercase string
- Used in configuration files and command-line arguments
- Example: `"taskwarrior"`, `"dstask"`, `"things"`

#### IsEnabled() bool
Determines if the backend is available for use.

- Should check for required executables in PATH
- Should verify configuration validity
- Returns false if dependencies are missing
- Called during auto-detection phase

#### CreateContactTask(contactName, state, label string) error
Creates a new task for a contact state change.

Parameters:
- `contactName`: Full name of the contact
- `state`: New contact state (e.g., "ping", "followup", "invite")
- `label`: Contact label used for tagging (e.g., "@johnd")

Returns:
- `nil` on success
- Error with descriptive message on failure

Task description format convention:
- "ping" → "Ping [contactName]"
- "followup" → "Follow up with [contactName]"
- "invite" → "Send invitation to [contactName]"

#### GetContactTasks(label string) ([]Task, error)
Retrieves all tasks associated with a contact label.

Parameters:
- `label`: Contact label to search for

Returns:
- Slice of Task structs matching the label
- Empty slice if no tasks found
- Error if retrieval fails

#### CompleteTask(taskID string, completionNote string) error
Marks a task as completed.

Parameters:
- `taskID`: Backend-specific task identifier
- `completionNote`: Optional completion note (may be ignored by backend)

Returns:
- `nil` on success
- Error if task not found or completion fails

## Task Structure

Tasks are represented using the following structure:

```go
type Task struct {
    ID          string                     // Backend-specific identifier
    Description string                     
    Status      string                     // "pending", "completed", etc.
    Tags        []string                   
    Created     time.Time
    Modified    time.Time
    Due         *time.Time                 // Optional
    Priority    string                     // Optional
    Metadata    map[string]interface{}     // Backend-specific data
}
```

## Implementation Requirements

### Package Structure

Create a new package under `internal/tasks/[backend-name]/`:

```
internal/tasks/mybackend/
├── backend.go      # Backend implementation
└── backend_test.go # Unit tests
```

### Registration

Backends must register themselves during initialization:

```go
func init() {
    tasks.Register("mybackend", func() tasks.Backend { 
        return NewBackend() 
    })
}
```

### Configuration

If the backend requires configuration, extend the `TasksConfig` structure in `internal/config/config.go`:

```go
type TasksConfig struct {
    Backend   string           `toml:"backend"`
    MyBackend MyBackendConfig  `toml:"mybackend"`
}

type MyBackendConfig struct {
    APIKey    string `toml:"api_key"`
    ServerURL string `toml:"server_url"`
}
```

Configuration is loaded from `~/.config/contacts/config.toml`:

```toml
[tasks]
backend = "mybackend"

[tasks.mybackend]
api_key = "secret-key"
server_url = "https://api.example.com"
```

## Example Implementation

### Minimal Backend

```go
package mybackend

import (
    "fmt"
    "github.com/pdxmph/contacts-tui/internal/tasks"
)

type Backend struct {
    enabled bool
}

func NewBackend() tasks.Backend {
    return &Backend{
        enabled: checkDependencies(),
    }
}

func (b *Backend) Name() string {
    return "mybackend"
}

func (b *Backend) IsEnabled() bool {
    return b.enabled
}

func (b *Backend) CreateContactTask(contactName, state, label string) error {
    if !b.enabled {
        return fmt.Errorf("mybackend not available")
    }
    
    description := formatDescription(state, contactName)
    // Implementation specific task creation
    return nil
}

func (b *Backend) GetContactTasks(label string) ([]tasks.Task, error) {
    if !b.enabled {
        return nil, fmt.Errorf("mybackend not available")
    }
    
    // Implementation specific task retrieval
    return []tasks.Task{}, nil
}

func (b *Backend) CompleteTask(taskID string, completionNote string) error {
    if !b.enabled {
        return fmt.Errorf("mybackend not available")
    }
    
    // Implementation specific task completion
    return nil
}

func checkDependencies() bool {
    // Verify required executables or configuration
    return true
}

func formatDescription(state, contactName string) string {
    switch state {
    case "ping":
        return fmt.Sprintf("Ping %s", contactName)
    case "followup":
        return fmt.Sprintf("Follow up with %s", contactName)
    default:
        return fmt.Sprintf("%s: %s", state, contactName)
    }
}

func init() {
    tasks.Register("mybackend", func() tasks.Backend { 
        return NewBackend() 
    })
}
```

## Testing

Backends should include unit tests covering:

1. Dependency detection
2. Task creation with various states
3. Task retrieval and filtering
4. Task completion
5. Error handling

## Backend Selection

The application selects backends using the following precedence:

1. Explicitly configured backend in config.toml
2. Auto-detection in order: taskwarrior, dstask, things, noop
3. Fallback to noop if no backend available

## Error Handling

- Return descriptive errors that help users diagnose configuration issues
- Distinguish between temporary failures and configuration problems
- Avoid panics; return errors instead

## Tagging Conventions

- Contact labels should be preserved in task tags
- Some backends prefix tags (e.g., TaskWarrior uses "+@label")
- Handle tag format conversions appropriately
