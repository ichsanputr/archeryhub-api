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
		case "scorecard":
			response.Content = generateScorecard(db, req)
		case "rankings":
			response.Content = generateRankings(db, req)
		case "bracket":
			response.Content = generateBracket(db, req)
		case "startlist":
			response.Content = generateStartList(db, req)
		case "backnumbers":
			response.Content = generateBackNumbers(db, req)
		case "medals":
			response.Content = generateMedalReport(db, req)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown print type"})
			return
		}

		c.JSON(http.StatusOK, response)
	}
}

func generateScorecard(db *sqlx.DB, req PrintRequest) interface{} {
	type Scorecard struct {
		ParticipantID string `db:"participant_id" json:"participant_id"`
		AthleteName   string `db:"athlete_name" json:"athlete_name"`
		BackNumber    string `db:"back_number" json:"back_number"`
		Target        string `db:"target" json:"target"`
		Session       int    `db:"session" json:"session"`
		Ends          []struct {
			EndNumber int   `json:"end_number"`
			Arrows    []int `json:"arrows"`
			Total     int   `json:"total"`
		} `json:"ends"`
	}

	var participants []struct {
		ParticipantID string  `db:"participant_id"`
		AthleteName   string  `db:"athlete_name"`
		BackNumber    *string `db:"back_number"`
		Target        *string `db:"target"`
		Session       int     `db:"session"`
	}

	db.Select(&participants, `
		SELECT tp.id as participant_id, 
			CONCAT(a.first_name, ' ', a.last_name) as athlete_name,
			tp.back_number, tp.target, tp.session
		FROM tournament_participants tp
		JOIN athletes a ON tp.athlete_id = a.id
		WHERE tp.tournament_id = ?
		ORDER BY tp.session, tp.target
	`, req.TournamentID)

	scorecards := make([]Scorecard, 0)
	for _, p := range participants {
		sc := Scorecard{
			ParticipantID: p.ParticipantID,
			AthleteName:   p.AthleteName,
			Session:       p.Session,
		}
		if p.BackNumber != nil {
			sc.BackNumber = *p.BackNumber
		}
		if p.Target != nil {
			sc.Target = *p.Target
		}
		scorecards = append(scorecards, sc)
	}

	return scorecards
}

func generateRankings(db *sqlx.DB, req PrintRequest) interface{} {
	type RankingRow struct {
		Rank        int     `db:"rank" json:"rank"`
		AthleteName string  `db:"athlete_name" json:"athlete_name"`
		Country     *string `db:"country" json:"country"`
		TotalScore  int     `db:"total_score" json:"total_score"`
		XCount      int     `db:"x_count" json:"x_count"`
		TenCount    int     `db:"ten_count" json:"ten_count"`
	}

	var rankings []RankingRow
	db.Select(&rankings, `
		SELECT 
			ROW_NUMBER() OVER (ORDER BY SUM(qs.end_total) DESC, SUM(qs.x_count) DESC, SUM(qs.ten_count) DESC) as rank,
			CONCAT(a.first_name, ' ', a.last_name) as athlete_name,
			a.country,
			COALESCE(SUM(qs.end_total), 0) as total_score,
			COALESCE(SUM(qs.x_count), 0) as x_count,
			COALESCE(SUM(qs.ten_count), 0) as ten_count
		FROM tournament_participants tp
		JOIN athletes a ON tp.athlete_id = a.id
		LEFT JOIN qualification_scores qs ON qs.participant_id = tp.id
		WHERE tp.tournament_id = ?
		GROUP BY tp.id, a.first_name, a.last_name, a.country
		ORDER BY total_score DESC, x_count DESC, ten_count DESC
	`, req.TournamentID)

	return rankings
}

