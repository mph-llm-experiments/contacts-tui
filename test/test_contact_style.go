package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	
	"github.com/pdxmph/contacts-tui/internal/db"
)

func main() {
	// Use test database
	dbPath := filepath.Join(os.Getenv("HOME"), ".config", "contacts", "contacts.db")
	
	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	
	// Get first contact
	contacts, err := database.ListContacts()
	if err != nil {
		log.Fatal(err)
	}
	
	if len(contacts) == 0 {
		fmt.Println("No contacts found")
		return
	}
	
	contact := contacts[0]
	fmt.Printf("Testing contact style for: %s\n", contact.Name)
	fmt.Printf("Current style: %s\n", contact.ContactStyle)
	if contact.CustomFrequencyDays.Valid {
		fmt.Printf("Custom frequency: %d days\n", contact.CustomFrequencyDays.Int64)
	}
	
	// Test UpdateContactStyle
	fmt.Println("\nTesting UpdateContactStyle...")
	
	// Set to ambient
	err = database.UpdateContactStyle(contact.ID, "ambient", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Set to ambient")
	
	// Set to periodic with custom frequency
	days := 45
	err = database.UpdateContactStyle(contact.ID, "periodic", &days)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Set to periodic with 45 days")
	
	// Verify changes
	contacts, err = database.ListContacts()
	if err != nil {
		log.Fatal(err)
	}
	
	contact = contacts[0]
	fmt.Printf("\nFinal state:\n")
	fmt.Printf("Style: %s\n", contact.ContactStyle)
	if contact.CustomFrequencyDays.Valid {
		fmt.Printf("Custom frequency: %d days\n", contact.CustomFrequencyDays.Int64)
	}
	fmt.Printf("Is overdue: %v\n", contact.IsOverdue())
}
