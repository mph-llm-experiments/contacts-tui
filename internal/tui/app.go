package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pdxmph/contacts-tui/internal/db"
	"github.com/pdxmph/contacts-tui/internal/taskwarrior"
)

// Model represents the main application state
type Model struct {
	db         *db.DB
	contacts   []db.Contact
	selected   int
	width      int
	height     int
	filterMode bool
	stateMode  bool
	stateSelected int
	noteMode   bool
	noteInput  textarea.Model
	noteType   int
	filter     textinput.Model
	err        error
	successMsg string
	
	// Smart filters
	stateFilter   bool // Show only non-ok states
	overdueFilter bool // Show only overdue contacts
	typeFilter    string // Filter by relationship type
	showArchived  bool // Show archived contacts
	
	// Relationship type selection mode
	typeFilterMode bool
	typeSelected   int
	
	// Edit mode
	editMode       bool
	editField      int // Which field is being edited
	editInputs     []textinput.Model
	editRelTypeIdx int // Selected relationship type in edit mode
	
	// Bump confirmation mode
	bumpConfirmMode bool
	bumpContactID   int
	
	// Delete confirmation mode
	deleteConfirmMode bool
	deleteContactID   int
	deleteContactName string
	
	// Help overlay mode
	showHelp bool
	helpScrollOffset int
	
	// New contact mode
	newContactMode   bool
	newContactField  int // Which field is being edited
	newContactInputs []textinput.Model
	newContactRelTypeIdx int // Selected relationship type for new contact
	
	// Interaction editing mode
	interactionEditMode bool
	selectedInteraction int // Index of selected interaction in the list
	interactions        []db.Log // Current contact's interactions
	interactionEditInput textarea.Model
	interactionEditType  int // Selected interaction type
	interactionDeleteConfirm bool
	interactionToDelete int // ID of interaction to delete
	
	// Contact style mode
	styleMode bool
	styleSelected int
	styleContactID int
	customFreqInput textinput.Model
	customFreqMode bool
	
	// TaskWarrior integration
	taskwarriorClient *taskwarrior.Client
	taskMode          bool // Task view mode
	tasks             []taskwarrior.Task
	selectedTask      int
	
	// Label prompt mode (when creating tasks for contacts without labels)
	labelPromptMode bool
	labelPromptInput textinput.Model
	labelPromptContactID int
	labelPromptNewState string
	
	// Menu hotkeys
	stateHotkeys []MenuHotkey
	interactionHotkeys []MenuHotkey
	relationshipHotkeys []MenuHotkey
}

// MenuHotkey represents a menu item with its assigned hotkey
type MenuHotkey struct {
	Key   rune
	Label string
	Value string
}

// assignHotkeys assigns unique hotkeys to menu items
func assignHotkeys(items []string) []MenuHotkey {
	hotkeys := make([]MenuHotkey, len(items))
	usedKeys := make(map[rune]bool)
	
	// First pass: try first character
	for i, item := range items {
		if len(item) > 0 {
			firstChar := rune(item[0])
			if firstChar >= 'a' && firstChar <= 'z' && !usedKeys[firstChar] {
				hotkeys[i] = MenuHotkey{Key: firstChar, Label: item, Value: item}
				usedKeys[firstChar] = true
			}
		}
	}
	
	// Second pass: find unique characters for conflicts
	for i, item := range items {
		if hotkeys[i].Key == 0 && len(item) > 0 {
			// Try each character in the word
			for _, char := range item {
				if char >= 'a' && char <= 'z' && !usedKeys[char] {
					hotkeys[i] = MenuHotkey{Key: char, Label: item, Value: item}
					usedKeys[char] = true
					break
				}
			}
		}
	}
	
	// Third pass: if still no key, use numbers
	nextNum := '1'
	for i := range items {
		if hotkeys[i].Key == 0 {
			if nextNum <= '9' {
				hotkeys[i] = MenuHotkey{Key: nextNum, Label: items[i], Value: items[i]}
				nextNum++
			} else {
				// Fallback to first available letter
				for c := 'a'; c <= 'z'; c++ {
					if !usedKeys[c] {
						hotkeys[i] = MenuHotkey{Key: c, Label: items[i], Value: items[i]}
						usedKeys[c] = true
						break
					}
				}
			}
		}
	}
	
	return hotkeys
}

// Available contact states
var ContactStates = []string{
	"ping",
	"invite", 
	"write",
	"followup",
	"sked",
	"notes",
	"scheduled",
	"timeout",
	"ok",
}

// Available relationship types
var RelationshipTypes = []string{
	"all", // Special case to show all
	"work",
	"close", 
	"family",
	"network",
	"social",
	"providers",
	"recruiters",
}

// Available interaction types
var InteractionTypes = []string{
	"manual",
	"email",
	"call",
	"meeting",
	"in-person",
	"social-media",
	"text",
}

// Available contact styles
var ContactStyles = []string{
	"periodic",
	"ambient",
	"triggered",
}

// Edit field indices
const (
	EditFieldName = iota
	EditFieldEmail
	EditFieldPhone
	EditFieldCompany
	EditFieldRelType
	EditFieldNotes
	EditFieldLabel
	EditFieldCount // Total number of fields
)

