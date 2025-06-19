# Things Backend for Contacts TUI

This backend integrates with [Things 3](https://culturedcode.com/things/) on macOS to create and manage tasks associated with contact state changes.

## Requirements

- macOS (Things is only available on Apple platforms)
- Things 3 installed
- Things auth token (for task creation)

## Configuration

### 1. Get your Things auth token

1. Open Things 3
2. Go to `Things > Preferences > General`
3. Enable "Enable Things URLs" 
4. Click "Manage" next to the checkbox
5. Copy your auth token

### 2. Configure contacts-tui

Add to your `~/.config/contacts/config.toml`:

```toml
[tasks]
backend = "things"  # Or leave empty for auto-detection

[tasks.things]
auth_token = "YOUR_AUTH_TOKEN_HERE"
```

## Features

- **Task Creation**: When you change a contact's state (ping, followup, etc.), a task is automatically created in Things
- **Task Viewing**: Press `t` on a contact to see all their associated tasks
- **Task Completion**: Select a task and mark it complete with optional notes
- **Tag-based Association**: Tasks are tagged with the contact's label (e.g., `@jeffv`)

## How It Works

The Things backend uses:
- **JavaScript for Automation (JXA)** for querying and updating tasks
- **Things URL scheme** for creating tasks (requires auth token)
- **Tag-based filtering** to associate tasks with contacts

Each task is tagged with:
- The contact's label (e.g., `@jeffv`)
- The contact state (e.g., `contact-ping`, `contact-followup`)

## Limitations

- Only available on macOS
- Cannot specify which Things list/project tasks are created in (uses inbox by default)
- Requires auth token for task creation
- Task updates are limited to what's available through JXA

## Troubleshooting

### "Things not available" error
- Ensure Things 3 is installed in `/Applications/Things3.app`
- Check that you're running on macOS

### "Things auth token not configured" error
- Follow the configuration steps above to get and set your auth token

### Tasks not appearing
- Ensure the contact has a label (tasks won't be created without one)
- Check Things to see if the task was created but perhaps filtered out of your current view
