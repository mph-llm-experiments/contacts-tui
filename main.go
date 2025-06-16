package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pdxmph/contacts-tui/internal/config"
	"github.com/pdxmph/contacts-tui/internal/db"
	"github.com/pdxmph/contacts-tui/internal/tui"
)

func main() {
	// Parse command line flags
	var (
		writeConfig = flag.Bool("write-config", false, "Write default configuration file")
		showConfig  = flag.Bool("show-config", false, "Show current configuration")
	)
	flag.Parse()
	
	// Handle config commands
	if *writeConfig {
		if err := writeDefaultConfig(); err != nil {
			log.Fatal("Error writing config:", err)
		}
		return
	}
	
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Error loading config:", err)
	}
	
	if *showConfig {
		fmt.Println("Current configuration:")
		fmt.Printf("Database path: %s\n", cfg.Database.Path)
		return
	}
	
	// Open database
	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	
	// Run migrations
	if err := database.RunMigrations(); err != nil {
		log.Fatal("Error running migrations:", err)
	}
	
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

func writeDefaultConfig() error {
	cfg := config.Default()
	if err := cfg.Save(); err != nil {
		return err
	}
	
	homeDir, _ := os.UserHomeDir()
	fmt.Printf("Configuration file written to: %s/.config/contacts/config.toml\n", homeDir)
	fmt.Printf("Default database path: %s\n", cfg.Database.Path)
	fmt.Println("\nYou can now edit this file to customize your database location.")
	return nil
}
