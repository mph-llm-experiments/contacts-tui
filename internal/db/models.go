package db

import (
	"database/sql"
	"time"
)

// Contact represents a person in the database
type Contact struct {
	ID                   int
	Name                 string
	Email                sql.NullString
	Phone                sql.NullString
	Company              sql.NullString
	RelationshipType     string
	State                sql.NullString
	Notes                sql.NullString
	Label                sql.NullString
	BasicMemoryURL       sql.NullString
	ContactedAt          sql.NullTime
	LastBumpDate         sql.NullTime
	BumpCount            int
	FollowUpDate         sql.NullTime
	DeadlineDate         sql.NullTime
	Archived             bool
	ArchivedAt           sql.NullTime
	ContactStyle         string
	CustomFrequencyDays  sql.NullInt64
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// Log represents an interaction log entry
type Log struct {
	ID              int
	ContactID       int
	InteractionDate time.Time
	InteractionType string
	Notes           sql.NullString
	CreatedAt       time.Time
}

// IsOverdue checks if a contact is overdue based on relationship type and contact style
func (c Contact) IsOverdue() bool {
	// Archived contacts are never overdue
	if c.Archived {
		return false
	}
	
	// Ambient and triggered contacts are never overdue
	if c.ContactStyle == "ambient" || c.ContactStyle == "triggered" {
		return false
	}
	
	// Get the most recent interaction date (either contacted or bumped)
	var lastInteraction sql.NullTime
	
	if c.ContactedAt.Valid && c.LastBumpDate.Valid {
		// Use whichever is more recent
		if c.ContactedAt.Time.After(c.LastBumpDate.Time) {
			lastInteraction = c.ContactedAt
		} else {
			lastInteraction = c.LastBumpDate
		}
	} else if c.ContactedAt.Valid {
		lastInteraction = c.ContactedAt
	} else if c.LastBumpDate.Valid {
		lastInteraction = c.LastBumpDate
	}
	
	if !lastInteraction.Valid {
		return true // Never contacted or bumped
	}
	
	daysSince := time.Since(lastInteraction.Time).Hours() / 24
	
	// Use custom frequency if set
	if c.CustomFrequencyDays.Valid && c.CustomFrequencyDays.Int64 > 0 {
		return daysSince > float64(c.CustomFrequencyDays.Int64)
	}
	
	// Otherwise use relationship type defaults
	switch c.RelationshipType {
	case "close", "family":
		return daysSince > 30
	case "network":
		return daysSince > 90
	default:
		return daysSince > 60
	}
}

// NewNullString creates a sql.NullString from a string
func NewNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
