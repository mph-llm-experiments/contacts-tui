package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

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