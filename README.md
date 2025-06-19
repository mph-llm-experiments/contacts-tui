# Contacts TUI

A fast, keyboard-driven terminal interface for contact management built with Go and Bubble Tea.

## Features

- **Keyboard-first interface** - Navigate and manage contacts without touching the mouse
- **Quick search** - Real-time filtering as you type
- **Contact states** - Track relationship status (ping, invite, followup, etc.)
- **Task management integration** - Supports TaskWarrior, dstask, and Things 3 with auto-detection
- **Relationship types** - Organize contacts by type (work, family, network, etc.)
- **SQLite database** - Portable, single-file storage
- **Configurable** - Customize database location and task backend preferences

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

You can configure the database location and task backend preferences:

```toml
[database]
# Path to the SQLite database file
path = "~/Dropbox/contacts/contacts.db"

[tasks]
# Task backend: taskwarrior, dstask, things, or noop
# Leave empty for auto-detection
backend = "things"

[tasks.things]
# Required for Things 3 task creation
auth_token = "YOUR-AUTH-TOKEN"
```

See `config.example.toml` for a complete example configuration.

## Task Management Integration

Contacts TUI integrates with multiple task management systems to automatically create actionable tasks when you change contact states. This bridges your contact management with task management for better follow-through.

### Supported Backends

1. **[TaskWarrior](https://taskwarrior.org)** - Command-line task management
2. **[dstask](https://github.com/naggie/dstask)** - Distributed task tracker  
3. **[Things 3](https://culturedcode.com/things/)** - macOS/iOS task manager
4. **noop** - Disable task integration

### Auto-Detection

By default, Contacts TUI automatically detects and uses the first available task backend in this order:
1. TaskWarrior
2. dstask
3. Things 3
4. noop (if none available)

To specify a backend explicitly, add to your config file (`~/.config/contacts/config.toml`):

```toml
[tasks]
backend = "things"  # Options: taskwarrior, dstask, things, noop
```

### Features

- **Automatic task creation** - When you change a contact's state from "ok" to any action state (ping, followup, invite, etc.), a corresponding task is automatically created
- **Contact-based tagging** - Tasks are tagged with the contact's label (e.g., `+@johnd` or `@johnd` depending on backend)
- **Task management** - View, complete, and refresh tasks directly from the contacts interface
- **Smart descriptions** - Task descriptions are formatted based on the state change (e.g., "Ping John Doe", "Follow up with Jane Smith")

### Usage

#### Automatic Task Creation

1. Select a contact and press `s` to change state
2. Choose an action state like "ping" or "followup"  
3. If the contact has a label, a task is automatically created
4. If no label exists, you'll be prompted to add one

#### Managing Tasks

- Press `t` on any contact to view their tasks
- Use `j/k` to navigate tasks
- Press `Enter` or `Space` to complete a task
- Press `r` to refresh the task list
- Press `Esc` to return to contacts

### Backend-Specific Configuration

#### TaskWarrior

Prerequisites:
- Install TaskWarrior from [taskwarrior.org](https://taskwarrior.org)
- No additional configuration needed

Task format:
```bash
task add "Ping John Doe" +@johnd
```

#### dstask

Prerequisites:
- Install dstask from [github.com/naggie/dstask](https://github.com/naggie/dstask)
- Initialize with `dstask help`

Task format:
```bash
dstask add "Ping John Doe" +@johnd
```

#### Things 3

Prerequisites:
- Things 3 must be installed (macOS only)
- Requires auth token for task creation

Configuration:
```toml
[tasks]
backend = "things"

[tasks.things]
auth_token = "YOUR-AUTH-TOKEN"  # Required for task creation
default_list = ""               # Optional: default list for tasks
tag_template = ""               # Optional: custom tag template
```

To get your Things auth token:
1. Open Things 3 → Preferences → General
2. Enable "Enable Things URLs"
3. Click "Manage" next to "Enable Things URLs"
4. Copy your auth token

Task features:
- Uses fast JXA (JavaScript for Automation) for querying
- Creates tasks with proper tags in Things format
- Keeps Things in background when creating tasks
- Shows completion confirmation messages

### Examples

**Contact State Change:**
```
Contact: John Doe (@johnd)
State: ok → ping
Result: Creates task "Ping John Doe" tagged with @johnd
```

**Task Management View:**
```
Press 't' on John Doe:
┌─ Tasks (Things) ─────────────────────────────┐
│ Contact: John Doe (@johnd)                   │
│                                              │
│ Tasks (2):                                   │
│                                              │
│ ▶ Ping John Doe                             │
│   Follow up about project                    │
│                                              │
│ j/k: navigate • Enter: complete • Esc: back  │
└──────────────────────────────────────────────┘
```

### Troubleshooting

#### General
- **"Contact must have a label"** - Add a label to the contact (e.g., `@johnd`) or you'll be prompted to create one
- **Tasks not appearing** - Ensure the contact's label matches the backend's tag format

#### TaskWarrior
- **"TaskWarrior not available"** - Install TaskWarrior and ensure `task` is in your PATH

#### dstask
- **"dstask not available"** - Install dstask and ensure `dstask` is in your PATH
- **No tasks showing** - Run `dstask sync` to ensure the database is initialized

#### Things 3
- **"Things not available"** - Things 3 must be installed (macOS only)
- **"Things: auth token required"** - Add your auth token to the config file
- **Task creation fails** - Ensure "Enable Things URLs" is turned on in Things preferences

The task integration makes contact states genuinely actionable, ensuring follow-up tasks don't fall through the cracks across your preferred task management system.

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