// Styles
var (
	// Contact list selection - no background, just bold and bright
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214")) // Orange
	
	// Note type selector style - no background, just bold brackets
	noteTypeSelectorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214")) // Orange
	
	overdueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
	
	stateStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")) // Orange for states
	
	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
	
	borderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))
	
	dimmedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238")) // Dim gray for archived
	
	greenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("34")) // Green for ambient
	
	yellowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")) // Yellow for triggered
)
// New creates a new application model
func New(database *db.DB) (*Model, error) {
	// Load initial contacts
	contacts, err := database.ListContacts()
	if err != nil {
		return nil, fmt.Errorf("loading contacts: %w", err)
	}
	
	// Setup filter input
	ti := textinput.New()
	ti.Placeholder = "Filter contacts..."
	ti.Width = 30 // Generous default width
	ti.CharLimit = 50
	ti.Prompt = "> " // Explicitly set the prompt
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("230"))
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	
	// Setup note input
	ta := textarea.New()
	ta.Placeholder = "Enter note..."
	ta.SetHeight(4)
	ta.SetWidth(50)
	ta.CharLimit = 500
	ta.ShowLineNumbers = false
	
	// Setup edit inputs
	editInputs := make([]textinput.Model, EditFieldCount)
	for i := range editInputs {
		editInputs[i] = textinput.New()
		editInputs[i].Width = 40
		editInputs[i].CharLimit = 200
		
		switch i {
		case EditFieldName:
			editInputs[i].Placeholder = "Name"
		case EditFieldEmail:
			editInputs[i].Placeholder = "Email"
		case EditFieldPhone:
			editInputs[i].Placeholder = "Phone"
		case EditFieldCompany:
			editInputs[i].Placeholder = "Company"
		case EditFieldNotes:
			editInputs[i].Placeholder = "Notes"
		case EditFieldLabel:
			editInputs[i].Placeholder = "Label (e.g. @john)"
		}
	}
	
	// Setup new contact inputs (same as edit inputs)
	newContactInputs := make([]textinput.Model, EditFieldCount)
	for i := range newContactInputs {
		newContactInputs[i] = textinput.New()
		newContactInputs[i].Width = 40
		newContactInputs[i].CharLimit = 200
		
		switch i {
		case EditFieldName:
			newContactInputs[i].Placeholder = "Name (required)"
		case EditFieldEmail:
			newContactInputs[i].Placeholder = "Email"
		case EditFieldPhone:
			newContactInputs[i].Placeholder = "Phone"
		case EditFieldCompany:
			newContactInputs[i].Placeholder = "Company"
		case EditFieldNotes:
			newContactInputs[i].Placeholder = "Notes"
		case EditFieldLabel:
			newContactInputs[i].Placeholder = "Label (e.g. @john)"
		}
	}
	
	// Setup interaction edit textarea
	interactionTA := textarea.New()
	interactionTA.Placeholder = "Edit interaction..."
	interactionTA.SetHeight(4)
	interactionTA.SetWidth(50)
	interactionTA.CharLimit = 500
	interactionTA.ShowLineNumbers = false
	
	// Setup custom frequency input
	customFreqInput := textinput.New()
	customFreqInput.Placeholder = "Days (e.g. 30)"
	customFreqInput.Width = 20
	customFreqInput.CharLimit = 4
	
	// Setup label prompt input
	labelPromptInput := textinput.New()
	labelPromptInput.Placeholder = "e.g. @johnd"
	labelPromptInput.Width = 30
	labelPromptInput.CharLimit = 50
	
	return &Model{
		db:         database,
		contacts:   contacts,
		filter:     ti,
		noteInput:  ta,
		editInputs: editInputs,
		newContactInputs: newContactInputs,
		interactionEditInput: interactionTA,
		customFreqInput: customFreqInput,
		labelPromptInput: labelPromptInput,
		taskwarriorClient: taskwarrior.NewClient(),
		stateHotkeys: assignHotkeys(ContactStates),
		interactionHotkeys: assignHotkeys(InteractionTypes),
		relationshipHotkeys: assignHotkeys(RelationshipTypes),
	}, nil
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update filter width when window size changes
		if m.width > 0 {
			listWidth := m.width / 3
			m.filter.Width = listWidth - 4 // account for borders and padding
		}
		return m, nil
		
	case tea.KeyMsg:
		// Relationship type filter mode handling
		if m.typeFilterMode {
			switch msg.String() {
			case "esc":
				m.typeFilterMode = false
				m.typeSelected = 0
				return m, nil
			case "enter":
				// Set the type filter
				selected := RelationshipTypes[m.typeSelected]
				if selected == "all" {
					m.typeFilter = ""
				} else {
					m.typeFilter = selected
				}
				m.typeFilterMode = false
				m.typeSelected = 0
				m.selected = m.ensureValidSelection()
				return m, nil
			case "j", "down":
				if m.typeSelected < len(RelationshipTypes)-1 {
					m.typeSelected++
				}
			case "k", "up":
				if m.typeSelected > 0 {
					m.typeSelected--
				}
			default:
				// Check if it's a hotkey
				if len(msg.String()) == 1 {
					char := rune(msg.String()[0])
					for i, hotkey := range m.relationshipHotkeys {
						if hotkey.Key == char {
							// Apply the filter immediately
							selected := RelationshipTypes[i]
							if selected == "all" {
								m.typeFilter = ""
							} else {
								m.typeFilter = selected
							}
							m.typeFilterMode = false
							m.typeSelected = 0
							m.selected = m.ensureValidSelection()
							return m, nil
						}
					}
				}
			}
			return m, nil
		}
		
		// Bump confirmation mode handling
		if m.bumpConfirmMode {
			switch msg.String() {
			case "y", "Y":
				// Perform the bump
				err := m.db.BumpContact(m.bumpContactID)
				if err != nil {
					m.err = err
				} else {
					// Reload contacts to show updated state
					if newContacts, err := m.db.ListContacts(); err == nil {
						m.contacts = newContacts
						m.selected = m.ensureValidSelection()
					}
				}
				m.bumpConfirmMode = false
				m.bumpContactID = 0
				return m, nil
			default:
				// Any other key cancels
				m.bumpConfirmMode = false
				m.bumpContactID = 0
				return m, nil
			}
		}
		
		// Delete confirmation mode handling
		if m.deleteConfirmMode {
			switch msg.String() {
			case "y", "Y":
				// Perform the delete
				err := m.db.DeleteContact(m.deleteContactID)
				if err != nil {
					m.err = err
				} else {
					// Reload contacts to show updated state
					if newContacts, err := m.db.ListContacts(); err == nil {
						m.contacts = newContacts
						m.selected = m.ensureValidSelection()
					}
				}
				m.deleteConfirmMode = false
				m.deleteContactID = 0
				m.deleteContactName = ""
				return m, nil
			default:
				// Any other key cancels
				m.deleteConfirmMode = false
				m.deleteContactID = 0
				m.deleteContactName = ""
				return m, nil
			}
		}
		
		// Task mode handling
		if m.taskMode {
			switch msg.String() {
			case "esc":
				// Exit task mode
				m.taskMode = false
				m.tasks = nil
				m.selectedTask = 0
				return m, nil
				
			case "j", "down":
				// Navigate down in task list
				if len(m.tasks) > 0 && m.selectedTask < len(m.tasks)-1 {
					m.selectedTask++
				}
				return m, nil
				
			case "k", "up":
				// Navigate up in task list
				if m.selectedTask > 0 {
					m.selectedTask--
				}
				return m, nil
				
			case "enter", " ":
				// Complete selected task
				if len(m.tasks) > 0 && m.selectedTask < len(m.tasks) {
					task := m.tasks[m.selectedTask]
					err := m.taskwarriorClient.CompleteTask(task.UUID)
					if err != nil {
						m.err = fmt.Errorf("completing task: %w", err)
						return m, nil
					}
					
					// Show confirmation message
					m.successMsg = fmt.Sprintf("✓ Completed: %s", task.Description)
					
					// Refresh task list
					contacts := m.filteredContacts()
					if len(contacts) > 0 && m.selected < len(contacts) {
						contact := contacts[m.selected]
						if contact.Label.Valid && contact.Label.String != "" {
							if tasks, err := m.taskwarriorClient.GetContactTasks(contact.Label.String); err == nil {
								m.tasks = tasks
								// Adjust selected task if we're at the end
								if m.selectedTask >= len(m.tasks) && len(m.tasks) > 0 {
									m.selectedTask = len(m.tasks) - 1
								} else if len(m.tasks) == 0 {
									m.selectedTask = 0
								}
							}
						}
					}
				}
				return m, nil
				
			case "r":
				// Refresh task list
				contacts := m.filteredContacts()
				if len(contacts) > 0 && m.selected < len(contacts) {
					contact := contacts[m.selected]
					if contact.Label.Valid && contact.Label.String != "" {
						if tasks, err := m.taskwarriorClient.GetContactTasks(contact.Label.String); err == nil {
							m.tasks = tasks
							m.selectedTask = 0
						} else {
							m.err = fmt.Errorf("refreshing tasks: %w", err)
						}
					}
				}
				return m, nil
			}
			return m, nil
		}
		
		// Label prompt mode handling
		if m.labelPromptMode {
			switch msg.String() {
			case "esc":
				// Cancel label prompt
				m.labelPromptMode = false
				m.labelPromptInput.Blur()
				m.labelPromptContactID = 0
				m.labelPromptNewState = ""
				return m, nil
				
			case "enter":
				// Save label and create task
				newLabel := strings.TrimSpace(m.labelPromptInput.Value())
				if newLabel == "" {
					m.err = fmt.Errorf("label cannot be empty")
					return m, nil
				}
				
				// Ensure label starts with @
				if !strings.HasPrefix(newLabel, "@") {
					newLabel = "@" + newLabel
				}
				
				// Check for uniqueness
				for _, contact := range m.contacts {
					if contact.Label.Valid && contact.Label.String == newLabel {
						m.err = fmt.Errorf("label %s already exists", newLabel)
						return m, nil
					}
				}
				
				// Update contact with new label
				err := m.db.UpdateContactLabel(m.labelPromptContactID, newLabel)
				if err != nil {
					m.err = fmt.Errorf("failed to update label: %w", err)
					return m, nil
				}
				
				// Create TaskWarrior task with new label
				if contact, err := m.db.GetContact(m.labelPromptContactID); err == nil {
					taskErr := m.taskwarriorClient.CreateContactTask(
						contact.Name,
						m.labelPromptNewState,
						newLabel,
					)
					if taskErr != nil {
						m.err = fmt.Errorf("label added but task creation failed: %w", taskErr)
					} else {
						m.successMsg = fmt.Sprintf("✓ Added label %s and created task", newLabel)
					}
				}
				
				// Reload contacts and exit label prompt mode
				if newContacts, err := m.db.ListContacts(); err == nil {
					m.contacts = newContacts
					m.selected = m.ensureValidSelection()
				}
				
				m.labelPromptMode = false
				m.labelPromptInput.Blur()
				m.labelPromptContactID = 0
				m.labelPromptNewState = ""
				return m, nil
			default:
				// Handle input
				var cmd tea.Cmd
				m.labelPromptInput, cmd = m.labelPromptInput.Update(msg)
				return m, cmd
			}
		}
		
		// New contact mode handling
		if m.newContactMode {
			switch msg.String() {
			case "esc":
				// Cancel new contact creation
				m.newContactMode = false
				m.newContactField = 0
				for i := range m.newContactInputs {
					m.newContactInputs[i].Blur()
				}
				return m, nil
				
			case "enter":
				// Save new contact
				if strings.TrimSpace(m.newContactInputs[EditFieldName].Value()) == "" {
					// Name is required
					m.err = fmt.Errorf("name is required")
					return m, nil
				}
				
				// Create new contact
				newContact := db.Contact{
					Name:             strings.TrimSpace(m.newContactInputs[EditFieldName].Value()),
					Email:            db.NewNullString(strings.TrimSpace(m.newContactInputs[EditFieldEmail].Value())),
					Phone:            db.NewNullString(strings.TrimSpace(m.newContactInputs[EditFieldPhone].Value())),
					Company:          db.NewNullString(strings.TrimSpace(m.newContactInputs[EditFieldCompany].Value())),
					RelationshipType: RelationshipTypes[m.newContactRelTypeIdx+1], // Skip "all"
					Notes:            db.NewNullString(strings.TrimSpace(m.newContactInputs[EditFieldNotes].Value())),
					Label:            db.NewNullString(strings.TrimSpace(m.newContactInputs[EditFieldLabel].Value())),
					State:            db.NewNullString("ok"), // Default state
				}
				
				// Save to database
				_, err := m.db.AddContact(newContact)
				if err != nil {
					m.err = err
					return m, nil
				}
				
				// Exit new contact mode
				m.newContactMode = false
				m.newContactField = 0
				for i := range m.newContactInputs {
					m.newContactInputs[i].Blur()
				}
				
				// Reload contacts
				if newContacts, err := m.db.ListContacts(); err == nil {
					m.contacts = newContacts
					// Try to select the newly created contact
					for i, c := range m.filteredContacts() {
						if c.Name == newContact.Name {
							m.selected = i
							break
						}
					}
				}
				
				return m, nil
				
			case "tab":
				// Move to next field
				m.newContactInputs[m.newContactField].Blur()
				
				if m.newContactField == EditFieldRelType {
					// Skip to notes field after relationship type
					m.newContactField = EditFieldNotes
				} else if m.newContactField < EditFieldCount-1 {
					m.newContactField++
					if m.newContactField == EditFieldRelType {
						m.newContactField++ // Skip relationship type field in tab order
					}
				} else {
					m.newContactField = 0
				}
				
				if m.newContactField < len(m.newContactInputs) && m.newContactField != EditFieldRelType {
					m.newContactInputs[m.newContactField].Focus()
					return m, textinput.Blink
				}
				return m, nil
				
			case "shift+tab":
				// Move to previous field
				m.newContactInputs[m.newContactField].Blur()
				
				if m.newContactField == EditFieldNotes {
					// Skip back to relationship type selector
					m.newContactField = EditFieldRelType
				} else if m.newContactField > 0 {
					m.newContactField--
					if m.newContactField == EditFieldRelType {
						m.newContactField-- // Skip relationship type field in tab order
					}
				} else {
					m.newContactField = EditFieldCount - 1
				}
				
				if m.newContactField < len(m.newContactInputs) && m.newContactField != EditFieldRelType {
					m.newContactInputs[m.newContactField].Focus()
					return m, textinput.Blink
				}
				return m, nil
				
			case "left", "h":
				if m.newContactField == EditFieldRelType {
					if m.newContactRelTypeIdx > 0 {
						m.newContactRelTypeIdx--
					}
					return m, nil
				}
				// Pass through to text input for other fields
				
			case "right", "l":
				if m.newContactField == EditFieldRelType {
					if m.newContactRelTypeIdx < len(RelationshipTypes)-2 {
						m.newContactRelTypeIdx++
					}
					return m, nil
				}
				// Pass through to text input for other fields
				
			case "up", "k":
				if m.newContactField == EditFieldRelType {
					// Move to previous field when pressing up on relationship type
					m.newContactField = EditFieldCompany
					m.newContactInputs[m.newContactField].Focus()
					return m, textinput.Blink
				}
				// Pass through to text input for other fields
				
			case "down", "j":
				if m.newContactField == EditFieldRelType {
					// Move to next field when pressing down on relationship type
					m.newContactField = EditFieldNotes
					m.newContactInputs[m.newContactField].Focus()
					return m, textinput.Blink
				}
				// Pass through to text input for other fields
			}
			
			// Pass through to text input if not on relationship type field
			if m.newContactField != EditFieldRelType {
				var cmd tea.Cmd
				m.newContactInputs[m.newContactField], cmd = m.newContactInputs[m.newContactField].Update(msg)
				return m, cmd
			}
			return m, nil
		}
		
		// Edit mode handling
		if m.editMode {
			switch msg.String() {
			case "esc":
				// Cancel editing
				m.editMode = false
				m.editField = 0
				for i := range m.editInputs {
					m.editInputs[i].Blur()
				}
				return m, nil
				
			case "enter":
				// Save changes if ctrl+enter or cmd+enter is pressed
				if msg.Type == tea.KeyCtrlJ || msg.Type == tea.KeyCtrlM {
					contacts := m.filteredContacts()
					if len(contacts) > 0 && m.selected < len(contacts) {
						contact := contacts[m.selected]
						
						// Update the contact
						contact.Name = m.editInputs[EditFieldName].Value()
						contact.Email = db.NewNullString(m.editInputs[EditFieldEmail].Value())
						contact.Phone = db.NewNullString(m.editInputs[EditFieldPhone].Value())
						contact.Company = db.NewNullString(m.editInputs[EditFieldCompany].Value())
						contact.Notes = db.NewNullString(m.editInputs[EditFieldNotes].Value())
						contact.Label = db.NewNullString(m.editInputs[EditFieldLabel].Value())
						
						// Set relationship type from the selected index
						contact.RelationshipType = RelationshipTypes[m.editRelTypeIdx+1] // Skip "all"
						
						// Save to database
						err := m.db.UpdateContact(contact)
						if err != nil {
							m.err = err
						} else {
							// Reload contacts
							if newContacts, err := m.db.ListContacts(); err == nil {
								m.contacts = newContacts
							}
						}
					}
					
					// Exit edit mode
					m.editMode = false
					m.editField = 0
					for i := range m.editInputs {
						m.editInputs[i].Blur()
					}
					return m, nil
				}
				
				// Regular enter - only cycle relationship type if on that field
				if m.editField == EditFieldRelType {
					// Cycle through relationship types
					m.editRelTypeIdx = (m.editRelTypeIdx + 1) % (len(RelationshipTypes) - 1) // Skip "all"
					return m, nil
				}
				
			case "tab", "down":
				// Move to next field
				if m.editField < EditFieldCount-1 {
					m.editInputs[m.editField].Blur()
					m.editField++
					if m.editField != EditFieldRelType {
						m.editInputs[m.editField].Focus()
					}
				}
				return m, textinput.Blink
				
			case "shift+tab", "up":
				// Move to previous field
				if m.editField > 0 {
					if m.editField != EditFieldRelType {
						m.editInputs[m.editField].Blur()
					}
					m.editField--
					m.editInputs[m.editField].Focus()
				}
				return m, textinput.Blink
				
			case "left", "right":
				// For relationship type field navigation
				if m.editField == EditFieldRelType {
					if msg.String() == "left" && m.editRelTypeIdx > 0 {
						m.editRelTypeIdx--
					} else if msg.String() == "right" && m.editRelTypeIdx < len(RelationshipTypes)-2 {
						m.editRelTypeIdx++
					}
					return m, nil
				}
			}
			
			// Update the active text input
			if m.editField != EditFieldRelType {
				var cmd tea.Cmd
				m.editInputs[m.editField], cmd = m.editInputs[m.editField].Update(msg)
				return m, cmd
			}
			return m, nil
		}
		
		// State mode handling
		if m.stateMode {
			switch msg.String() {
			case "esc":
				m.stateMode = false
				m.stateSelected = 0
				return m, nil
			case "enter":
				// Update the contact state
				contacts := m.filteredContacts()
				if len(contacts) > 0 && m.selected < len(contacts) {
					contact := contacts[m.selected]
					newState := ContactStates[m.stateSelected]
					err := m.db.UpdateContactState(contact.ID, newState)
					if err != nil {
						m.err = err
					} else {
						// Create TaskWarrior task if state changed from "ok" to something else
						if newState != "ok" && m.taskwarriorClient.IsEnabled() {
							if contact.Label.Valid && contact.Label.String != "" {
								taskErr := m.taskwarriorClient.CreateContactTask(
									contact.Name, 
									newState, 
									contact.Label.String,
								)
								if taskErr != nil {
									// Don't fail the state change, just log the error
									m.err = fmt.Errorf("state updated but task creation failed: %w", taskErr)
								}
							} else {
								// Prompt for label instead of showing error
								m.labelPromptMode = true
								m.labelPromptContactID = contact.ID
								m.labelPromptNewState = newState
								m.labelPromptInput.SetValue("")
								m.labelPromptInput.Focus()
								m.stateMode = false // Exit state mode
								return m, textinput.Blink
							}
						}
						
						// Reload contacts to show updated state
						if newContacts, err := m.db.ListContacts(); err == nil {
							m.contacts = newContacts
							// Maintain selection within bounds after reload
							m.selected = m.ensureValidSelection()
						}
					}
				}
				m.stateMode = false
				m.stateSelected = 0
				return m, nil
			case "j", "down":
				if m.stateSelected < len(ContactStates)-1 {
					m.stateSelected++
				}
			case "k", "up":
				if m.stateSelected > 0 {
					m.stateSelected--
				}
			default:
				// Check if it's a hotkey
				if len(msg.String()) == 1 {
					char := rune(msg.String()[0])
					for i, hotkey := range m.stateHotkeys {
						if hotkey.Key == char {
							// Apply the state immediately
							contacts := m.filteredContacts()
							if len(contacts) > 0 && m.selected < len(contacts) {
								contact := contacts[m.selected]
								newState := ContactStates[i]
								err := m.db.UpdateContactState(contact.ID, newState)
								if err != nil {
									m.err = err
								} else {
									// Create TaskWarrior task if state changed from "ok" to something else
									if newState != "ok" && m.taskwarriorClient.IsEnabled() {
										if contact.Label.Valid && contact.Label.String != "" {
											taskErr := m.taskwarriorClient.CreateContactTask(
												contact.Name, 
												newState, 
												contact.Label.String,
											)
											if taskErr != nil {
												// Don't fail the state change, just log the error
												m.err = fmt.Errorf("state updated but task creation failed: %w", taskErr)
											}
										} else {
											// Prompt for label instead of showing error
											m.labelPromptMode = true
											m.labelPromptContactID = contact.ID
											m.labelPromptNewState = newState
											m.labelPromptInput.SetValue("")
											m.labelPromptInput.Focus()
											m.stateMode = false // Exit state mode
											return m, textinput.Blink
										}
									}
									
									// Reload contacts to show updated state
									if newContacts, err := m.db.ListContacts(); err == nil {
										m.contacts = newContacts
										m.selected = m.ensureValidSelection()
									}
								}
							}
							m.stateMode = false
							m.stateSelected = 0
							return m, nil
						}
					}
				}
			}
			return m, nil
		}
		
		// Note mode handling
		if m.noteMode {
			switch msg.String() {
			case "esc":
				m.noteMode = false
				m.noteType = 0
				m.noteInput.Reset()
				return m, nil
			case "enter":
				// Save the note only if ctrl+enter or cmd+enter is pressed
				if msg.Type == tea.KeyCtrlJ || msg.Type == tea.KeyCtrlM {
					// Save the note
					contacts := m.filteredContacts()
					if len(contacts) > 0 && m.selected < len(contacts) {
						contact := contacts[m.selected]
						note := m.noteInput.Value()
						if note != "" {
							interactionType := InteractionTypes[m.noteType]
							err := m.db.AddInteractionNote(contact.ID, interactionType, note)
							if err != nil {
								m.err = err
							}
						}
					}
					m.noteMode = false
					m.noteType = 0
					m.noteInput.Reset()
					return m, nil
				}
			case "tab":
				// Cycle through interaction types
				m.noteType = (m.noteType + 1) % len(InteractionTypes)
				return m, nil
			}
			
			// Pass other keys to the note input
			var cmd tea.Cmd
			m.noteInput, cmd = m.noteInput.Update(msg)
			return m, cmd
		}
		
		// Contact style mode handling
		if m.styleMode {
			if m.customFreqMode {
				// Custom frequency input mode
				switch msg.String() {
				case "enter":
					// Parse and save custom frequency
					var customDays *int
					if freq := m.customFreqInput.Value(); freq != "" {
						if days, err := fmt.Sscanf(freq, "%d", &customDays); err == nil && days == 1 {
							// Valid number entered
						} else {
							customDays = nil
						}
					}
					
					// Update the contact style
					err := m.db.UpdateContactStyle(m.styleContactID, "periodic", customDays)
					if err != nil {
						m.err = err
					} else {
						// Reload contacts
						if newContacts, err := m.db.ListContacts(); err == nil {
							m.contacts = newContacts
						}
					}
					
					m.customFreqMode = false
					m.styleMode = false
					m.customFreqInput.Reset()
					return m, nil
					
				case "esc":
					// Cancel custom frequency input
					m.customFreqMode = false
					m.customFreqInput.Reset()
					return m, nil
					
				default:
					// Update input field
					var cmd tea.Cmd
					m.customFreqInput, cmd = m.customFreqInput.Update(msg)
					return m, cmd
				}
			}
			
			// Style selection mode
			switch msg.String() {
			case "esc":
				m.styleMode = false
				m.styleSelected = 0
				return m, nil
				
			case "enter":
				// Apply selected style
				style := ContactStyles[m.styleSelected]
				
				if style == "periodic" {
					// Switch to custom frequency input mode
					m.customFreqMode = true
					m.customFreqInput.Focus()
					return m, nil
				} else {
					// Apply ambient or triggered style
					err := m.db.UpdateContactStyle(m.styleContactID, style, nil)
					if err != nil {
						m.err = err
					} else {
						// Reload contacts
						if newContacts, err := m.db.ListContacts(); err == nil {
							m.contacts = newContacts
						}
					}
					m.styleMode = false
					m.styleSelected = 0
				}
				return m, nil
				
			case "j", "down":
				if m.styleSelected < len(ContactStyles)-1 {
					m.styleSelected++
				}
				return m, nil
				
			case "k", "up":
				if m.styleSelected > 0 {
					m.styleSelected--
				}
				return m, nil
			}
			
			return m, nil
		}
		
		// Interaction edit mode handling
		if m.interactionEditMode {
			if m.interactionDeleteConfirm {
				// Delete confirmation mode
				switch msg.String() {
				case "y":
					// Confirm delete
					if m.interactionToDelete > 0 {
						err := m.db.DeleteInteraction(m.interactionToDelete)
						if err != nil {
							m.err = err
						} else {
							// Reload interactions
							contacts := m.filteredContacts()
							if len(contacts) > 0 && m.selected < len(contacts) {
								contact := contacts[m.selected]
								if interactions, err := m.db.GetContactInteractions(contact.ID, 20); err == nil {
									m.interactions = interactions
									// Adjust selection if needed
									if m.selectedInteraction >= len(m.interactions) {
										m.selectedInteraction = len(m.interactions) - 1
									}
									if m.selectedInteraction < 0 {
										// No more interactions, exit mode
										m.interactionEditMode = false
									}
								}
							}
						}
					}
					m.interactionDeleteConfirm = false
					m.interactionToDelete = 0
					return m, nil
				default:
					// Cancel delete
					m.interactionDeleteConfirm = false
					m.interactionToDelete = 0
					return m, nil
				}
			}
			
			// Check if we're editing an interaction
			if m.interactionEditInput.Focused() {
				switch msg.String() {
				case "esc":
					// Cancel edit
					m.interactionEditInput.Blur()
					m.interactionEditInput.Reset()
					return m, nil
				case "tab":
					// Cycle through interaction types
					m.interactionEditType = (m.interactionEditType + 1) % len(InteractionTypes)
					return m, nil
				case "enter":
					// Save on ctrl+enter or cmd+enter
					if msg.Type == tea.KeyCtrlJ || msg.Type == tea.KeyCtrlM {
						// Save the edit
						if m.selectedInteraction < len(m.interactions) {
							interaction := m.interactions[m.selectedInteraction]
							notes := m.interactionEditInput.Value()
							interactionType := InteractionTypes[m.interactionEditType]
							err := m.db.UpdateInteraction(interaction.ID, interactionType, notes)
							if err != nil {
								m.err = err
							} else {
								// Reload interactions
								contacts := m.filteredContacts()
								if len(contacts) > 0 && m.selected < len(contacts) {
									contact := contacts[m.selected]
									if interactions, err := m.db.GetContactInteractions(contact.ID, 20); err == nil {
										m.interactions = interactions
									}
								}
							}
						}
						m.interactionEditInput.Blur()
						m.interactionEditInput.Reset()
						return m, nil
					}
				}
				// Pass other keys to the textarea
				var cmd tea.Cmd
				m.interactionEditInput, cmd = m.interactionEditInput.Update(msg)
				return m, cmd
			}
			
			// Navigation mode
			switch msg.String() {
			case "esc", "q":
				// Exit interaction mode
				m.interactionEditMode = false
				m.selectedInteraction = 0
				m.interactions = nil
				return m, nil
			case "j", "down":
				if m.selectedInteraction < len(m.interactions)-1 {
					m.selectedInteraction++
				}
				return m, nil
			case "k", "up":
				if m.selectedInteraction > 0 {
					m.selectedInteraction--
				}
				return m, nil
			case "e":
				// Edit selected interaction
				if m.selectedInteraction < len(m.interactions) {
					interaction := m.interactions[m.selectedInteraction]
					m.interactionEditInput.Reset()
					if interaction.Notes.Valid {
						m.interactionEditInput.SetValue(interaction.Notes.String)
					}
					// Find current interaction type
					for i, iType := range InteractionTypes {
						if iType == interaction.InteractionType {
							m.interactionEditType = i
							break
						}
					}
					m.interactionEditInput.Focus()
					// Set width
					if m.width > 0 {
						detailWidth := m.width - (m.width / 3) - 3
						m.interactionEditInput.SetWidth(detailWidth - 10)
					}
					return m, textarea.Blink
				}
				return m, nil
			case "d":
				// Delete selected interaction
				if m.selectedInteraction < len(m.interactions) {
					m.interactionDeleteConfirm = true
					m.interactionToDelete = m.interactions[m.selectedInteraction].ID
				}
				return m, nil
			}
			return m, nil
		}
		
		// Filter mode handling
		if m.filterMode {
			switch msg.String() {
			case "esc":
				m.filterMode = false
				m.filter.Reset()
				m.selected = m.ensureValidSelection()
				return m, nil
			case "enter":
				m.filterMode = false
				m.selected = m.ensureValidSelection()
				return m, nil
			case "up":
				// Allow navigation with arrow keys
				if m.selected > 0 {
					m.selected--
				}
				return m, nil
			case "down":
				// Allow navigation with arrow keys
				if m.selected < len(m.filteredContacts())-1 {
					m.selected++
				}
				return m, nil
			}
			
			// Pass all other keys to the textinput
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			
			// Ensure selection is valid after filter change
			m.selected = m.ensureValidSelection()
			return m, cmd
		}
		
		// Help mode handling
		if m.showHelp {
			switch msg.String() {
			case "esc", "?", "q":
				m.showHelp = false
				m.helpScrollOffset = 0
				return m, nil
			case "j", "down":
				m.helpScrollOffset++
				return m, nil
			case "k", "up":
				if m.helpScrollOffset > 0 {
					m.helpScrollOffset--
				}
				return m, nil
			case "g":
				m.helpScrollOffset = 0
				return m, nil
			case "G":
				// This will be adjusted in renderHelpOverlay to max scroll
				m.helpScrollOffset = 999
				return m, nil
			}
			// Ignore other keys in help mode
			return m, nil
		}
		
		// Normal mode handling
		switch msg.String() {
		case "?":
			// Toggle help overlay
			m.showHelp = !m.showHelp
			if m.showHelp {
				m.helpScrollOffset = 0
			}
			return m, nil
			
		case "+", "N":
			// Enter new contact mode
			m.newContactMode = true
			m.newContactField = 0
			m.newContactRelTypeIdx = 3 // Default to "network"
			// Reset all inputs
			for i := range m.newContactInputs {
				m.newContactInputs[i].Reset()
			}
			m.newContactInputs[0].Focus() // Focus on name field
			return m, textinput.Blink
			
		case "r":
			// Enter relationship type filter mode
			m.typeFilterMode = true
			m.typeSelected = 0
			// If a filter is already active, select it
			if m.typeFilter != "" {
				for i, rType := range RelationshipTypes {
					if rType == m.typeFilter {
						m.typeSelected = i
						break
					}
				}
			}
			return m, nil
			
		case "q", "ctrl+c":
			return m, tea.Quit
			
		case "j", "down":
			if m.selected < len(m.filteredContacts())-1 {
				m.selected++
			}
			
		case "k", "up":
			if m.selected > 0 {
				m.selected--
			}
			
		case "/":
			m.filterMode = true
			// Reset and configure the textinput
			m.filter.Reset()
			m.filter.SetValue("") // Explicitly set empty value
			m.filter.Placeholder = "Filter contacts..."
			m.filter.Prompt = "> "
			// Set filter width
			if m.width > 0 {
				listWidth := m.width / 3
				m.filter.Width = listWidth - 6
			} else {
				m.filter.Width = 25
			}
			m.filter.Focus()
			// Force an immediate render
			return m, tea.Batch(textinput.Blink, tea.ClearScreen)
			
		case "esc":
			// Clear any error messages and return to normal operation
			if m.err != nil {
				m.err = nil
				return m, nil
			}
			// Clear any success messages
			if m.successMsg != "" {
				m.successMsg = ""
				return m, nil
			}
			// Close help overlay if open
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
			// Clear filter and return to full list
			if m.filter.Value() != "" {
				m.filter.Reset()
				m.selected = m.ensureValidSelection()
				return m, nil
			}
			
		case "s":
			// Enter state selection mode
			contacts := m.filteredContacts()
			if len(contacts) > 0 && m.selected < len(contacts) {
				m.stateMode = true
				m.stateSelected = 0
				// If contact has a current state, select it
				contact := contacts[m.selected]
				if contact.State.Valid {
					for i, state := range ContactStates {
						if state == contact.State.String {
							m.stateSelected = i
							break
						}
					}
				} else {
					// Default to "ok" if no state set
					for i, state := range ContactStates {
						if state == "ok" {
							m.stateSelected = i
							break
						}
					}
				}
			}
			
		case "S":
			// Toggle state filter (show non-ok states)
			m.stateFilter = !m.stateFilter
			m.selected = m.ensureValidSelection()
			return m, nil
			
		case "o":
			// Toggle overdue filter
			m.overdueFilter = !m.overdueFilter
			m.selected = m.ensureValidSelection()
			return m, nil
			
		case "n":
			// Enter note mode
			contacts := m.filteredContacts()
			if len(contacts) > 0 && m.selected < len(contacts) {
				m.noteMode = true
				m.noteType = 0 // Default to "manual"
				m.noteInput.Reset()
				m.noteInput.Focus()
				// Set note input width based on detail pane width
				if m.width > 0 {
					detailWidth := m.width - (m.width / 3) - 3
					m.noteInput.SetWidth(detailWidth - 10)
				}
				return m, textarea.Blink
			}
			
		case "C":
			// Clear all filters
			m.stateFilter = false
			m.overdueFilter = false
			m.typeFilter = ""
			m.showArchived = false
			m.filter.Reset()
			m.selected = m.ensureValidSelection()
			return m, nil
			
		case "b":
			// Bump contact - enter confirmation mode
			contacts := m.filteredContacts()
			if len(contacts) > 0 && m.selected < len(contacts) {
				contact := contacts[m.selected]
				m.bumpConfirmMode = true
				m.bumpContactID = contact.ID
			}
			return m, nil
			
		case "c":
			// Mark as contacted
			contacts := m.filteredContacts()
			if len(contacts) > 0 && m.selected < len(contacts) {
				contact := contacts[m.selected]
				err := m.db.MarkContacted(contact.ID, "manual", "Marked via TUI")
				if err != nil {
					m.err = err
				} else {
					// Reload contacts to show updated state
					if newContacts, err := m.db.ListContacts(); err == nil {
						m.contacts = newContacts
						// Maintain selection within bounds after reload
						m.selected = m.ensureValidSelection()
					}
				}
			}
			
		case "e":
			// Enter edit mode
			contacts := m.filteredContacts()
			if len(contacts) > 0 && m.selected < len(contacts) {
				contact := contacts[m.selected]
				m.enterEditMode(contact)
			}
			return m, nil
			
		case "a":
			// Toggle archive status
			contacts := m.filteredContacts()
			if len(contacts) > 0 && m.selected < len(contacts) {
				contact := contacts[m.selected]
				var err error
				if contact.Archived {
					err = m.db.UnarchiveContact(contact.ID)
				} else {
					err = m.db.ArchiveContact(contact.ID)
				}
				if err != nil {
					m.err = err
				} else {
					// Reload contacts to show updated state
					if newContacts, err := m.db.ListContacts(); err == nil {
						m.contacts = newContacts
						m.selected = m.ensureValidSelection()
					}
				}
			}
			return m, nil
			
		case "A":
			// Toggle showing archived contacts
			m.showArchived = !m.showArchived
			m.selected = m.ensureValidSelection()
			return m, nil
			
		case "D":
			// Delete contact with confirmation
			contacts := m.filteredContacts()
			if len(contacts) > 0 && m.selected < len(contacts) {
				contact := contacts[m.selected]
				m.deleteConfirmMode = true
				m.deleteContactID = contact.ID
				m.deleteContactName = contact.Name
			}
			return m, nil
			
		case "i":
			// Enter interaction view/edit mode
			contacts := m.filteredContacts()
			if len(contacts) > 0 && m.selected < len(contacts) {
				contact := contacts[m.selected]
				// Load interactions for this contact
				interactions, err := m.db.GetContactInteractions(contact.ID, 20) // Get more interactions
				if err == nil && len(interactions) > 0 {
					m.interactionEditMode = true
					m.selectedInteraction = 0
					m.interactions = interactions
					m.interactionEditInput.Reset()
					m.interactionEditType = 0
				}
			}
			return m, nil
			
		case "t":
			// Enter task view mode
			contacts := m.filteredContacts()
			if len(contacts) > 0 && m.selected < len(contacts) {
				contact := contacts[m.selected]
				if m.taskwarriorClient.IsEnabled() && contact.Label.Valid && contact.Label.String != "" {
					tasks, err := m.taskwarriorClient.GetContactTasks(contact.Label.String)
					if err == nil {
						m.taskMode = true
						m.tasks = tasks
						m.selectedTask = 0
					} else {
						m.err = fmt.Errorf("loading tasks: %w", err)
					}
				} else if !m.taskwarriorClient.IsEnabled() {
					m.err = fmt.Errorf("TaskWarrior not available")
				} else {
					m.err = fmt.Errorf("contact must have a label to view tasks")
				}
			}
			return m, nil
			
		case "m":
			// Change contact style
			contacts := m.filteredContacts()
			if len(contacts) > 0 && m.selected < len(contacts) {
				contact := contacts[m.selected]
				m.styleMode = true
				m.styleSelected = 0
				m.styleContactID = contact.ID
				// Set initial selection based on current style
				for i, style := range ContactStyles {
					if style == contact.ContactStyle {
						m.styleSelected = i
						break
					}
				}
			}
			return m, nil
		}
	}
	
	return m, nil
}

