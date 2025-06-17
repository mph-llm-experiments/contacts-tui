package db

import (
	"database/sql"
	"fmt"
	"time"
)

// CreateFixturesDatabase creates a test database with realistic sample data
func CreateFixturesDatabase(dbPath string) error {
	// Initialize empty database
	if err := Initialize(dbPath); err != nil {
		return fmt.Errorf("initializing fixtures database: %w", err)
	}
	
	// Open database to add test data
	database, err := Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening fixtures database: %w", err)
	}
	defer database.Close()
	
	// Add fixture contacts
	fixtures := []Contact{
		// Close relationships
		{
			Name:             "Sarah Chen",
			Email:            NewNullString("sarah.chen@email.com"),
			Phone:            NewNullString("555-0101"),
			Company:          NewNullString("Tech Startup Inc"),
			RelationshipType: "close",
			State:            NewNullString("ok"),
			Notes:            NewNullString("Close friend from college, now working at a promising startup. Great for product feedback."),
			Label:            NewNullString("@sarahc"),
			ContactedAt:      sql.NullTime{Time: time.Now().AddDate(0, 0, -15), Valid: true},
			ContactStyle:     "periodic",
		},
		{
			Name:             "Marcus Williams",
			Email:            NewNullString("marcus.w@company.com"),
			Phone:            NewNullString("555-0102"),
			Company:          NewNullString("Design Studio"),
			RelationshipType: "close",
			State:            NewNullString("ping"),
			Notes:            NewNullString("Talented designer and close collaborator. Always has interesting project ideas."),
			Label:            NewNullString("@marcusw"),
			ContactedAt:      sql.NullTime{Time: time.Now().AddDate(0, 0, -35), Valid: true},
			ContactStyle:     "periodic",
		},
		
		// Family
		{
			Name:             "Mom",
			Email:            NewNullString("mom@family.com"),
			Phone:            NewNullString("555-0103"),
			RelationshipType: "family",
			State:            NewNullString("ok"),
			Notes:            NewNullString("Regular check-ins, usually Sunday calls"),
			Label:            NewNullString("@mom"),
			ContactedAt:      sql.NullTime{Time: time.Now().AddDate(0, 0, -7), Valid: true},
			ContactStyle:     "periodic",
		},
		{
			Name:             "Alex Thompson",
			Email:            NewNullString("alex.thompson@email.com"),
			Phone:            NewNullString("555-0104"),
			RelationshipType: "family",
			State:            NewNullString("followup"),
			Notes:            NewNullString("Cousin in Seattle, software engineer. Mentioned wanting to connect about career advice."),
			Label:            NewNullString("@alext"),
			ContactedAt:      sql.NullTime{Time: time.Now().AddDate(0, 0, -45), Valid: true},
			ContactStyle:     "periodic",
		},
		
		// Work relationships
		{
			Name:             "Jennifer Rodriguez",
			Email:            NewNullString("jen.rodriguez@company.com"),
			Phone:            NewNullString("555-0105"),
			Company:          NewNullString("Big Corp Ltd"),
			RelationshipType: "work",
			State:            NewNullString("ok"),
			Notes:            NewNullString("Product manager at Big Corp, excellent collaboration on API project"),
			Label:            NewNullString("@jenr"),
			ContactedAt:      sql.NullTime{Time: time.Now().AddDate(0, 0, -10), Valid: true},
			ContactStyle:     "triggered",
		},
		{
			Name:             "David Kim",
			Email:            NewNullString("d.kim@startup.io"),
			Phone:            NewNullString("555-0106"),
			Company:          NewNullString("AI Startup"),
			RelationshipType: "work",
			State:            NewNullString("sked"),
			Notes:            NewNullString("CTO at promising AI startup, interested in potential partnership"),
			Label:            NewNullString("@davidk"),
			ContactedAt:      sql.NullTime{Time: time.Now().AddDate(0, 0, -20), Valid: true},
			ContactStyle:     "periodic",
			FollowUpDate:     sql.NullTime{Time: time.Now().AddDate(0, 0, 5), Valid: true},
		},
		
		// Network relationships
		{
			Name:             "Lisa Park",
			Email:            NewNullString("lisa.park@consulting.com"),
			Phone:            NewNullString("555-0107"),
			Company:          NewNullString("Strategy Consulting"),
			RelationshipType: "network",
			State:            NewNullString("ok"),
			Notes:            NewNullString("Met at tech conference, works in strategy consulting. Good connection for business insights."),
			Label:            NewNullString("@lisap"),
			ContactedAt:      sql.NullTime{Time: time.Now().AddDate(0, 0, -60), Valid: true},
			ContactStyle:     "periodic",
		},
		{
			Name:             "Robert Martinez",
			Email:            NewNullString("r.martinez@venture.capital"),
			Company:          NewNullString("VC Firm"),
			RelationshipType: "network",
			State:            NewNullString("write"),
			Notes:            NewNullString("Venture capitalist, interested in developer tools. Potential investor contact."),
			Label:            NewNullString("@robertm"),
			ContactedAt:      sql.NullTime{Time: time.Now().AddDate(0, 0, -120), Valid: true},
			ContactStyle:     "periodic",
		},
		{
			Name:             "Emily Zhang",
			Email:            NewNullString("emily.zhang@freelance.com"),
			Phone:            NewNullString("555-0108"),
			Company:          NewNullString("Freelance"),
			RelationshipType: "network",
			State:            NewNullString("ping"),
			Notes:            NewNullString("Freelance UX designer, excellent portfolio. Could be good for design projects."),
			Label:            NewNullString("@emilyz"),
			ContactedAt:      sql.NullTime{Time: time.Now().AddDate(0, 0, -95), Valid: true},
			ContactStyle:     "periodic",
		},
		
		// Social relationships
		{
			Name:             "Mike Johnson",
			Email:            NewNullString("mike.j@social.com"),
			Phone:            NewNullString("555-0109"),
			RelationshipType: "social",
			State:            NewNullString("ok"),
			Notes:            NewNullString("Friend from hiking group, always up for outdoor adventures"),
			Label:            NewNullString("@mikej"),
			ContactedAt:      sql.NullTime{Time: time.Now().AddDate(0, 0, -25), Valid: true},
			ContactStyle:     "ambient",
		},
		{
			Name:             "Rachel Green",
			Email:            NewNullString("rachel.green@book.club"),
			RelationshipType: "social",
			State:            NewNullString("ok"),
			Notes:            NewNullString("Book club organizer, great recommendations for business and fiction books"),
			Label:            NewNullString("@rachelg"),
			ContactedAt:      sql.NullTime{Time: time.Now().AddDate(0, 0, -40), Valid: true},
			ContactStyle:     "ambient",
		},
		
		// Provider relationships
		{
			Name:             "Dr. Anderson",
			Email:            NewNullString("office@medical.practice"),
			Phone:            NewNullString("555-0110"),
			Company:          NewNullString("Medical Practice"),
			RelationshipType: "providers",
			State:            NewNullString("ok"),
			Notes:            NewNullString("Primary care physician, annual checkup due in March"),
			Label:            NewNullString("@dranderson"),
			ContactStyle:     "triggered",
		},
		{
			Name:             "Tom's Auto Shop",
			Email:            NewNullString("service@tomsauto.com"),
			Phone:            NewNullString("555-0111"),
			Company:          NewNullString("Tom's Auto Repair"),
			RelationshipType: "providers",
			State:            NewNullString("ok"),
			Notes:            NewNullString("Reliable mechanic, honest pricing. Next oil change due in 2 months."),
			Label:            NewNullString("@tomsauto"),
			ContactStyle:     "triggered",
		},
		
		// Recruiter relationships
		{
			Name:             "Amanda Foster",
			Email:            NewNullString("amanda@tech.recruiting"),
			Phone:            NewNullString("555-0112"),
			Company:          NewNullString("Tech Recruiting Firm"),
			RelationshipType: "recruiters",
			State:            NewNullString("ok"),
			Notes:            NewNullString("Technical recruiter specializing in senior engineering roles. Good market insights."),
			Label:            NewNullString("@amandaf"),
			ContactedAt:      sql.NullTime{Time: time.Now().AddDate(0, 0, -180), Valid: true},
			ContactStyle:     "triggered",
		},
		
		// Archived contact
		{
			Name:             "Old Colleague",
			Email:            NewNullString("old@former.company"),
			Company:          NewNullString("Former Company"),
			RelationshipType: "work",
			State:            NewNullString("ok"),
			Notes:            NewNullString("Former colleague who left the industry"),
			Label:            NewNullString("@oldcolleague"),
			ContactedAt:      sql.NullTime{Time: time.Now().AddDate(0, 0, -300), Valid: true},
			ContactStyle:     "periodic",
			Archived:         true,
			ArchivedAt:       sql.NullTime{Time: time.Now().AddDate(0, 0, -30), Valid: true},
		},
	}
	
	// Add all fixture contacts and track their IDs
	contactIDs := make(map[string]int)
	for _, contact := range fixtures {
		id, err := database.AddContact(contact)
		if err != nil {
			return fmt.Errorf("adding fixture contact %s: %w", contact.Name, err)
		}
		contactIDs[contact.Name] = int(id)
	}
	
	// Add some interaction logs using the contact IDs
	logs := []struct {
		contactName     string
		interactionType string
		daysAgo        int
		notes          string
	}{
		{"Sarah Chen", "email", 15, "Caught up over coffee, discussed her new startup"},
		{"Sarah Chen", "text", 20, "Quick check-in about project progress"},
		{"Jennifer Rodriguez", "meeting", 10, "API integration planning meeting"},
		{"David Kim", "email", 20, "Follow-up on partnership discussion"},
		{"Mike Johnson", "call", 25, "Planned weekend hiking trip"},
		{"Mom", "call", 7, "Weekly family check-in"},
		{"Lisa Park", "email", 60, "Shared article about market trends"},
	}
	
	for _, log := range logs {
		contactID, exists := contactIDs[log.contactName]
		if !exists {
			continue // Skip if contact not found
		}
		
		// Use AddInteractionNote method instead of AddLog
		if err := database.AddInteractionNote(contactID, log.interactionType, log.notes); err != nil {
			return fmt.Errorf("adding interaction note for %s: %w", log.contactName, err)
		}
	}
	
	return nil
}
