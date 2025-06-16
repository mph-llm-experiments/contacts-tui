package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pdxmph/contacts-tui/internal/db"
	"github.com/pdxmph/contacts-tui/internal/tui"
)

func main() {
	// Open database
	database, err := db.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	
	// Create model
	model, err := tui.New(database)
	if err != nil {
		log.Fatal(err)
	}
	
	// Start the program
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
