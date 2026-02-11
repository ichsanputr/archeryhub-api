package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// GetPublicQualificationResults returns qualification results for a specific category
func GetPublicQualificationResults(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		categoryID := c.Query("category_id")
		
		if eventID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "eventId is required"})
			return
		}
		
		if categoryID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category_id is required"})
			return
		}

		// Resolve event UUID (allow slug)
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Calculate total cumulative ends for the event summary
		var totalCumulativeEnds int
		err = db.Get(&totalCumulativeEnds, `
			SELECT COALESCE(SUM(total_ends), 0) 
			FROM qualification_sessions 
			WHERE event_uuid = ?
		`, eventUUID)
		if err != nil || totalCumulativeEnds == 0 {
			totalCumulativeEnds = 12
		}

		type SessionScore struct {
			SessionCode string `json:"session_code"`
			SessionName string `json:"session_name"`
			EndScores   string `json:"end_scores"`
			TotalEnds   int    `json:"total_ends"`
			TotalScore  int    `json:"total_score"`
			TotalTenX   int    `json:"total_10x"`
			TotalX      int    `json:"total_x"`
		}

		type Entry struct {
			Rank            int            `json:"rank"`
			ParticipantUUID string         `json:"participant_id" db:"participant_uuid"`
			ArcherUUID      string         `json:"archer_uuid" db:"archer_uuid"`
			ArcherName      string         `json:"archer_name" db:"archer_name"`
			AvatarURL       *string        `json:"avatar_url" db:"avatar_url"`
			ClubName        *string        `json:"club_name" db:"club_name"`
			TotalScore      int            `json:"total_score"`
			TotalTenX       int            `json:"total_10x"`
			TotalX          int            `json:"total_x"`
			EndsCompleted   int            `json:"ends_completed"`
			Sessions        []SessionScore `json:"sessions"`
		}

		type dbEntry struct {
			ParticipantUUID string  `db:"participant_uuid"`
			ArcherName      string  `db:"archer_name"`
			ArcherUUID      string  `db:"archer_uuid"`
			AvatarURL       *string `db:"avatar_url"`
			ClubName        *string `db:"club_name"`
			SessionName     *string `db:"session_name"`
			SessionCode     *string `db:"session_code"`
			SessionTotalEnds int    `db:"session_total_ends"`
			TotalScore      int     `db:"total_score"`
			TotalTenX       int     `db:"total_10x"`
			TotalX          int     `db:"total_x"`
			EndsCompleted   int     `db:"ends_completed"`
			EndScores       *string `db:"end_scores"`
		}

		var dbEntries []dbEntry
		err = db.Select(&dbEntries, `
			SELECT 
				ep.uuid as participant_uuid,
				a.uuid as archer_uuid,
				a.full_name as archer_name,
				a.avatar_url as avatar_url,
				cl.name as club_name,
				qs.name as session_name,
				qs.session_code as session_code,
				qs.total_ends as session_total_ends,
				COALESCE(score_summary.total_score, 0) as total_score,
				COALESCE(score_summary.total_10x, 0) as total_10x,
				COALESCE(score_summary.total_x, 0) as total_x,
				COALESCE(score_summary.ends_completed, 0) as ends_completed,
				score_summary.end_scores
			FROM event_participants ep
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			JOIN qualification_target_assignments qta ON qta.participant_uuid = ep.uuid
			JOIN qualification_sessions qs ON qs.uuid = qta.session_uuid
			LEFT JOIN (
				SELECT 
					participant_uuid, 
					session_uuid,
					SUM(total_score_end) as total_score,
					SUM(ten_count_end) as total_10x,
					SUM(x_count_end) as total_x,
					COUNT(uuid) as ends_completed,
					GROUP_CONCAT(COALESCE(total_score_end, 0) ORDER BY end_number ASC SEPARATOR ',') as end_scores
				FROM qualification_end_scores
				GROUP BY participant_uuid, session_uuid
			) score_summary ON score_summary.participant_uuid = ep.uuid AND score_summary.session_uuid = qs.uuid
			WHERE ep.category_id = ?
			ORDER BY archer_name, qs.created_at ASC`,
			categoryID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch results", "details": err.Error()})
			return
		}

		// Group by archer
		archerMap := make(map[string]*Entry)
		archerOrder := []string{}

		for _, de := range dbEntries {
			if _, ok := archerMap[de.ParticipantUUID]; !ok {
				archerMap[de.ParticipantUUID] = &Entry{
					ParticipantUUID: de.ParticipantUUID,
					ArcherUUID:      de.ArcherUUID,
					ArcherName:      de.ArcherName,
					AvatarURL:       de.AvatarURL,
					ClubName:        de.ClubName,
					Sessions:        []SessionScore{},
				}
				archerOrder = append(archerOrder, de.ParticipantUUID)
			}

			entry := archerMap[de.ParticipantUUID]
			entry.TotalScore += de.TotalScore
			entry.TotalTenX += de.TotalTenX
			entry.TotalX += de.TotalX
			entry.EndsCompleted += de.EndsCompleted

			if de.SessionCode != nil {
				endScores := ""
				if de.EndScores != nil {
					endScores = *de.EndScores
				}
				entry.Sessions = append(entry.Sessions, SessionScore{
					SessionCode: *de.SessionCode,
					SessionName: *de.SessionName,
					EndScores:   endScores,
					TotalEnds:   de.SessionTotalEnds,
					TotalScore:  de.TotalScore,
					TotalTenX:   de.TotalTenX,
					TotalX:      de.TotalX,
				})
			}
		}

		// Convert map to slice and sort
		leaderboard := make([]*Entry, 0, len(archerOrder))
		for _, uuid := range archerOrder {
			leaderboard = append(leaderboard, archerMap[uuid])
		}

		sort.Slice(leaderboard, func(i, j int) bool {
			if leaderboard[i].TotalScore != leaderboard[j].TotalScore {
				return leaderboard[i].TotalScore > leaderboard[j].TotalScore
			}
			if leaderboard[i].TotalTenX != leaderboard[j].TotalTenX {
				return leaderboard[i].TotalTenX > leaderboard[j].TotalTenX
			}
			return leaderboard[i].TotalX > leaderboard[j].TotalX
		})

		// Assign ranks
		for i := range leaderboard {
			leaderboard[i].Rank = i + 1
		}

		c.JSON(http.StatusOK, gin.H{
			"results": leaderboard,
			"total_ends": totalCumulativeEnds,
		})
	}
}

