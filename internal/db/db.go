package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the database connection
type DB struct {
	conn *sql.DB
}

// Open creates a new database connection
func Open() (*DB, error) {
	// Default to ~/.config/contacts/contacts.db
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home dir: %w", err)
	}
	
	dbPath := filepath.Join(homeDir, ".config", "contacts", "contacts.db")
	
	// Check if DB exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("database not found at %s", dbPath)
	}
	
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	
	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}// ListContacts returns all contacts ordered by name
func (db *DB) ListContacts() ([]Contact, error) {
	query := `
		SELECT 
			id, name, email, phone, company, 
			relationship_type, state, notes, label,
			basic_memory_url, contacted_at,
			follow_up_date, deadline_date,
			created_at, updated_at
		FROM contacts
		ORDER BY name
	`
	
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("querying contacts: %w", err)
	}
	defer rows.Close()
	
	var contacts []Contact
	for rows.Next() {
		var c Contact
		err := rows.Scan(
			&c.ID, &c.Name, &c.Email, &c.Phone, &c.Company,
			&c.RelationshipType, &c.State, &c.Notes, &c.Label,
			&c.BasicMemoryURL, &c.ContactedAt,
			&c.FollowUpDate, &c.DeadlineDate,
			&c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning contact: %w", err)
		}
		
		// Clean up the name field - remove newlines and trim whitespace
		c.Name = strings.TrimSpace(strings.ReplaceAll(c.Name, "\n", " "))
		
		contacts = append(contacts, c)
	}
	
	return contacts, rows.Err()
}
// MarkContacted marks a contact as contacted with today's date
func (db *DB) MarkContacted(contactID int, interactionType string, notes string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Update contact's contacted_at
	updateQuery := `UPDATE contacts SET contacted_at = CURRENT_TIMESTAMP WHERE id = ?`
	if _, err := tx.Exec(updateQuery, contactID); err != nil {
		return fmt.Errorf("updating contact: %w", err)
	}
	
	// Insert interaction log
	logQuery := `
		INSERT INTO contact_interactions (contact_id, interaction_date, interaction_type, notes)
		VALUES (?, CURRENT_TIMESTAMP, ?, ?)
	`
	if _, err := tx.Exec(logQuery, contactID, interactionType, notes); err != nil {
		return fmt.Errorf("inserting interaction log: %w", err)
	}
	
	return tx.Commit()
}

// GetContact retrieves a single contact by ID
func (db *DB) GetContact(id int) (*Contact, error) {
	query := `
		SELECT 
			id, name, email, phone, company, 
			relationship_type, state, notes, label,
			basic_memory_url, contacted_at,
			follow_up_date, deadline_date,
			created_at, updated_at
		FROM contacts
		WHERE id = ?
	`
	
	var c Contact
	err := db.conn.QueryRow(query, id).Scan(
		&c.ID, &c.Name, &c.Email, &c.Phone, &c.Company,
		&c.RelationshipType, &c.State, &c.Notes, &c.Label,
		&c.BasicMemoryURL, &c.ContactedAt,
		&c.FollowUpDate, &c.DeadlineDate,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	
	return &c, nil
}

// UpdateContactState updates the state of a contact
func (db *DB) UpdateContactState(contactID int, state string) error {
	query := `UPDATE contacts SET state = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.conn.Exec(query, state, contactID)
	if err != nil {
		return fmt.Errorf("updating contact state: %w", err)
	}
	return nil
}

// AddInteractionNote adds a note without updating contacted_at
func (db *DB) AddInteractionNote(contactID int, interactionType string, notes string) error {
	if notes == "" {
		return fmt.Errorf("notes cannot be empty")
	}
	
	query := `
		INSERT INTO contact_interactions (contact_id, interaction_date, interaction_type, notes)
		VALUES (?, CURRENT_TIMESTAMP, ?, ?)
	`
	_, err := db.conn.Exec(query, contactID, interactionType, notes)
	if err != nil {
		return fmt.Errorf("inserting interaction note: %w", err)
	}
	
	return nil
}

// GetContactInteractions retrieves recent interaction logs for a contact
func (db *DB) GetContactInteractions(contactID int, limit int) ([]Log, error) {
	query := `
		SELECT 
			id, contact_id, interaction_date, interaction_type, notes, created_at
		FROM contact_interactions
		WHERE contact_id = ?
		ORDER BY interaction_date DESC
		LIMIT ?
	`
	
	rows, err := db.conn.Query(query, contactID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying interactions: %w", err)
	}
	defer rows.Close()
	
	var logs []Log
	for rows.Next() {
		var l Log
		err := rows.Scan(
			&l.ID, &l.ContactID, &l.InteractionDate, 
			&l.InteractionType, &l.Notes, &l.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning log: %w", err)
		}
		logs = append(logs, l)
	}
	
	return logs, rows.Err()
}