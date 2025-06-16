package db

import (
	"fmt"
	"log"
)

// RunMigrations applies any pending database migrations
func (db *DB) RunMigrations() error {
	// Check if bump columns exist
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('contacts') 
		WHERE name IN ('last_bump_date', 'bump_count')
	`).Scan(&count)
	
	if err != nil {
		return fmt.Errorf("checking for bump columns: %w", err)
	}
	
	// If columns don't exist, add them
	if count < 2 {
		log.Println("Running migration: Adding bump functionality columns...")
		
		tx, err := db.conn.Begin()
		if err != nil {
			return fmt.Errorf("starting transaction: %w", err)
		}
		defer tx.Rollback()
		
		// Add last_bump_date column
		_, err = tx.Exec(`ALTER TABLE contacts ADD COLUMN last_bump_date TIMESTAMP`)
		if err != nil && err.Error() != "duplicate column name: last_bump_date" {
			return fmt.Errorf("adding last_bump_date column: %w", err)
		}
		
		// Add bump_count column
		_, err = tx.Exec(`ALTER TABLE contacts ADD COLUMN bump_count INTEGER DEFAULT 0`)
		if err != nil && err.Error() != "duplicate column name: bump_count" {
			return fmt.Errorf("adding bump_count column: %w", err)
		}
		
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing migration: %w", err)
		}
		
		log.Println("Migration completed successfully")
	}
	
	return nil
}