// filteredContacts returns contacts matching the current filter
func (m Model) filteredContacts() []db.Contact {
	var filtered []db.Contact
	
	// Start with all contacts
	contacts := m.contacts
	
	// Filter archived contacts (unless showing archived)
	if !m.showArchived {
		var activeContacts []db.Contact
		for _, c := range contacts {
			if !c.Archived {
				activeContacts = append(activeContacts, c)
			}
		}
		contacts = activeContacts
	}
	
	// Apply relationship type filter
	if m.typeFilter != "" {
		var typeFiltered []db.Contact
		for _, c := range contacts {
			if c.RelationshipType == m.typeFilter {
				typeFiltered = append(typeFiltered, c)
			}
		}
		contacts = typeFiltered
	}
	
	// Apply smart filters
	if m.stateFilter {
		var stateFiltered []db.Contact
		for _, c := range contacts {
			// Include contacts with non-ok states or no state set
			if c.State.Valid && c.State.String != "ok" {
				stateFiltered = append(stateFiltered, c)
			}
		}
		contacts = stateFiltered
	}
	
	if m.overdueFilter {
		var overdueFiltered []db.Contact
		for _, c := range contacts {
			if c.IsOverdue() {
				overdueFiltered = append(overdueFiltered, c)
			}
		}
		contacts = overdueFiltered
	}
	
	// Apply text filter if present
	if m.filter.Value() == "" {
		return contacts
	}
	
	filter := strings.ToLower(m.filter.Value())
	
	for _, c := range contacts {
		if strings.Contains(strings.ToLower(c.Name), filter) ||
		   (c.Label.Valid && strings.Contains(strings.ToLower(c.Label.String), filter)) ||
		   (c.Company.Valid && strings.Contains(strings.ToLower(c.Company.String), filter)) {
			filtered = append(filtered, c)
		}
	}
	
	return filtered
}

