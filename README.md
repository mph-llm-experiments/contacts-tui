# Contacts TUI

A fast, keyboard-driven terminal interface for contact management built with Go and Bubble Tea.

## Features

- **Keyboard-first interface** - Navigate and manage contacts without touching the mouse
- **Quick search** - Real-time filtering as you type
- **Contact states** - Track relationship status (ping, invite, followup, etc.)
- **TaskWarrior integration** - Automatically create and manage tasks when contact states change
- **Relationship types** - Organize contacts by type (work, family, network, etc.)
- **SQLite database** - Portable, single-file storage
- **Configurable** - Customize database location for syncing across devices

## Installation

### From source

```bash
go install github.com/pdxmph/contacts-tui@latest
```

### Build locally

```bash
git clone https://github.com/pdxmph/contacts-tui.git
cd contacts-tui
go build
./contacts-tui
```

## Usage

### Quick Start

```bash
# Run with default configuration
contacts-tui

# Generate a configuration file
contacts-tui -write-config

# Show current configuration
contacts-tui -show-config
```

### Key Bindings

- `↑/↓` or `j/k` - Navigate contacts
- `/` - Search contacts
- `+` or `n` - Add new contact
- `Enter` - View/edit contact details
- `s` - Change contact state (ping, followup, etc.)
- `t` - View/manage TaskWarrior tasks for contact
- `Tab` - Switch between list and details
- `Esc` - Cancel/go back
- `q` - Quit

## Configuration

The application looks for configuration at `~/.config/contacts/config.toml`. If no configuration file exists, it will use default values.

### Command-line Options

- `contacts-tui -write-config` - Generate a default configuration file
- `contacts-tui -show-config` - Display the current configuration
- `contacts-tui -init` - Initialize database and configuration for first-time setup
- `contacts-tui --database <path>` - Use a specific database file (overrides config)
- `contacts-tui --create-fixtures` - Create a test database with sample data
- `contacts-tui --fixtures-path <path>` - Specify path for fixtures database

### Testing with Fixtures

For testing or demonstration purposes, you can create a fixtures database with realistic sample data:

```bash
# Create fixtures database with default name (fixtures.db)
contacts-tui --create-fixtures

# Create fixtures database at custom location
contacts-tui --create-fixtures --fixtures-path test-data.db

# Use the fixtures database
contacts-tui --database fixtures.db
```

The fixtures database includes:
- Contacts across all relationship types (work, family, network, social, etc.)
- Various contact states and interaction histories
- Sample data for testing different features

### Database Location

You can configure the database location to support sharing across systems via network shares:

```toml
[database]
# Path to the SQLite database file
path = "~/Dropbox/contacts/contacts.db"
```

See `config.example.toml` for a complete example configuration.

## TaskWarrior Integration

Contacts TUI integrates with [TaskWarrior](https://taskwarrior.org) to automatically create actionable tasks when you change contact states. This bridges your contact management with task management for better follow-through.

### Features

- **Automatic task creation** - When you change a contact's state from "ok" to any action state (ping, followup, invite, etc.), a corresponding TaskWarrior task is automatically created
- **Contact-based tagging** - Tasks are tagged with the contact's label (e.g., `+@johnd`) for easy filtering
- **Task management** - View, complete, and refresh tasks directly from the contacts interface
- **Smart descriptions** - Task descriptions are formatted based on the state change (e.g., "Ping John Doe", "Follow up with Jane Smith")

### Prerequisites

1. **TaskWarrior installed** - Install from [taskwarrior.org](https://taskwarrior.org) or your package manager
2. **Contact labels** - Contacts need labels (e.g., `@johnd`) to create tagged tasks

### Usage

#### Automatic Task Creation

1. Select a contact and press `s` to change state
2. Choose an action state like "ping" or "followup"  
3. If the contact has a label, a TaskWarrior task is automatically created
4. If no label exists, you'll be prompted to add one

#### Managing Tasks

- Press `t` on any contact to view their TaskWarrior tasks
- Use `j/k` to navigate tasks
- Press `Enter` or `Space` to complete a task
- Press `r` to refresh the task list
- Press `Esc` to return to contacts

#### TaskWarrior Commands

The integration uses standard TaskWarrior commands:

```bash
# Create task (automatic)
task add "Ping John Doe" +@johnd

# View contact's tasks
task tag:@johnd list

# Complete task
task <id> done
```

### Examples

**Contact State Change:**
```
Contact: John Doe (@johnd)
State: ok → ping
Result: Creates task "Ping John Doe" tagged with +@johnd
```

**Task Management:**
```
Press 't' on John Doe:
┌─ TaskWarrior Tasks ─────────────────────────┐
│ Contact: John Doe (@johnd)                  │
│                                             │
│ Tasks (2):                                  │
│                                             │
│ ▶ Ping John Doe                            │
│   Follow up about project                   │
│                                             │
│ j/k: navigate • Enter: complete • Esc: back │
└─────────────────────────────────────────────┘
```

### Troubleshooting

- **"TaskWarrior not available"** - Install TaskWarrior and ensure it's in your PATH
- **"Contact must have a label"** - Add a label to the contact (e.g., `@johnd`) or you'll be prompted to create one
- **Tasks not appearing** - Ensure the contact's label matches the TaskWarrior tag format

The TaskWarrior integration makes contact states genuinely actionable, ensuring follow-up tasks don't fall through the cracks.

## Building

Requirements:
- Go 1.21 or later

```bash
make build    # Build for current platform
make test     # Run tests
make clean    # Clean build artifacts
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
