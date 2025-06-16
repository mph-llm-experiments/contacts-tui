package tui

import (
	"fmt"
	"strings"
	"time"

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
	filter     textinput.Model
	err        error
}

// Styles
var (
	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230"))
	
	overdueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
	
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
	
	return &Model{
		db:       database,
		contacts: contacts,
		filter:   ti,
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
		return m, nil
		
	case tea.KeyMsg:
		// Filter mode handling
		if m.filterMode {
			switch msg.String() {
			case "esc":
				m.filterMode = false
				m.filter.Reset()
				return m, nil
			case "enter":
				m.filterMode = false
				return m, nil
			}
			
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			return m, cmd
		}
		// Normal mode handling
		switch msg.String() {
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
			m.filter.Focus()
			return m, textinput.Blink
			
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
					}
				}
			}
		}
	}
	
	return m, nil
}

// filteredContacts returns contacts matching the current filter
func (m Model) filteredContacts() []db.Contact {
	if m.filter.Value() == "" {
		return m.contacts
	}
	
	filter := strings.ToLower(m.filter.Value())
	var filtered []db.Contact
	
	for _, c := range m.contacts {
		if strings.Contains(strings.ToLower(c.Name), filter) ||
		   (c.Label.Valid && strings.Contains(strings.ToLower(c.Label.String), filter)) ||
		   (c.Company.Valid && strings.Contains(strings.ToLower(c.Company.String), filter)) {
			filtered = append(filtered, c)
		}
	}
	
	return filtered
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
	
	return lipgloss.JoinVertical(lipgloss.Left, content, help)
}

// renderList renders the contact list
func (m Model) renderList(width, height int) string {
	var lines []string
	
	if m.filterMode {
		lines = append(lines, m.filter.View())
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
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", width-2))
	
	// Contact list
	for i := startIdx; i < len(contacts) && i < startIdx+visibleHeight; i++ {
		c := contacts[i]
		
		// Build the display line
		var line string
		
		// Add overdue indicator or spacing
		if c.IsOverdue() {
			line = "* "
		} else {
			line = "  "
		}
		
		// Add name
		line += c.Name
		
		// Add label if present
		if c.Label.Valid {
			line += " " + labelStyle.Render("["+c.Label.String+"]")
		}
		
		// Apply selection styling to entire line if selected
		if i == m.selected {
			line = selectedStyle.Width(width-2).Render(line)
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
	
	lines = append(lines, "")
	
	// Notes
	if c.Notes.Valid && c.Notes.String != "" {
		lines = append(lines, "Notes:")
		lines = append(lines, c.Notes.String)
	}
	
	return strings.Join(lines, "\n")
}

// renderHelp renders the help line
func (m Model) renderHelp() string {
	if m.filterMode {
		return " Type to filter • Enter to confirm • Esc to cancel"
	}
	
	return " j/k: navigate • /: filter • c: mark contacted • q: quit"
}