// ensureValidSelection ensures the current selection is within bounds
func (m Model) ensureValidSelection() int {
	contacts := m.filteredContacts()
	if len(contacts) == 0 {
		return 0
	}
	if m.selected >= len(contacts) {
		return len(contacts) - 1
	}
	if m.selected < 0 {
		return 0
	}
	return m.selected
}
// View renders the UI
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress Esc to continue or q to quit.", m.err)
	}
	
	if m.successMsg != "" {
		return fmt.Sprintf("Success: %v\n\nPress Esc to continue.", m.successMsg)
	}
	
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	
	// Calculate pane widths
	listWidth := m.width / 3
	detailWidth := m.width - listWidth - 3 // account for borders
	
	// Build the list view
	listView := m.renderList(listWidth, m.height-3)
	
	// Build the detail view
	detailView := m.renderDetail(detailWidth, m.height-3)
	
	// Join horizontally
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		borderStyle.Width(listWidth).Height(m.height-3).Render(listView),
		borderStyle.Width(detailWidth).Height(m.height-3).Render(detailView),
	)
	
	// Add help line
	help := m.renderHelp()
	
	view := lipgloss.JoinVertical(lipgloss.Left, content, help)
	
	// Overlay relationship type selection if in type filter mode
	if m.typeFilterMode {
		return m.renderTypeSelection()
	}
	
	// Overlay state selection if in state mode
	if m.stateMode {
		return m.renderStateSelection()
	}
	
	// Overlay note input if in note mode
	if m.noteMode {
		return m.renderNoteInput()
	}
	
	// Overlay edit mode if active
	if m.editMode {
		return m.renderEditMode()
	}
	
	// Overlay new contact mode if active
	if m.newContactMode {
		return m.renderNewContactMode()
	}
	
	// Overlay bump confirmation if active
	if m.bumpConfirmMode {
		return m.renderBumpConfirmation()
	}
	
	// Overlay delete confirmation if active
	if m.deleteConfirmMode {
		return m.renderDeleteConfirmation()
	}
	
	// Overlay style mode if active
	if m.styleMode {
		return m.renderStyleMode()
	}
	
	// Overlay task mode if active
	if m.taskMode {
		return m.renderTaskMode()
	}
	
	// Overlay label prompt mode if active
	if m.labelPromptMode {
		return m.renderLabelPrompt()
	}
	
	// Overlay help if active
	if m.showHelp {
		return m.renderHelpOverlay()
	}
	
	// Overlay interaction edit mode if active
	if m.interactionEditMode {
		return m.renderInteractionEditMode()
	}
	
	return view
}

