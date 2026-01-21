package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// PrintRequest represents a request for generating printable output
type PrintRequest struct {
	Type         string `json:"type" binding:"required"` // scorecard, rankings, bracket, startlist, backnumbers
	TournamentID string `json:"tournament_id"`
	EventID      string `json:"event_id"`
	SessionID    string `json:"session_id"`
	Format       string `json:"format"` // pdf, html, csv
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
		Country     *string `db:"country" json:"country"`
		Club        *string `db:"club" json:"club"`
		Division    string  `db:"division" json:"division"`
		Category    string  `db:"category" json:"category"`
	}

	var entries []StartListEntry
	query := `
		SELECT 
			COALESCE(tp.back_number, '') as back_number,
			COALESCE(tp.target, '') as target,
			a.full_name as athlete_name,
			a.club_id as club,
			d.name as division, c.name as category
		FROM tournament_participants tp
		JOIN archers a ON tp.athlete_id = a.id
		JOIN tournament_events te ON tp.event_id = te.id
		JOIN divisions d ON te.division_id = d.id
		JOIN categories c ON te.category_id = c.id
		WHERE tp.tournament_id = ?
	`
	args := []interface{}{req.TournamentID}

	if req.SessionID != "" {
		query += " AND tp.session = ?"
		args = append(args, req.SessionID)
	}

	query += " ORDER BY tp.session, tp.target, tp.back_number"

	db.Select(&entries, query, args...)
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
	db.Select(&cards, `
		SELECT 
			COALESCE(tp.back_number, '') as back_number,
			a.full_name as athlete_name,
			a.country,
			d.name as division,
			COALESCE(tp.target, '') as target,
			COALESCE(tp.session, 1) as session
		FROM tournament_participants tp
		JOIN archers a ON tp.athlete_id = a.id
		JOIN event_categories te ON tp.event_id = te.id
		JOIN divisions d ON te.division_id = d.id
		WHERE tp.tournament_id = ?
		ORDER BY tp.back_number
	`, req.TournamentID)

	return cards
}


// ExportCSV exports data in CSV format
func ExportCSV(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		exportType := c.Param("type")
		tournamentID := c.Query("tournament_id")

		var data interface{}
		var filename string

		switch exportType {
		case "startlist":
			data = generateStartList(db, PrintRequest{TournamentID: tournamentID})
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