// GetPublicEliminationResults returns elimination bracket for a specific category
func GetPublicEliminationResults(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		categoryID := c.Query("category_id")

		if eventID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "eventId is required"})
			return
		}

		if categoryID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category_id is required"})
			return
		}

		// Resolve event UUID (allow slug)
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Get bracket for this category
		type Bracket struct {
			UUID         string  `db:"uuid" json:"uuid"`
			BracketID    string  `db:"bracket_id" json:"bracket_id"`
			CategoryUUID string  `db:"category_uuid" json:"category_uuid"`
			BracketType  string  `db:"bracket_type" json:"bracket_type"`
			Format       string  `db:"format" json:"format"`
			BracketSize  int     `db:"bracket_size" json:"bracket_size"`
			EndsPerMatch int     `db:"ends_per_match" json:"ends_per_match"`
			ArrowsPerEnd int     `db:"arrows_per_end" json:"arrows_per_end"`
			GeneratedAt  *string `db:"generated_at" json:"generated_at"`
		}

		var bracket Bracket
		err = db.Get(&bracket, `
			SELECT 
				eb.uuid,
				eb.bracket_id,
				eb.category_uuid,
				eb.bracket_type,
				eb.format,
				eb.bracket_size,
				eb.ends_per_match,
				eb.arrows_per_end,
				eb.generated_at
			FROM elimination_brackets eb
			WHERE eb.event_uuid = ? AND eb.category_uuid = ? AND eb.generated_at IS NOT NULL
			LIMIT 1
		`, eventUUID, categoryID)

		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusOK, gin.H{"bracket": nil})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bracket", "details": err.Error()})
			return
		}

		// Get all matches for this bracket
		type Match struct {
			UUID            string  `db:"uuid" json:"id"`
			RoundNo         int     `db:"round_no" json:"round_no"`
			MatchNo         int     `db:"match_no" json:"match_no"`
			EntryAUUID      *string `db:"entry_a_uuid" json:"entry_a_id"`
			EntryAName      *string `db:"entry_a_name" json:"entry_a_name"`
			EntryASeed      *int    `db:"entry_a_seed" json:"entry_a_seed"`
			EntryBUUID      *string `db:"entry_b_uuid" json:"entry_b_id"`
			EntryBName      *string `db:"entry_b_name" json:"entry_b_name"`
			EntryBSeed      *int    `db:"entry_b_seed" json:"entry_b_seed"`
			WinnerEntryUUID *string `db:"winner_entry_uuid" json:"winner_entry_id"`
			Status          string  `db:"status" json:"status"`
			IsBye           bool    `db:"is_bye" json:"is_bye"`
			TotalScoreA     int     `json:"total_score_a"`
			TotalScoreB     int     `json:"total_score_b"`
			SetPointsA      int     `json:"set_points_a"`
			SetPointsB      int     `json:"set_points_b"`
			ShootOffA       *string `json:"shoot_off_a"`
			ShootOffB       *string `json:"shoot_off_b"`
		}

		var matches []Match
		err = db.Select(&matches, `
			SELECT 
				em.uuid,
				em.round_no,
				em.match_no,
				em.entry_a_uuid,
				COALESCE(a1.full_name, t1.team_name) as entry_a_name,
				ee1.seed as entry_a_seed,
				em.entry_b_uuid,
				COALESCE(a2.full_name, t2.team_name) as entry_b_name,
				ee2.seed as entry_b_seed,
				em.winner_entry_uuid,
				em.status,
				em.is_bye
			FROM elimination_matches em
			LEFT JOIN elimination_entries ee1 ON em.entry_a_uuid = ee1.uuid
			LEFT JOIN elimination_entries ee2 ON em.entry_b_uuid = ee2.uuid
			LEFT JOIN archers a1 ON ee1.participant_uuid = a1.uuid AND ee1.participant_type = 'archer'
			LEFT JOIN archers a2 ON ee2.participant_uuid = a2.uuid AND ee2.participant_type = 'archer'
			LEFT JOIN teams t1 ON ee1.participant_uuid = t1.uuid AND ee1.participant_type = 'team'
			LEFT JOIN teams t2 ON ee2.participant_uuid = t2.uuid AND ee2.participant_type = 'team'
			WHERE em.bracket_uuid = ?
			ORDER BY em.round_no ASC, em.match_no ASC
		`, bracket.UUID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch matches", "details": err.Error()})
			return
		}

		// Calculate scores and points
		type matchEnd struct {
			MatchUUID string `db:"match_uuid"`
			EndNo     int    `db:"end_no"`
			Side      string `db:"side"`
			EndTotal  int    `db:"end_total"`
		}
		var allEnds []matchEnd
		err = db.Select(&allEnds, `
			SELECT match_uuid, end_no, side, end_total
			FROM elimination_match_ends
			WHERE match_uuid IN (SELECT uuid FROM elimination_matches WHERE bracket_uuid = ?)
		`, bracket.UUID)

		// Fetch shoot-off arrows for all matches
		type shootOffArrow struct {
			MatchUUID string `db:"match_uuid"`
			Side      string `db:"side"`
			Score     int    `db:"score"`
			IsX       bool   `db:"is_x"`
		}
		var allSoArrows []shootOffArrow
		db.Select(&allSoArrows, `
			SELECT eme.match_uuid, eme.side, emas.score, emas.is_x
			FROM elimination_match_arrow_scores emas
			JOIN elimination_match_ends eme ON emas.match_end_uuid = eme.uuid
			WHERE eme.match_uuid IN (SELECT uuid FROM elimination_matches WHERE bracket_uuid = ?)
			  AND eme.end_no = 99
		`, bracket.UUID)

		soArrowsMap := make(map[string]map[string]string)
		for _, a := range allSoArrows {
			if soArrowsMap[a.MatchUUID] == nil {
				soArrowsMap[a.MatchUUID] = make(map[string]string)
			}
			val := fmt.Sprintf("%d", a.Score)
			if a.IsX {
				val = "X"
			} else if a.Score == 0 {
				val = "M"
			}
			soArrowsMap[a.MatchUUID][a.Side] = val
		}

		if err == nil {
			endsByMatch := make(map[string]map[int]map[string]int)
			for _, e := range allEnds {
				if endsByMatch[e.MatchUUID] == nil {
					endsByMatch[e.MatchUUID] = make(map[int]map[string]int)
				}
				if endsByMatch[e.MatchUUID][e.EndNo] == nil {
					endsByMatch[e.MatchUUID][e.EndNo] = make(map[string]int)
				}
				endsByMatch[e.MatchUUID][e.EndNo][e.Side] = e.EndTotal
			}

			for i := range matches {
				mEnds := endsByMatch[matches[i].UUID]
				tSA, tSB := 0, 0
				tPA, tPB := 0, 0

				var endNos []int
				for en := range mEnds {
					endNos = append(endNos, en)
				}
				sort.Ints(endNos)

				for _, en := range endNos {
					if en == 99 { continue }
					scA := mEnds[en]["A"]
					scB := mEnds[en]["B"]
					tSA += scA
					tSB += scB

					if bracket.Format == "recurve_set" {
						if scA > scB {
							tPA += 2
						} else if scB > scA {
							tPB += 2
						} else if scA == scB && scA > 0 {
							tPA += 1
							tPB += 1
						}
					}
				}

				// Add shoot-off info
				if soMap, ok := soArrowsMap[matches[i].UUID]; ok {
					var vA, vB string
					if val, ok := soMap["A"]; ok {
						matches[i].ShootOffA = &val
						vA = val
					}
					if val, ok := soMap["B"]; ok {
						matches[i].ShootOffB = &val
						vB = val
					}

					// If both shot, determine winner and add 1 point
					if vA != "" && vB != "" {
						getV := func(v string) int {
							if v == "X" {
								return 11
							}
							if v == "M" {
								return 0
							}
							var sc int
							fmt.Sscanf(v, "%d", &sc)
							return sc
						}
						scA := getV(vA)
						scB := getV(vB)
						if scA > scB {
							if bracket.Format == "recurve_set" {
								tPA++
							} else {
								tSA++
							}
						} else if scB > scA {
							if bracket.Format == "recurve_set" {
								tPB++
							} else {
								tSB++
							}
						}
					}
				}

				matches[i].TotalScoreA = tSA
				matches[i].TotalScoreB = tSB
				matches[i].SetPointsA = tPA
				matches[i].SetPointsB = tPB
			}
		}

		// Group matches by round
		matchesByRound := make(map[int][]Match)
		for _, match := range matches {
			matchesByRound[match.RoundNo] = append(matchesByRound[match.RoundNo], match)
		}

		bracket_result := gin.H{
			"uuid":           bracket.UUID,
			"bracket_id":     bracket.BracketID,
			"bracket_type":   bracket.BracketType,
			"format":         bracket.Format,
			"bracket_size":   bracket.BracketSize,
			"ends_per_match": bracket.EndsPerMatch,
			"arrows_per_end": bracket.ArrowsPerEnd,
			"generated_at":   bracket.GeneratedAt,
			"matches":        matchesByRound,
		}

		c.JSON(http.StatusOK, gin.H{"bracket": bracket_result})
	}
}