// renderList renders the contact list
func (m Model) renderList(width, height int) string {
	var lines []string
	
	if m.filterMode {
		// Always show the filter when in filter mode, even if empty
		filterView := m.filter.View()
		if filterView == "" {
			// Fallback if View() returns empty
			filterView = "> " + m.filter.Placeholder
		}
		lines = append(lines, filterView)
		lines = append(lines, "")
		height -= 2
	}
	
	contacts := m.filteredContacts()
	
	// Calculate visible range
	visibleHeight := height - 2 // account for header
	startIdx := 0
	if m.selected >= visibleHeight {
		startIdx = m.selected - visibleHeight + 1
	}
	
	// Header
	header := "Contacts (" + fmt.Sprintf("%d", len(contacts)) + ")"
	
	// Add filter indicators
	var filterIndicators []string
	if m.typeFilter != "" {
		filterIndicators = append(filterIndicators, "type:"+m.typeFilter)
	}
	if m.stateFilter {
		filterIndicators = append(filterIndicators, "state:non-ok")
	}
	if m.overdueFilter {
		filterIndicators = append(filterIndicators, "overdue")
	}
	if m.showArchived {
		filterIndicators = append(filterIndicators, "archived")
	}
	if len(filterIndicators) > 0 {
		header += " [" + strings.Join(filterIndicators, ", ") + "]"
	}
	
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", width-2))
	
	// Contact list
	for i := startIdx; i < len(contacts) && i < startIdx+visibleHeight; i++ {
		c := contacts[i]
		
		// Determine the single most important indicator to show
		// Priority: non-ok state > overdue > contact style > none
		var indicator string
		var indicatorStyle func(...string) string
		
		if c.State.Valid && c.State.String != "ok" {
			indicator = "●"
			indicatorStyle = stateStyle.Render
		} else if c.IsOverdue() {
			indicator = "*"
			indicatorStyle = overdueStyle.Render
		} else {
			switch c.ContactStyle {
			case "ambient":
				indicator = "∞"
				indicatorStyle = greenStyle.Render
			case "triggered":
				indicator = "⚡"
				indicatorStyle = yellowStyle.Render
			default:
				indicator = " "
				indicatorStyle = func(s ...string) string { return strings.Join(s, "") }
			}
		}
		
		// Build name content
		nameContent := c.Name
		if c.Label.Valid {
			label := strings.TrimSpace(strings.ReplaceAll(c.Label.String, "\n", " "))
			nameContent += " [" + label + "]"
		}
		if c.Archived {
			nameContent = "[ARCH] " + nameContent
		}
		
		// Build the line with consistent spacing and leading space
		var line string
		if i == m.selected {
			// Selected: style the entire line uniformly with leading space
			rawLine := fmt.Sprintf("▶ %s %s", indicator, nameContent)
			line = selectedStyle.Render(rawLine)
		} else {
			// Non-selected: leading space + styled indicator + space + name
			line = "  " + indicatorStyle(indicator) + " "
			
			// Add name content with appropriate styling
			if c.Archived {
				if c.Label.Valid {
					label := strings.TrimSpace(strings.ReplaceAll(c.Label.String, "\n", " "))
					line += dimmedStyle.Render("[ARCH] ") + c.Name + " " + labelStyle.Render("["+label+"]")
				} else {
					line += dimmedStyle.Render("[ARCH] ") + c.Name
				}
			} else {
				if c.Label.Valid {
					label := strings.TrimSpace(strings.ReplaceAll(c.Label.String, "\n", " "))
					line += c.Name + " " + labelStyle.Render("["+label+"]")
				} else {
					line += c.Name
				}
			}
		}
		
		lines = append(lines, line)
	}
	
	return strings.Join(lines, "\n")
}
// renderDetail renders the contact detail view
func (m Model) renderDetail(width, height int) string {
	contacts := m.filteredContacts()
	if len(contacts) == 0 || m.selected >= len(contacts) {
		return "No contact selected"
	}
	
	c := contacts[m.selected]
	var lines []string
	
	// Header
	header := c.Name
	if c.Label.Valid {
		header += " (" + c.Label.String + ")"
	}
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", width-2))
	lines = append(lines, "")
	
	// Basic info
	if c.Company.Valid {
		lines = append(lines, fmt.Sprintf("Company: %s", c.Company.String))
	}
	lines = append(lines, fmt.Sprintf("Relationship: %s", c.RelationshipType))
	
	if c.State.Valid {
		lines = append(lines, fmt.Sprintf("State: %s", c.State.String))
	} else {
		lines = append(lines, "State: ok")
	}
	
	if c.Email.Valid {
		lines = append(lines, fmt.Sprintf("Email: %s", c.Email.String))
	}
	if c.Phone.Valid {
		lines = append(lines, fmt.Sprintf("Phone: %s", c.Phone.String))
	}
	
	if c.ContactedAt.Valid {
		days := int(time.Since(c.ContactedAt.Time).Hours() / 24)
		lines = append(lines, fmt.Sprintf("Last Contact: %s (%d days ago)", 
			c.ContactedAt.Time.Format("2006-01-02"), days))
	} else {
		lines = append(lines, "Last Contact: Never")
	}
	
	// Show bump info if contact has been bumped
	if c.BumpCount > 0 {
		bumpInfo := fmt.Sprintf("Bumped: %d time", c.BumpCount)
		if c.BumpCount > 1 {
			bumpInfo += "s"
		}
		if c.LastBumpDate.Valid {
			days := int(time.Since(c.LastBumpDate.Time).Hours() / 24)
			bumpInfo += fmt.Sprintf(" (last: %d days ago)", days)
		}
		lines = append(lines, bumpInfo)
	}
	
	// Contact style
	styleInfo := fmt.Sprintf("Style: %s", c.ContactStyle)
	if c.ContactStyle == "periodic" && c.CustomFrequencyDays.Valid {
		styleInfo += fmt.Sprintf(" (%d days)", c.CustomFrequencyDays.Int64)
	}
	lines = append(lines, styleInfo)
	
	lines = append(lines, "")
	
	// Notes
	if c.Notes.Valid && c.Notes.String != "" {
		lines = append(lines, "Notes:")
		lines = append(lines, c.Notes.String)
		lines = append(lines, "")
	}
	
	// Recent Interactions
	interactions, err := m.db.GetContactInteractions(c.ID, 5)
	if err == nil && len(interactions) > 0 {
		lines = append(lines, "Recent Interactions:")
		lines = append(lines, strings.Repeat("─", width-2))
		for _, log := range interactions {
			dateStr := log.InteractionDate.Format("2006-01-02 15:04")
			typeStr := fmt.Sprintf("[%s]", log.InteractionType)
			lines = append(lines, fmt.Sprintf("%s %s", dateStr, typeStr))
			if log.Notes.Valid && log.Notes.String != "" {
				// Wrap long notes
				noteLines := wrapText(log.Notes.String, width-4)
				for _, noteLine := range noteLines {
					lines = append(lines, "  "+noteLine)
				}
			}
			lines = append(lines, "")
		}
	}
	
	return strings.Join(lines, "\n")
}