func generateBracket(db *sqlx.DB, req PrintRequest) interface{} {
	type MatchNode struct {
		MatchID     string  `db:"id" json:"match_id"`
		Round       string  `db:"round" json:"round"`
		MatchNumber int     `db:"match_number" json:"match_number"`
		Athlete1    *string `json:"athlete1"`
		Athlete2    *string `json:"athlete2"`
		Score1      int     `db:"score1" json:"score1"`
		Score2      int     `db:"score2" json:"score2"`
		Winner      *string `json:"winner"`
		Status      string  `db:"status" json:"status"`
	}

	var matches []struct {
		ID              string  `db:"id"`
		Round           string  `db:"round"`
		MatchNumber     int     `db:"match_number"`
		Participant1ID  *string `db:"participant1_id"`
		Participant2ID  *string `db:"participant2_id"`
		Score1          int     `db:"score1"`
		Score2          int     `db:"score2"`
		WinnerID        *string `db:"winner_id"`
		Status          string  `db:"status"`
	}

	db.Select(&matches, `
		SELECT id, round, match_number, participant1_id, participant2_id, 
			score1, score2, winner_id, status
		FROM elimination_matches
		WHERE event_id = ?
		ORDER BY 
			CASE round
				WHEN 'R64' THEN 1
				WHEN 'R32' THEN 2
				WHEN 'R16' THEN 3
				WHEN 'R8' THEN 4
				WHEN 'QF' THEN 5
				WHEN 'SF' THEN 6
				WHEN 'BM' THEN 7
				WHEN 'GM' THEN 8
			END, match_number
	`, req.EventID)

	// Get athlete names
	nameMap := make(map[string]string)
	var names []struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	db.Select(&names, `
		SELECT tp.id, CONCAT(a.first_name, ' ', a.last_name) as name
		FROM tournament_participants tp
		JOIN athletes a ON tp.athlete_id = a.id
		WHERE tp.event_id = ?
	`, req.EventID)
	
	for _, n := range names {
		nameMap[n.ID] = n.Name
	}

	nodes := make([]MatchNode, 0)
	for _, m := range matches {
		node := MatchNode{
			MatchID:     m.ID,
			Round:       m.Round,
			MatchNumber: m.MatchNumber,
			Score1:      m.Score1,
			Score2:      m.Score2,
			Status:      m.Status,
		}
		if m.Participant1ID != nil {
			if name, ok := nameMap[*m.Participant1ID]; ok {
				node.Athlete1 = &name
			}
		}
		if m.Participant2ID != nil {
			if name, ok := nameMap[*m.Participant2ID]; ok {
				node.Athlete2 = &name
			}
		}
		if m.WinnerID != nil {
			if name, ok := nameMap[*m.WinnerID]; ok {
				node.Winner = &name
			}
		}
		nodes = append(nodes, node)
	}

	return nodes
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
			CONCAT(a.first_name, ' ', a.last_name) as athlete_name,
			a.country, a.club,
			d.name as division, c.name as category
		FROM tournament_participants tp
		JOIN athletes a ON tp.athlete_id = a.id
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
			CONCAT(a.first_name, ' ', a.last_name) as athlete_name,
			a.country,
			d.name as division,
			COALESCE(tp.target, '') as target,
			COALESCE(tp.session, 1) as session
		FROM tournament_participants tp
		JOIN athletes a ON tp.athlete_id = a.id
		JOIN tournament_events te ON tp.event_id = te.id
		JOIN divisions d ON te.division_id = d.id
		WHERE tp.tournament_id = ?
		ORDER BY tp.back_number
	`, req.TournamentID)

	return cards
}

func generateMedalReport(db *sqlx.DB, req PrintRequest) interface{} {
	type MedalEntry struct {
		Event       string  `json:"event"`
		Gold        *string `json:"gold"`
		GoldCountry *string `json:"gold_country"`
		Silver      *string `json:"silver"`
		SilverCountry *string `json:"silver_country"`
		Bronze      *string `json:"bronze"`
		BronzeCountry *string `json:"bronze_country"`
	}

	// Get events with medal winners
	var events []struct {
		EventID  string `db:"event_id"`
		EventName string `db:"event_name"`
	}
	db.Select(&events, `
		SELECT te.id as event_id, CONCAT(d.name, ' - ', c.name) as event_name
		FROM tournament_events te
		JOIN divisions d ON te.division_id = d.id
		JOIN categories c ON te.category_id = c.id
		WHERE te.tournament_id = ?
	`, req.TournamentID)

	medals := make([]MedalEntry, 0)
	for _, e := range events {
		entry := MedalEntry{Event: e.EventName}

		// Get medal winners for this event
		var winners []struct {
			AwardType string `db:"award_type"`
			Name      string `db:"name"`
			Country   *string `db:"country"`
		}
		db.Select(&winners, `
			SELECT a.award_type, CONCAT(ath.first_name, ' ', ath.last_name) as name, ath.country
			FROM awards a
			JOIN tournament_participants tp ON a.recipient_id = tp.id
			JOIN athletes ath ON tp.athlete_id = ath.id
			WHERE a.event_id = ? AND a.award_type IN ('gold', 'silver', 'bronze')
		`, e.EventID)

		for _, w := range winners {
			switch w.AwardType {
			case "gold":
				entry.Gold = &w.Name
				entry.GoldCountry = w.Country
			case "silver":
				entry.Silver = &w.Name
				entry.SilverCountry = w.Country
			case "bronze":
				entry.Bronze = &w.Name
				entry.BronzeCountry = w.Country
			}
		}

		medals = append(medals, entry)
	}

	return medals
}

// ExportCSV exports data in CSV format
func ExportCSV(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		exportType := c.Param("type")
		tournamentID := c.Query("tournament_id")

		var data interface{}
		var filename string

		switch exportType {
		case "rankings":
			data = generateRankings(db, PrintRequest{TournamentID: tournamentID})
			filename = "rankings.csv"
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
