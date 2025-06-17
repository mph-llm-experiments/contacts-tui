package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the database connection
type DB struct {
	conn *sql.DB
}

// Open creates a new database connection
func Open(dbPath string) (*DB, error) {
	// Check if DB exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("database not found at %s\nRun 'contacts-tui -init' to create it", dbPath)
	}
	
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	
	db := &DB{conn: conn}
	
	// Run any pending migrations
	if err := db.RunMigrations(); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	
	return db, nil
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
			basic_memory_url, contacted_at, last_bump_date, bump_count,
			follow_up_date, deadline_date,
			archived, archived_at,
			contact_style, custom_frequency_days,
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
			&c.BasicMemoryURL, &c.ContactedAt, &c.LastBumpDate, &c.BumpCount,
			&c.FollowUpDate, &c.DeadlineDate,
			&c.Archived, &c.ArchivedAt,
			&c.ContactStyle, &c.CustomFrequencyDays,
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
			basic_memory_url, contacted_at, last_bump_date, bump_count,
			follow_up_date, deadline_date,
			archived, archived_at,
			contact_style, custom_frequency_days,
			created_at, updated_at
		FROM contacts
		WHERE id = ?
	`
	
	var c Contact
	err := db.conn.QueryRow(query, id).Scan(
		&c.ID, &c.Name, &c.Email, &c.Phone, &c.Company,
		&c.RelationshipType, &c.State, &c.Notes, &c.Label,
		&c.BasicMemoryURL, &c.ContactedAt, &c.LastBumpDate, &c.BumpCount,
		&c.FollowUpDate, &c.DeadlineDate,
		&c.Archived, &c.ArchivedAt,
		&c.ContactStyle, &c.CustomFrequencyDays,
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

// UpdateContact updates all fields of a contact
func (db *DB) UpdateContact(contact Contact) error {
	query := `
		UPDATE contacts 
		SET name = ?, 
		    email = ?, 
		    phone = ?, 
		    company = ?, 
		    relationship_type = ?, 
		    notes = ?, 
		    label = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	
	_, err := db.conn.Exec(query, 
		contact.Name,
		contact.Email,
		contact.Phone,
		contact.Company,
		contact.RelationshipType,
		contact.Notes,
		contact.Label,
		contact.ID,
	)
	
	if err != nil {
		return fmt.Errorf("updating contact: %w", err)
	}
	
	return nil
}

// BumpContact updates the bump date and increments bump count
func (db *DB) BumpContact(contactID int) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Update contact's bump date and increment count
	updateQuery := `
		UPDATE contacts 
		SET last_bump_date = CURRENT_TIMESTAMP,
		    bump_count = bump_count + 1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	if _, err := tx.Exec(updateQuery, contactID); err != nil {
		return fmt.Errorf("updating contact: %w", err)
	}
	
	// Insert interaction log
	logQuery := `
		INSERT INTO contact_interactions (contact_id, interaction_date, interaction_type, notes)
		VALUES (?, CURRENT_TIMESTAMP, 'bump', 'Contact reviewed and bumped')
	`
	if _, err := tx.Exec(logQuery, contactID); err != nil {
		return fmt.Errorf("inserting bump log: %w", err)
	}
	
	return tx.Commit()
}
// ArchiveContact archives a contact
func (db *DB) ArchiveContact(contactID int) error {
	query := `
		UPDATE contacts 
		SET archived = 1,
		    archived_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := db.conn.Exec(query, contactID)
	if err != nil {
		return fmt.Errorf("archiving contact: %w", err)
	}
	return nil
}

// UnarchiveContact unarchives a contact
func (db *DB) UnarchiveContact(contactID int) error {
	query := `
		UPDATE contacts 
		SET archived = 0,
		    archived_at = NULL,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := db.conn.Exec(query, contactID)
	if err != nil {
		return fmt.Errorf("unarchiving contact: %w", err)
	}
	return nil
}

// DeleteContact permanently deletes a contact and all associated logs
func (db *DB) DeleteContact(contactID int) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Delete interaction logs first (foreign key constraint)
	_, err = tx.Exec(`DELETE FROM contact_interactions WHERE contact_id = ?`, contactID)
	if err != nil {
		return fmt.Errorf("deleting interaction logs: %w", err)
	}
	
	// Delete the contact
	_, err = tx.Exec(`DELETE FROM contacts WHERE id = ?`, contactID)
	if err != nil {
		return fmt.Errorf("deleting contact: %w", err)
	}
	
	return tx.Commit()
}

// AddContact creates a new contact in the database
func (db *DB) AddContact(contact Contact) (int64, error) {
	query := `
		INSERT INTO contacts (
			name, email, phone, company, 
			relationship_type, state, notes, label,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	
	result, err := db.conn.Exec(query,
		contact.Name,
		contact.Email,
		contact.Phone,
		contact.Company,
		contact.RelationshipType,
		contact.State,
		contact.Notes,
		contact.Label,
	)
	
	if err != nil {
		return 0, fmt.Errorf("inserting contact: %w", err)
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting insert ID: %w", err)
	}
	
	return id, nil
}

// UpdateInteraction updates an existing interaction
func (db *DB) UpdateInteraction(interactionID int, interactionType string, notes string) error {
	query := `
		UPDATE contact_interactions 
		SET interaction_type = ?, notes = ?
		WHERE id = ?
	`
	_, err := db.conn.Exec(query, interactionType, notes, interactionID)
	if err != nil {
		return fmt.Errorf("updating interaction: %w", err)
	}
	return nil
}

// DeleteInteraction deletes an interaction by ID
func (db *DB) DeleteInteraction(interactionID int) error {
	query := `DELETE FROM contact_interactions WHERE id = ?`
	_, err := db.conn.Exec(query, interactionID)
	if err != nil {
		return fmt.Errorf("deleting interaction: %w", err)
	}
	return nil
}

// UpdateContactStyle updates the contact style and custom frequency
func (db *DB) UpdateContactStyle(contactID int, style string, customFrequencyDays *int) error {
	var query string
	var args []interface{}
	
	if customFrequencyDays != nil {
		query = `UPDATE contacts SET contact_style = ?, custom_frequency_days = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
		args = []interface{}{style, *customFrequencyDays, contactID}
	} else {
		query = `UPDATE contacts SET contact_style = ?, custom_frequency_days = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
		args = []interface{}{style, contactID}
	}
	
	_, err := db.conn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("updating contact style: %w", err)
	}
	return nil
}