// renderHelp renders the help line
func (m Model) renderHelp() string {
	if m.deleteConfirmMode {
		return " y: DELETE CONTACT • any other key: cancel"
	}
	
	if m.bumpConfirmMode {
		return " y: confirm bump • any other key: cancel"
	}
	
	if m.typeFilterMode {
		return " Press hotkey to select • Esc: cancel"
	}
	
	if m.stateMode {
		return " j/k: navigate • Enter: confirm • Esc: cancel"
	}
	
	if m.taskMode {
		return " j/k: navigate tasks • Enter/Space: mark task complete • r: refresh • Esc: back to contacts"
	}
	
	if m.labelPromptMode {
		return " Enter: save label and create task • Esc: cancel"
	}
	
	if m.noteMode {
		return " Type note • Tab: change type • Ctrl+Enter: save • Esc: cancel"
	}
	
	if m.editMode {
		return " Tab/↓: next • Shift+Tab/↑: prev • Ctrl+Enter: save • Esc: cancel"
	}
	
	if m.filterMode {
		return " Type to filter • ↑/↓: navigate • Enter: confirm • Esc: cancel"
	}
	
	help := " j/k: navigate • /: filter • c: contacted • ?: help • q: quit"
	
	// Show clear option if any filters are active
	if m.stateFilter || m.overdueFilter || m.typeFilter != "" || m.filter.Value() != "" || m.showArchived {
		help += " • C: clear filters"
	}
	
	if m.filter.Value() != "" {
		help += " • Esc: clear filter"
	}
	
	return help
}

// renderStateSelection renders the state selection overlay
func (m Model) renderStateSelection() string {
	contacts := m.filteredContacts()
	if len(contacts) == 0 || m.selected >= len(contacts) {
		return "No contact selected"
	}
	
	contact := contacts[m.selected]
	
	var lines []string
	lines = append(lines, fmt.Sprintf("Set state for %s:", contact.Name))
	lines = append(lines, "")
	
	for i, hotkey := range m.stateHotkeys {
		// Format the hotkey display
		stateDisplay := ""
		foundKey := false
		for _, char := range hotkey.Label {
			if !foundKey && char == hotkey.Key {
				stateDisplay += fmt.Sprintf("[%c]", char)
				foundKey = true
			} else {
				stateDisplay += string(char)
			}
		}
		
		// If hotkey wasn't in the word, prepend it
		if !foundKey {
			stateDisplay = fmt.Sprintf("[%c] %s", hotkey.Key, hotkey.Label)
		}
		
		line := fmt.Sprintf("  %s", stateDisplay)
		if i == m.stateSelected {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)
	}
	
	lines = append(lines, "")
	lines = append(lines, "Press hotkey to select, Esc to cancel")
	
	// Create a bordered box and center it
	content := strings.Join(lines, "\n")
	box := borderStyle.
		Padding(1).
		Render(content)
	
	// Center the box on the screen
	centered := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
	
	return centered
}

// renderNoteInput renders the note input overlay
func (m Model) renderNoteInput() string {
	contacts := m.filteredContacts()
	if len(contacts) == 0 || m.selected >= len(contacts) {
		return "No contact selected"
	}
	
	contact := contacts[m.selected]
	
	var lines []string
	lines = append(lines, fmt.Sprintf("Add note for %s:", contact.Name))
	lines = append(lines, "")
	
	// Show interaction type selector
	lines = append(lines, "Type: ")
	typeSelector := ""
	for i, iType := range InteractionTypes {
		if i == m.noteType {
			typeSelector += noteTypeSelectorStyle.Render(fmt.Sprintf("[%s]", iType)) + " "
		} else {
			typeSelector += fmt.Sprintf(" %s  ", iType)
		}
	}
	lines = append(lines, typeSelector)
	lines = append(lines, "")
	
	// Show note input
	lines = append(lines, m.noteInput.View())
	lines = append(lines, "")
	lines = append(lines, "Tab: change type • Ctrl+Enter: save • Esc: cancel")
	
	// Create a bordered box and center it
	content := strings.Join(lines, "\n")
	box := borderStyle.
		Padding(1).
		Render(content)
	
	// Center the box on the screen
	centered := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
	
	return centered
}

