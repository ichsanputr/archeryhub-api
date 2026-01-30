package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// PrintRequest represents a request for generating printable output
type PrintRequest struct {
	Type      string `json:"type" binding:"required"` // scorecard, rankings, bracket, startlist, backnumbers
	EventID   string `json:"event_id"`
	CategoryID string `json:"category_id"`
	SessionID string `json:"session_id"`
	Format    string `json:"format"` // pdf, html, csv
}

// PrintResponse contains the generated content or URL
type PrintResponse struct {
	Type    string      `json:"type"`
	Format  string      `json:"format"`
	Content interface{} `json:"content"`
	URL     string      `json:"url,omitempty"`
}

// GeneratePrintOutput generates various printable outputs
func GeneratePrintOutput(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req PrintRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Format == "" {
			req.Format = "html"
		}

		var response PrintResponse
		response.Type = req.Type
		response.Format = req.Format

		switch req.Type {
		case "startlist":
			response.Content = generateStartList(db, req)
		case "backnumbers":
			response.Content = generateBackNumbers(db, req)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown print type"})
			return
		}

		c.JSON(http.StatusOK, response)
	}
}


func generateStartList(db *sqlx.DB, req PrintRequest) interface{} {
	type StartListEntry struct {
		BackNumber  string  `db:"back_number" json:"back_number"`
		Target      string  `db:"target" json:"target"`
		AthleteName string  `db:"athlete_name" json:"athlete_name"`
		City        *string `db:"city" json:"city"`
		Club        *string `db:"club" json:"club"`
		Division    string  `db:"division" json:"division"`
		Category    string  `db:"category" json:"category"`
	}

	var entries []StartListEntry
	query := `
		SELECT 
			COALESCE(tp.back_number, '') as back_number,
			COALESCE(tp.target_number, '') as target,
			a.full_name as athlete_name,
			a.club_id as club,
			d.name as division, c.name as category
		FROM event_participants tp
		JOIN archers a ON tp.athlete_id = a.uuid
		JOIN event_categories ec ON tp.event_category_id = ec.uuid
		JOIN ref_disciplines d ON ec.discipline_id = d.uuid
		JOIN ref_gender_divisions c ON ec.gender_division_id = c.uuid
		WHERE tp.event_id = ?
	`
	// Note: ref_disciplines and ref_gender_divisions are the actual table names in my context usually,
	// but I should check divisions and categories names in this DB.
	// Step 1746 showed: ref_disciplines, ref_gender_divisions, etc.
	
	args := []interface{}{req.EventID}

	if req.SessionID != "" {
		query += " AND tp.session = ?"
		args = append(args, req.SessionID)
	}

	query += " ORDER BY tp.session, tp.target_number, tp.back_number"

	err := db.Select(&entries, query, args...)
	if err != nil {
		return []StartListEntry{}
	}
	return entries
}

func generateBackNumbers(db *sqlx.DB, req PrintRequest) interface{} {
	type BackNumberCard struct {
		BackNumber  string  `db:"back_number" json:"back_number"`
		AthleteName string  `db:"athlete_name" json:"athlete_name"`
		Country     *string `db:"country" json:"country"`
		Division    string  `db:"division" json:"division"`
		Target      string  `db:"target" json:"target"`
		Session     int     `db:"session" json:"session"`
	}

	var cards []BackNumberCard
	err := db.Select(&cards, `
		SELECT 
			COALESCE(tp.back_number, '') as back_number,
			a.full_name as athlete_name,
			a.city,
			d.name as division,
			COALESCE(tp.target_number, '') as target,
			COALESCE(tp.session, 1) as session
		FROM event_participants tp
		JOIN archers a ON tp.athlete_id = a.uuid
		JOIN event_categories ec ON tp.event_category_id = ec.uuid
		JOIN ref_disciplines d ON ec.discipline_id = d.uuid
		WHERE tp.event_id = ?
		ORDER BY tp.back_number
	`, req.EventID)
	
	if err != nil {
		return []BackNumberCard{}
	}

	return cards
}


// ExportCSV exports data in CSV format
func ExportCSV(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		exportType := c.Param("type")
		eventID := c.Query("event_id")

		var data interface{}
		var filename string

		switch exportType {
		case "startlist":
			data = generateStartList(db, PrintRequest{EventID: eventID})
			filename = "startlist.csv"
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown export type"})
			return
		}

		// Convert to JSON for now (actual CSV generation would go here)
		jsonData, _ := json.Marshal(data)

		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Content-Type", "text/csv")
		c.String(http.StatusOK, string(jsonData))
	}
}
