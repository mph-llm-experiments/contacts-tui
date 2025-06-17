package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

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
		initDB      = flag.Bool("init", false, "Initialize database and configuration for first-time setup")
	)
	flag.Parse()
	
	// Handle init command
	if *initDB {
		if err := initializeSetup(); err != nil {
			log.Fatal("Error initializing:", err)
		}
		return
	}
	
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
	
	// Check if database exists
	if _, err := os.Stat(cfg.Database.Path); os.IsNotExist(err) {
		fmt.Printf("Database not found at %s\n", cfg.Database.Path)
		fmt.Println("\nTo initialize the database and configuration, run:")
		fmt.Println("  contacts-tui -init")
		os.Exit(1)
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

func initializeSetup() error {
	fmt.Println("Initializing contacts-tui...")
	
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}
	
	// Create config directory
	configDir := filepath.Join(homeDir, ".config", "contacts")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	fmt.Printf("✓ Created config directory: %s\n", configDir)
	
	// Check if config file exists
	configPath := filepath.Join(configDir, "config.toml")
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("✓ Config file already exists: %s\n", configPath)
	} else {
		// Create default config file
		cfg := config.Default()
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		fmt.Printf("✓ Created config file: %s\n", configPath)
	}
	
	// Load config to get database path
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	
	// Check if database exists
	dbPath := cfg.Database.Path
	if _, err := os.Stat(dbPath); err == nil {
		fmt.Printf("✗ Database already exists at: %s\n", dbPath)
		fmt.Println("\nTo start fresh, delete the existing database and run -init again.")
		return nil
	}
	
	// Initialize database
	if err := db.Initialize(dbPath); err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}
	fmt.Printf("✓ Created database: %s\n", dbPath)
	
	// Add sample contact
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()
	
	sampleContact := db.Contact{
		Name:             "Sample Contact",
		Email:            db.NewNullString("sample@example.com"),
		Phone:            db.NewNullString("555-0123"),
		Company:          db.NewNullString("Example Corp"),
		RelationshipType: "network",
		State:            db.NewNullString("ok"),
		Notes:            db.NewNullString("This is a sample contact. Feel free to edit or delete it using the 'e' or 'D' keys."),
		Label:            db.NewNullString("@sample"),
	}
	
	_, err = database.AddContact(sampleContact)
	if err != nil {
		return fmt.Errorf("adding sample contact: %w", err)
	}
	fmt.Println("✓ Added sample contact")
	
	fmt.Println("\nInitialization complete! You can now run:")
	fmt.Println("  contacts-tui")
	
	return nil
}
