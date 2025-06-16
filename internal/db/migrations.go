package db

import (
	"fmt"
	"log"
)

// RunMigrations applies any pending database migrations
func (db *DB) RunMigrations() error {
	// Run bump columns migration
	if err := db.runBumpMigration(); err != nil {
		return err
	}
	
	// Run archive columns migration
	if err := db.runArchiveMigration(); err != nil {
		return err
	}
	
	return nil
}

func (db *DB) runBumpMigration() error {
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

func (db *DB) runArchiveMigration() error {
	// Check if archive columns exist
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('contacts') 
		WHERE name IN ('archived', 'archived_at')
	`).Scan(&count)
	
	if err != nil {
		return fmt.Errorf("checking for archive columns: %w", err)
	}
	
	// If columns don't exist, add them
	if count < 2 {
		log.Println("Running migration: Adding archive functionality columns...")
		
		tx, err := db.conn.Begin()
		if err != nil {
			return fmt.Errorf("starting transaction: %w", err)
		}
		defer tx.Rollback()
		
		// Add archived column
		_, err = tx.Exec(`ALTER TABLE contacts ADD COLUMN archived BOOLEAN DEFAULT 0`)
		if err != nil && err.Error() != "duplicate column name: archived" {
			return fmt.Errorf("adding archived column: %w", err)
		}
		
		// Add archived_at column
		_, err = tx.Exec(`ALTER TABLE contacts ADD COLUMN archived_at TIMESTAMP`)
		if err != nil && err.Error() != "duplicate column name: archived_at" {
			return fmt.Errorf("adding archived_at column: %w", err)
		}
		
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing archive migration: %w", err)
		}
		
		log.Println("Archive migration completed successfully")
	}
	
	return nil
}