// renderTypeSelection renders the relationship type selection overlay
func (m Model) renderTypeSelection() string {
	var lines []string
	lines = append(lines, "Filter by relationship type:")
	lines = append(lines, "")
	
	for i, hotkey := range m.relationshipHotkeys {
		// Format the hotkey display
		display := ""
		foundKey := false
		for _, char := range hotkey.Label {
			if !foundKey && char == hotkey.Key {
				display += fmt.Sprintf("[%c]", char)
				foundKey = true
			} else {
				display += string(char)
			}
		}
		
		// If hotkey wasn't in the word, prepend it
		if !foundKey {
			display = fmt.Sprintf("[%c] %s", hotkey.Key, hotkey.Label)
		}
		
		// Special case for "all"
		if hotkey.Label == "all" {
			display += " (clear filter)"
		}
		
		line := fmt.Sprintf("  %s", display)
		if i == m.typeSelected {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)
	}
	
	lines = append(lines, "")
	lines = append(lines, "Press hotkey to select, Esc to cancel")
	
	// Create a bordered box and center it
	content := strings.Join(lines, "\n")
	box := borderStyle.
		Padding(1).
		Render(content)
	
	// Center the box on the screen
	centered := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
	
	return centered
}

// renderEditMode renders the edit mode overlay
func (m Model) renderEditMode() string {
	contacts := m.filteredContacts()
	if len(contacts) == 0 || m.selected >= len(contacts) {
		return "No contact selected"
	}
	
	contact := contacts[m.selected]
	
	var lines []string
	lines = append(lines, fmt.Sprintf("Edit Contact: %s", contact.Name))
	lines = append(lines, strings.Repeat("─", 40))
	lines = append(lines, "")
	
	// Field labels and inputs
	fieldLabels := []string{
		"Name:            ",
		"Email:           ",
		"Phone:           ",
		"Company:         ",
		"Relationship:    ",
		"Notes:           ",
		"Label:           ",
	}
	
	for i, label := range fieldLabels {
		var fieldView string
		
		if i == EditFieldRelType {
			// Special handling for relationship type
			relType := RelationshipTypes[m.editRelTypeIdx+1] // Skip "all"
			if i == m.editField {
				fieldView = label + selectedStyle.Render(fmt.Sprintf("< %s >", relType))
			} else {
				fieldView = label + fmt.Sprintf("  %s  ", relType)
			}
		} else {
			// Regular text input fields
			if i == m.editField {
				fieldView = label + m.editInputs[i].View()
			} else {
				value := m.editInputs[i].Value()
				if value == "" {
					value = m.editInputs[i].Placeholder
				}
				fieldView = label + value
			}
		}
		
		lines = append(lines, fieldView)
		lines = append(lines, "")
	}
	
	lines = append(lines, "")
	lines = append(lines, "Tab/↓: next field • Shift+Tab/↑: previous • Ctrl+Enter: save • Esc: cancel")
	
	// Create a bordered box
	content := strings.Join(lines, "\n")
	box := borderStyle.
		Padding(1).
		Width(60).
		Render(content)
	
	// Center the box on the screen
	centered := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
	
	return centered
}

// wrapText wraps text to fit within the specified width
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	
	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}
	
	currentLine := words[0]
	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	
	return lines
}

// renderBumpConfirmation renders the bump confirmation prompt
func (m Model) renderBumpConfirmation() string {
	contacts := m.filteredContacts()
	var contactName string
	
	// Find the contact being bumped
	for _, c := range contacts {
		if c.ID == m.bumpContactID {
			contactName = c.Name
			break
		}
	}
	
	// Build the confirmation prompt
	width := 60
	height := 7
	
	prompt := fmt.Sprintf("Bump contact '%s'? (y/n)", contactName)
	
	content := lipgloss.NewStyle().
		Width(width-4).
		Height(height-4).
		Align(lipgloss.Center, lipgloss.Center).
		Render(prompt)
	
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Width(width).
		Height(height).
		Render(content)
	
	// Center on screen
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}
// renderDeleteConfirmation renders the delete confirmation prompt
func (m Model) renderDeleteConfirmation() string {
	// Build the confirmation prompt
	width := 60
	height := 10
	
	prompt := fmt.Sprintf("Delete contact '%s'?\n\n"+
		"This will permanently delete the contact\n"+
		"and all associated interaction logs.\n\n"+
		"This action cannot be undone!\n\n"+
		"Press 'y' to confirm, any other key to cancel.", m.deleteContactName)
	
	content := lipgloss.NewStyle().
		Width(width-4).
		Height(height-4).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(lipgloss.Color("196")). // Red text for warning
		Render(prompt)
	
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")). // Red border for danger
		Width(width).
		Height(height).
		Render(content)
	
	// Center on screen
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}

// enterEditMode enters edit mode for the given contact
func (m *Model) enterEditMode(contact db.Contact) {
	m.editMode = true
	m.editField = 0
	
	// Populate edit inputs with current values
	m.editInputs[EditFieldName].SetValue(contact.Name)
	if contact.Email.Valid {
		m.editInputs[EditFieldEmail].SetValue(contact.Email.String)
	} else {
		m.editInputs[EditFieldEmail].SetValue("")
	}
	if contact.Phone.Valid {
		m.editInputs[EditFieldPhone].SetValue(contact.Phone.String)
	} else {
		m.editInputs[EditFieldPhone].SetValue("")
	}
	if contact.Company.Valid {
		m.editInputs[EditFieldCompany].SetValue(contact.Company.String)
	} else {
		m.editInputs[EditFieldCompany].SetValue("")
	}
	if contact.Notes.Valid {
		m.editInputs[EditFieldNotes].SetValue(contact.Notes.String)
	} else {
		m.editInputs[EditFieldNotes].SetValue("")
	}
	if contact.Label.Valid {
		m.editInputs[EditFieldLabel].SetValue(contact.Label.String)
	} else {
		m.editInputs[EditFieldLabel].SetValue("")
	}
	
	// Set the relationship type index
	m.editRelTypeIdx = 0 // Default to first type
	if contact.RelationshipType != "" {
		for i, rType := range RelationshipTypes[1:] { // Skip "all"
			if rType == contact.RelationshipType {
				m.editRelTypeIdx = i
				break
			}
		}
	}
	
	// Focus first field
	m.editInputs[0].Focus()
}

// renderStyleMode renders the contact style selection overlay
func (m Model) renderStyleMode() string {
	width := 60
	height := 20
	
	content := "Select Contact Style:\n\n"
	
	// Show current contact info
	contacts := m.filteredContacts()
	if len(contacts) > m.selected {
		contact := contacts[m.selected]
		content += fmt.Sprintf("Contact: %s\n", contact.Name)
		content += fmt.Sprintf("Current style: %s", contact.ContactStyle)
		if contact.ContactStyle == "periodic" && contact.CustomFrequencyDays.Valid {
			content += fmt.Sprintf(" (%d days)", contact.CustomFrequencyDays.Int64)
		}
		content += "\n\n"
	}
	
	if m.customFreqMode {
		// Custom frequency input mode
		content += "Enter custom frequency in days:\n\n"
		content += m.customFreqInput.View() + "\n\n"
		content += "(Press Enter to save, Esc to cancel)"
	} else {
		// Style selection mode
		for i, style := range ContactStyles {
			if i == m.styleSelected {
				content += fmt.Sprintf("→ %s", style)
			} else {
				content += fmt.Sprintf("  %s", style)
			}
			
			// Add description
			switch style {
			case "periodic":
				content += " - Regular cadence checking"
			case "ambient":
				content += " - Regular/automatic contact (∞)"
			case "triggered":
				content += " - Event-based outreach (⚡)"
			}
			content += "\n"
		}
		
		content += "\n(Press Enter to select, Esc to cancel)"
	}
	
	// Create bordered box
	boxStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2)
	
	// Center the box
	centeredStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)
	
	return centeredStyle.Render(boxStyle.Render(content))
}

