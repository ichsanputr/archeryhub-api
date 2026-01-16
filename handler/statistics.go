package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// MatchStatistics represents detailed statistics for a match
type MatchStatistics struct {
	MatchID       string          `json:"match_id"`
	Round         string          `json:"round"`
	Participant1  *AthleteStats   `json:"participant1"`
	Participant2  *AthleteStats   `json:"participant2"`
	SetDetails    []SetDetail     `json:"set_details"`
	HighestArrow  int             `json:"highest_arrow"`
	TotalArrows   int             `json:"total_arrows"`
	AverageScore  float64         `json:"average_score"`
}

type AthleteStats struct {
	ParticipantID string  `json:"participant_id"`
	Name          string  `json:"name"`
	Country       *string `json:"country"`
	TotalScore    int     `json:"total_score"`
	SetScore      int     `json:"set_score"`
	Arrows        []int   `json:"arrows"`
	Tens          int     `json:"tens"`
	XCount        int     `json:"x_count"`
	Average       float64 `json:"average"`
	HighestEnd    int     `json:"highest_end"`
}

type SetDetail struct {
	SetNumber int   `json:"set_number"`
	P1Arrows  []int `json:"p1_arrows"`
	P2Arrows  []int `json:"p2_arrows"`
	P1Total   int   `json:"p1_total"`
	P2Total   int   `json:"p2_total"`
	P1SetPts  int   `json:"p1_set_pts"`
	P2SetPts  int   `json:"p2_set_pts"`
	Winner    int   `json:"winner"` // 1, 2, or 0 for tie
}

// GetMatchStatistics returns detailed statistics for a match
func GetMatchStatistics(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchID := c.Param("matchId")

		// Get basic match info
		var match struct {
			ID             string  `db:"id"`
			Round          string  `db:"round"`
			Participant1ID *string `db:"participant1_id"`
			Participant2ID *string `db:"participant2_id"`
			Score1         int     `db:"score1"`
			Score2         int     `db:"score2"`
			SetScore1      int     `db:"set_score1"`
			SetScore2      int     `db:"set_score2"`
		}

		err := db.Get(&match, "SELECT id, round, participant1_id, participant2_id, score1, score2, set_score1, set_score2 FROM elimination_matches WHERE id = ?", matchID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
			return
		}

		stats := MatchStatistics{
			MatchID: matchID,
			Round:   match.Round,
		}

		// Get participant 1 stats
		if match.Participant1ID != nil {
			p1Stats, _ := getAthleteMatchStats(db, *match.Participant1ID, match.Score1, match.SetScore1)
			stats.Participant1 = p1Stats
		}

		// Get participant 2 stats
		if match.Participant2ID != nil {
			p2Stats, _ := getAthleteMatchStats(db, *match.Participant2ID, match.Score2, match.SetScore2)
			stats.Participant2 = p2Stats
		}

		// Calculate totals
		if stats.Participant1 != nil && stats.Participant2 != nil {
			totalArrows := len(stats.Participant1.Arrows) + len(stats.Participant2.Arrows)
			stats.TotalArrows = totalArrows

			if totalArrows > 0 {
				totalScore := stats.Participant1.TotalScore + stats.Participant2.TotalScore
				stats.AverageScore = float64(totalScore) / float64(totalArrows)
			}

			// Find highest arrow
			for _, a := range stats.Participant1.Arrows {
				if a > stats.HighestArrow {
					stats.HighestArrow = a
				}
			}
			for _, a := range stats.Participant2.Arrows {
				if a > stats.HighestArrow {
					stats.HighestArrow = a
				}
			}
		}

		c.JSON(http.StatusOK, stats)
	}
}

