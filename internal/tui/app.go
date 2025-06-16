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
	
	// Smart filters
	stateFilter   bool // Show only non-ok states
	overdueFilter bool // Show only overdue contacts
	typeFilter    string // Filter by relationship type
	
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
	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230"))
	
	overdueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
	
	stateStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")) // Orange for states
	
	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
	
	borderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))
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
	
	return &Model{
		db:         database,
		contacts:   contacts,
		filter:     ti,
		noteInput:  ta,
		editInputs: editInputs,
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
		// Normal mode handling
		switch msg.String() {
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
				
				// Set width for edit inputs based on detail pane
				if m.width > 0 {
					detailWidth := m.width - (m.width / 3) - 10
					for i := range m.editInputs {
						m.editInputs[i].Width = detailWidth - 20
					}
				}
				
				return m, textinput.Blink
			}
		}
	}
	
	return m, nil
}

// filteredContacts returns contacts matching the current filter
func (m Model) filteredContacts() []db.Contact {
	var filtered []db.Contact
	
	// Start with all contacts
	contacts := m.contacts
	
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
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
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
	
	// Overlay bump confirmation if active
	if m.bumpConfirmMode {
		return m.renderBumpConfirmation()
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
	if len(filterIndicators) > 0 {
		header += " [" + strings.Join(filterIndicators, ", ") + "]"
	}
	
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", width-2))
	
	// Contact list
	for i := startIdx; i < len(contacts) && i < startIdx+visibleHeight; i++ {
		c := contacts[i]
		
		// Build the display line
		var line string
		
		// Add overdue indicator or state indicator or spacing
		if c.IsOverdue() {
			line = "* "
		} else if c.State.Valid && c.State.String != "ok" {
			line = stateStyle.Render("•") + " "
		} else {
			line = "  "
		}
		
		// Add name
		line += c.Name
		
		// Add label if present
		if c.Label.Valid {
			// Clean up label too - remove newlines
			label := strings.TrimSpace(strings.ReplaceAll(c.Label.String, "\n", " "))
			line += " " + labelStyle.Render("["+label+"]")
		}
		
		// Apply selection styling to entire line if selected
		if i == m.selected {
			line = selectedStyle.Render(line)
		} else if c.IsOverdue() {
			// For non-selected overdue contacts, just color the asterisk
			line = overdueStyle.Render("*") + line[1:]
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
	if m.bumpConfirmMode {
		return " y: confirm bump • any other key: cancel"
	}
	
	if m.typeFilterMode {
		return " j/k: navigate • Enter: select • Esc: cancel"
	}
	
	if m.stateMode {
		return " j/k: navigate • Enter: confirm • Esc: cancel"
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
	
	help := " j/k: navigate • /: filter • c: contacted • b: bump • e: edit • s: state • n: note"
	
	// Add smart filter shortcuts
	help += " • S: state • o: overdue • r: type"
	
	// Show clear option if any filters are active
	if m.stateFilter || m.overdueFilter || m.typeFilter != "" || m.filter.Value() != "" {
		help += " • C: clear all"
	}
	
	if m.filter.Value() != "" {
		help += " • Esc: clear filter"
	}
	
	help += " • q: quit"
	
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
	
	for i, state := range ContactStates {
		line := fmt.Sprintf("  %s", state)
		if i == m.stateSelected {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)
	}
	
	lines = append(lines, "")
	lines = append(lines, "Press Enter to confirm, Esc to cancel")
	
	// Create a bordered box and center it
	content := strings.Join(lines, "\n")
	box := borderStyle.
		Padding(1).
		Background(lipgloss.Color("235")).
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
			typeSelector += selectedStyle.Render(fmt.Sprintf("[%s]", iType)) + " "
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
		Background(lipgloss.Color("235")).
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
	
	for i, rType := range RelationshipTypes {
		line := fmt.Sprintf("  %s", rType)
		if rType == "all" {
			line = "  all (clear filter)"
		}
		if i == m.typeSelected {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)
	}
	
	lines = append(lines, "")
	lines = append(lines, "Press Enter to confirm, Esc to cancel")
	
	// Create a bordered box and center it
	content := strings.Join(lines, "\n")
	box := borderStyle.
		Padding(1).
		Background(lipgloss.Color("235")).
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
		Background(lipgloss.Color("235")).
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