// renderHelpOverlay renders the full help screen with scrolling support
func (m Model) renderHelpOverlay() string {
	width := 80
	height := 30
	
	helpLines := []string{
		"Contacts TUI - Keyboard Shortcuts",
		"",
		"Navigation:",
		"  j/k, ↓/↑     Navigate contacts",
		"  g            Go to top",
		"  G            Go to bottom",
		"  q, Ctrl+C    Quit",
		"",
		"Contact Actions:",
		"  +, N         Create new contact",
		"  c            Mark as contacted",
		"  b            Bump (reset date without contact)",
		"  e            Edit contact details",
		"  n            Add note/interaction",
		"  i            View/edit interaction history",
		"  t            View/manage TaskWarrior tasks",
		"  a            Archive/unarchive contact",
		"  m            Change contact style (periodic/ambient/triggered)",
		"  D            Delete contact (with confirmation)",
		"",
		"State Management:",
		"  s            Change contact state (ping, write, ok, etc.)",
		"  S            Toggle filter: show only non-ok states",
		"",
		"Filtering:",
		"  /            Search/filter contacts",
		"  r            Filter by relationship type",
		"  o            Toggle filter: show only overdue",
		"  A            Toggle: show/hide archived contacts",
		"  C            Clear all active filters",
		"  Esc          Clear search filter / Close help",
		"",
		"Help:",
		"  ?            Toggle this help screen",
		"",
		"In Help Mode:",
		"  j/k          Scroll down/up",
		"  g/G          Go to top/bottom",
		"  Esc, ?, q    Close help",
	}
	
	// Calculate visible area (accounting for borders and padding)
	visibleHeight := height - 4
	totalLines := len(helpLines)
	
	// Adjust scroll offset bounds
	maxOffset := totalLines - visibleHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	
	// Handle "G" - go to bottom (use local variable for calculations)
	scrollOffset := m.helpScrollOffset
	if scrollOffset > maxOffset {
		scrollOffset = maxOffset
	}
	
	// Ensure scroll offset is within bounds
	if scrollOffset < 0 {
		scrollOffset = 0
	}
	if scrollOffset > maxOffset {
		scrollOffset = maxOffset
	}
	
	// Get visible lines
	startLine := scrollOffset
	endLine := startLine + visibleHeight
	if endLine > totalLines {
		endLine = totalLines
	}
	
	visibleLines := helpLines[startLine:endLine]
	
	// Build content with scroll indicators
	content := ""
	
	// Add scroll up indicator if needed
	if scrollOffset > 0 {
		content += lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("▲ (more above)") + "\n"
		visibleLines = visibleLines[1:] // Remove one line to make room
	}
	
	// Add the visible help content
	for _, line := range visibleLines {
		content += line + "\n"
	}
	
	// Add scroll down indicator if needed
	if scrollOffset < maxOffset {
		// Remove last line to make room for indicator
		lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
		if len(lines) > 1 {
			content = strings.Join(lines[:len(lines)-1], "\n") + "\n"
		}
		content += lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("▼ (more below)")
	}
	
	// Style the help content
	styledContent := lipgloss.NewStyle().
		Width(width-4).
		Height(height-4).
		Padding(1).
		Render(content)
	
	// Create the box
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Width(width).
		Height(height).
		Render(styledContent)
	
	// Center on screen
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}

func (m Model) renderTaskMode() string {
	width := 80
	height := 20
	
	content := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("32")).
		MarginBottom(1).
		Render("TaskWarrior Tasks") + "\n\n"
	
	// Show current contact info
	contacts := m.filteredContacts()
	if len(contacts) > 0 && m.selected < len(contacts) {
		contact := contacts[m.selected]
		contactInfo := fmt.Sprintf("Contact: %s", contact.Name)
		if contact.Label.Valid && contact.Label.String != "" {
			contactInfo += fmt.Sprintf(" (%s)", contact.Label.String)
		}
		content += lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			MarginBottom(1).
			Render(contactInfo) + "\n\n"
	}
	
	// Show error if any
	if m.err != nil {
		content += lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			MarginBottom(1).
			Render("Error: " + m.err.Error()) + "\n\n"
	}
	
	// Show tasks
	if len(m.tasks) == 0 {
		content += lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("No tasks found for this contact.") + "\n"
	} else {
		content += fmt.Sprintf("Tasks (%d):\n\n", len(m.tasks))
		
		// Display tasks with selection
		for i, task := range m.tasks {
			line := fmt.Sprintf("  %s", task.Description)
			
			// Add task metadata
			if task.Priority != "" {
				line += fmt.Sprintf(" [%s]", task.Priority)
			}
			if task.Due != "" {
				line += fmt.Sprintf(" (due: %s)", task.Due)
			}
			
			// Highlight selected task
			if i == m.selectedTask {
				line = selectedStyle.Render("▶ " + line[2:])
			}
			
			content += line + "\n"
		}
	}
	
	content += "\n\n"
	
	// Add help text at the bottom
	helpText := " j/k: navigate tasks • Enter/Space: mark task complete • r: refresh • Esc: back to contacts"
	content += lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render(helpText) + "\n"
	
	// Create a box style
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1).
		Width(width).
		Height(height)
	
	// Center the box on screen
	centeredStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)
	
	return centeredStyle.Render(boxStyle.Render(content))
}

func (m Model) renderLabelPrompt() string {
	width := 60
	height := 12
	
	// Get the contact name for the prompt
	contactName := "Contact"
	if contact, err := m.db.GetContact(m.labelPromptContactID); err == nil {
		contactName = contact.Name
	}
	
	content := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("32")).
		MarginBottom(1).
		Render("Add Label for TaskWarrior Task") + "\n\n"
	
	content += fmt.Sprintf("Contact: %s\n", contactName)
	content += fmt.Sprintf("New State: %s\n\n", m.labelPromptNewState)
	content += "This contact needs a label to create TaskWarrior tasks.\n"
	content += "Enter a unique label (will be used as @tag):\n\n"
	
	// Show error if any
	if m.err != nil {
		content += lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			MarginBottom(1).
			Render("Error: " + m.err.Error()) + "\n\n"
	}
	
	content += "Label: " + m.labelPromptInput.View() + "\n\n"
	content += lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("Enter: save • Esc: cancel")
	
	// Create a box style
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1).
		Width(width).
		Height(height)
	
	// Center the box on screen
	centeredStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)
	
	return centeredStyle.Render(boxStyle.Render(content))
}

func (m Model) renderNewContactMode() string {
	width := 60
	fieldHeight := 3
	totalHeight := (EditFieldCount-1)*fieldHeight + 12 // account for title, spacing, and buttons
	
	content := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("32")).
		MarginBottom(1).
		Render("Create New Contact") + "\n\n"
	
	// Show error if any
	if m.err != nil {
		content += lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			MarginBottom(1).
			Render("Error: " + m.err.Error()) + "\n\n"
	}
	
	// Name field
	nameLabel := "Name: "
	if m.newContactField == EditFieldName {
		nameLabel = selectedStyle.Render(nameLabel)
	}
	content += nameLabel + m.newContactInputs[EditFieldName].View() + "\n\n"
	
	// Email field
	emailLabel := "Email: "
	if m.newContactField == EditFieldEmail {
		emailLabel = selectedStyle.Render(emailLabel)
	}
	content += emailLabel + m.newContactInputs[EditFieldEmail].View() + "\n\n"
	
	// Phone field
	phoneLabel := "Phone: "
	if m.newContactField == EditFieldPhone {
		phoneLabel = selectedStyle.Render(phoneLabel)
	}
	content += phoneLabel + m.newContactInputs[EditFieldPhone].View() + "\n\n"
	
	// Company field
	companyLabel := "Company: "
	if m.newContactField == EditFieldCompany {
		companyLabel = selectedStyle.Render(companyLabel)
	}
	content += companyLabel + m.newContactInputs[EditFieldCompany].View() + "\n\n"
	
	// Relationship type selector
	relLabel := "Relationship: "
	if m.newContactField == EditFieldRelType {
		relLabel = selectedStyle.Render(relLabel)
	}
	content += relLabel
	
	// Show relationship types with selection
	for i, rType := range RelationshipTypes[1:] { // Skip "all"
		if i == m.newContactRelTypeIdx {
			content += selectedStyle.Render(fmt.Sprintf("[%s]", rType)) + " "
		} else {
			content += fmt.Sprintf(" %s  ", rType)
		}
	}
	content += "\n\n"
	
	// Notes field
	notesLabel := "Notes: "
	if m.newContactField == EditFieldNotes {
		notesLabel = selectedStyle.Render(notesLabel)
	}
	content += notesLabel + m.newContactInputs[EditFieldNotes].View() + "\n\n"
	
	// Label field
	labelLabel := "Label: "
	if m.newContactField == EditFieldLabel {
		labelLabel = selectedStyle.Render(labelLabel)
	}
	content += labelLabel + m.newContactInputs[EditFieldLabel].View() + "\n\n"
	
	// Instructions
	content += lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("Tab/Shift+Tab: Navigate • Enter: Save • Esc: Cancel")
	
	// Create the box
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Width(width).
		Height(totalHeight).
		Padding(1).
		Render(content)
	
	// Center on screen
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}


// renderInteractionEditMode renders the interaction view/edit overlay
func (m Model) renderInteractionEditMode() string {
	width := 80
	height := 30
	
	content := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("32")).
		MarginBottom(1).
		Render("Interaction History") + "\n\n"
	
	// Show interactions
	for i, interaction := range m.interactions {
		// Date and type
		dateStr := interaction.InteractionDate.Format("2006-01-02 15:04")
		typeStr := fmt.Sprintf("[%s]", interaction.InteractionType)
		
		// Selection indicator
		var prefix string
		if i == m.selectedInteraction {
			prefix = selectedStyle.Render("> ")
		} else {
			prefix = "  "
		}
		
		content += prefix + dateStr + " " + typeStr + "\n"
		
		// Notes (indented)
		if interaction.Notes.Valid && interaction.Notes.String != "" {
			noteLines := wrapText(interaction.Notes.String, width-8)
			for _, line := range noteLines {
				content += "    " + line + "\n"
			}
		}
		content += "\n"
	}
	
	// If editing, show the edit textarea
	if m.interactionEditInput.Focused() {
		content += "\n" + lipgloss.NewStyle().
			Bold(true).
			Render("Editing - Type: " + InteractionTypes[m.interactionEditType]) + "\n"
		content += m.interactionEditInput.View() + "\n"
	}
	
	// Show delete confirmation if active
	if m.interactionDeleteConfirm {
		content += "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Render("Delete this interaction? (y/n)")
	}
	
	// Instructions
	var instructions string
	if m.interactionEditInput.Focused() {
		instructions = "Tab: change type • Ctrl+Enter: save • Esc: cancel"
	} else if m.interactionDeleteConfirm {
		instructions = "y: confirm delete • any key: cancel"
	} else {
		instructions = "j/k: navigate • e: edit • d: delete • Esc: exit"
	}
	
	content += "\n" + lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render(instructions)
	
	// Create scrollable view if content is too long
	contentHeight := strings.Count(content, "\n") + 1
	if contentHeight > height-4 {
		// Simple truncation for now - could implement proper scrolling later
		lines := strings.Split(content, "\n")
		visibleLines := height - 6
		if len(lines) > visibleLines {
			content = strings.Join(lines[:visibleLines], "\n") + "\n..."
		}
	}
	
	// Create the box
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Width(width).
		Height(height).
		Padding(1).
		Render(content)
	
	// Center on screen
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}
