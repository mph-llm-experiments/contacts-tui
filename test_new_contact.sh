#!/bin/bash
echo "Testing the new contact creation feature..."
echo ""
echo "The '+' or 'N' key should open a form to create a new contact."
echo "You should be able to:"
echo "  - Enter name (required)"
echo "  - Tab through fields: email, phone, company, relationship type, notes, label"
echo "  - Use arrow keys to select relationship type"
echo "  - Press Enter to save"
echo "  - Press Esc to cancel"
echo ""
echo "Press Enter to start the TUI..."
read

cd /Users/mph/code/contacts-tui
./contacts-tui