func getAthleteMatchStats(db *sqlx.DB, participantID string, totalScore, setScore int) (*AthleteStats, error) {
	var info struct {
		FirstName string  `db:"first_name"`
		LastName  string  `db:"last_name"`
		Country   *string `db:"country"`
	}

	err := db.Get(&info, `
		SELECT a.first_name, a.last_name, a.country
		FROM tournament_participants tp
		JOIN athletes a ON tp.athlete_id = a.id
		WHERE tp.id = ?
	`, participantID)

	if err != nil {
		return nil, err
	}

	stats := &AthleteStats{
		ParticipantID: participantID,
		Name:          info.FirstName + " " + info.LastName,
		Country:       info.Country,
		TotalScore:    totalScore,
		SetScore:      setScore,
		Arrows:        []int{},
	}

	// Get arrows from qualification_scores for this participant
	type ArrowRow struct {
		Arrow1 *int `db:"arrow_1"`
		Arrow2 *int `db:"arrow_2"`
		Arrow3 *int `db:"arrow_3"`
		Arrow4 *int `db:"arrow_4"`
		Arrow5 *int `db:"arrow_5"`
		Arrow6 *int `db:"arrow_6"`
	}

	var arrows []ArrowRow
	db.Select(&arrows, `
		SELECT arrow_1, arrow_2, arrow_3, arrow_4, arrow_5, arrow_6
		FROM qualification_scores
		WHERE participant_id = ?
		ORDER BY session, distance_order, end_number
	`, participantID)

	endScores := []int{}
	for _, row := range arrows {
		arrowsInEnd := []*int{row.Arrow1, row.Arrow2, row.Arrow3, row.Arrow4, row.Arrow5, row.Arrow6}
		endTotal := 0
		for _, a := range arrowsInEnd {
			if a != nil {
				val := *a
				if val == 11 {
					stats.XCount++
					stats.Tens++
					val = 10
				} else if val == 10 {
					stats.Tens++
				}
				stats.Arrows = append(stats.Arrows, val)
				endTotal += val
			}
		}
		endScores = append(endScores, endTotal)
	}

	// Calculate highest end
	for _, e := range endScores {
		if e > stats.HighestEnd {
			stats.HighestEnd = e
		}
	}

	// Calculate average
	if len(stats.Arrows) > 0 {
		sum := 0
		for _, a := range stats.Arrows {
			sum += a
		}
		stats.Average = float64(sum) / float64(len(stats.Arrows))
	}

	return stats, nil
}

// GetEventStatistics returns aggregate statistics for an event
func GetEventStatistics(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("eventId")

		type EventStats struct {
			EventID           string  `json:"event_id"`
			TotalParticipants int     `json:"total_participants"`
			TotalMatches      int     `json:"total_matches"`
			CompletedMatches  int     `json:"completed_matches"`
			HighestQualScore  int     `json:"highest_qual_score"`
			AverageQualScore  float64 `json:"average_qual_score"`
			TotalArrows       int     `json:"total_arrows"`
			TotalXCount       int     `json:"total_x_count"`
			TotalTens         int     `json:"total_tens"`
		}

		var stats EventStats
		stats.EventID = eventID

		// Get participant count
		db.Get(&stats.TotalParticipants, `
			SELECT COUNT(*) FROM tournament_participants WHERE event_id = ?
		`, eventID)

		// Get match counts
		db.Get(&stats.TotalMatches, `
			SELECT COUNT(*) FROM elimination_matches WHERE event_id = ?
		`, eventID)

		db.Get(&stats.CompletedMatches, `
			SELECT COUNT(*) FROM elimination_matches WHERE event_id = ? AND status = 'completed'
		`, eventID)

		// Get qualification score stats
		type QualStats struct {
			HighScore  int     `db:"high_score"`
			AvgScore   float64 `db:"avg_score"`
			TotalX     int     `db:"total_x"`
			TotalTens  int     `db:"total_tens"`
			ArrowCount int     `db:"arrow_count"`
		}

		var qualStats QualStats
		db.Get(&qualStats, `
			SELECT 
				MAX(qs.running_total) as high_score,
				AVG(qs.end_total) as avg_score,
				SUM(qs.x_count) as total_x,
				SUM(qs.ten_count) as total_tens,
				COUNT(*) * 6 as arrow_count
			FROM qualification_scores qs
			JOIN tournament_participants tp ON qs.participant_id = tp.id
			WHERE tp.event_id = ?
		`, eventID)

		stats.HighestQualScore = qualStats.HighScore
		stats.AverageQualScore = qualStats.AvgScore
		stats.TotalXCount = qualStats.TotalX
		stats.TotalTens = qualStats.TotalTens
		stats.TotalArrows = qualStats.ArrowCount

		c.JSON(http.StatusOK, stats)
	}
}
