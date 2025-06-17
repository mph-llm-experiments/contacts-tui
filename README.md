# Contacts TUI

A fast, keyboard-driven terminal interface for contact management built with Go and Bubble Tea.

## Features

- **Keyboard-first interface** - Navigate and manage contacts without touching the mouse
- **Quick search** - Real-time filtering as you type
- **Contact states** - Track relationship status (ping, invite, followup, etc.)
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
