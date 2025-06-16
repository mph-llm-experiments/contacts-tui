# Contacts TUI

Keyboard-driven terminal interface for contact management using Go and Bubble Tea.

## Configuration

The application looks for configuration at `~/.config/contacts/config.toml`. If no configuration file exists, it will use default values.

### Command-line Options

- `contacts-tui -write-config` - Generate a default configuration file
- `contacts-tui -show-config` - Display the current configuration

### Database Location

You can configure the database location to support sharing across systems via network shares:

```toml
[database]
# Path to the SQLite database file
path = "~/Dropbox/contacts/contacts.db"
```

See `config.example.toml` for a complete example configuration.
