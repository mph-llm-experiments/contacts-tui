package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	
	_ "github.com/mattn/go-sqlite3"
)

// Initialize creates a new database with the complete schema
func Initialize(dbPath string) error {
	// Check if database already exists
	if _, err := os.Stat(dbPath); err == nil {
		return fmt.Errorf("database already exists at %s", dbPath)
	}
	
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating database directory: %w", err)
	}
	
	// Create database file
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("creating database: %w", err)
	}
	defer db.Close()
	
	// Create schema
	schema := `
-- Contact MCP Database Schema
CREATE TABLE IF NOT EXISTS contacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    external_id TEXT UNIQUE,
    source TEXT NOT NULL DEFAULT 'manual',
    name TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    company TEXT,
    notes TEXT,
    relationship_type TEXT CHECK (relationship_type IN ('close', 'family', 'network', 'social', 'providers', 'recruiters', 'work')) NOT NULL DEFAULT 'network',
    contacted_at DATE,
    state TEXT CHECK (state IN ('ping', 'invite', 'write', 'pinged', 'followup', 'sked', 'notes', 'scheduled', 'timeout', 'ok')),
    follow_up_date DATE,
    deadline_date DATE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    synced_at DATETIME,
    label TEXT,
    basic_memory_url TEXT,
    -- Bump functionality columns
    last_bump_date TIMESTAMP,
    bump_count INTEGER DEFAULT 0,
    -- Archive functionality columns
    archived BOOLEAN DEFAULT 0,
    archived_at TIMESTAMP,
    -- Contact style columns
    contact_style TEXT DEFAULT 'periodic',
    custom_frequency_days INTEGER
);

CREATE TABLE IF NOT EXISTS contact_interactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    contact_id INTEGER NOT NULL,
    interaction_type TEXT NOT NULL,
    interaction_date DATE NOT NULL,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (contact_id) REFERENCES contacts (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS log_contacts (
    log_id INTEGER NOT NULL,
    contact_id INTEGER NOT NULL,
    PRIMARY KEY (log_id, contact_id),
    FOREIGN KEY (log_id) REFERENCES logs(id) ON DELETE CASCADE,
    FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_contacts_relationship_type ON contacts (relationship_type);
CREATE INDEX IF NOT EXISTS idx_contacts_contacted_at ON contacts (contacted_at);
CREATE INDEX IF NOT EXISTS idx_contacts_state ON contacts (state);
CREATE INDEX IF NOT EXISTS idx_contacts_label ON contacts (label);
CREATE INDEX IF NOT EXISTS idx_contacts_relationship_contacted ON contacts(relationship_type, contacted_at);
CREATE INDEX IF NOT EXISTS idx_contacts_search ON contacts(name, email, company, label);
CREATE INDEX IF NOT EXISTS idx_interactions_contact_date ON contact_interactions(contact_id, interaction_date DESC);
CREATE INDEX IF NOT EXISTS idx_logs_content ON logs(content);
CREATE INDEX IF NOT EXISTS idx_log_contacts_contact ON log_contacts(contact_id);
CREATE INDEX IF NOT EXISTS idx_logs_created_at ON logs(created_at DESC);

-- Triggers for timestamp updates
CREATE TRIGGER IF NOT EXISTS update_contact_timestamp 
AFTER UPDATE ON contacts
BEGIN
    UPDATE contacts SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_log_timestamp 
AFTER UPDATE ON logs
BEGIN
    UPDATE logs SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;`
	
	// Execute schema
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("creating schema: %w", err)
	}
	
	return nil
}
