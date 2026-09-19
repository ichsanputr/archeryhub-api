package handler

import (
	"encoding/json"
	"net/http"

	"Archeris-api/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// GetExternalTournaments returns list of external scraped tournaments
func GetExternalTournaments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tournaments []models.TournamentExternalListItem
		query := `
			SELECT 
				id, uuid, slug, source_platform, external_id,
				name, short_name, venue, location, city, country,
				start_date, end_date, banner_url, logo_url, status,
				categories_count, participants_count
			FROM tournament_externals
			ORDER BY start_date DESC, created_at DESC
		`
		err := db.Select(&tournaments, query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data turnamen eksternal", "details": err.Error()})
			return
		}

		if tournaments == nil {
			tournaments = []models.TournamentExternalListItem{}
		}

		c.JSON(http.StatusOK, gin.H{
			"tournaments": tournaments,
			"total":       len(tournaments),
		})
	}
}

// GetExternalTournamentDetail returns detail + relational structured data of external tournament
func GetExternalTournamentDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slugOrID := c.Param("slug")
		trimmedID := slugOrID
		if len(slugOrID) > 7 && slugOrID[:7] == "ianseo-" {
			trimmedID = slugOrID[7:]
		}

		var t models.TournamentExternal
		query := `
			SELECT 
				id, uuid, slug, source_platform, external_id, source_url,
				name, short_name, venue, location, city, country,
				start_date, end_date, banner_url, logo_url, status,
				categories_count, participants_count, data_json,
				created_at, updated_at
			FROM tournament_externals
			WHERE slug = ? OR external_id = ? OR external_id = ? OR uuid = ?
			LIMIT 1
		`
		err := db.Get(&t, query, slugOrID, slugOrID, trimmedID, slugOrID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen eksternal tidak ditemukan"})
			return
		}

		// 1. Fetch Categories
		var categories []models.TournamentExternalCategory
		_ = db.Select(&categories, `
			SELECT id, tournament_id, category_name, category_code, is_team, distance, target_face, created_at 
			FROM tournament_external_categories 
			WHERE tournament_id = ? 
			ORDER BY category_name ASC
		`, t.ID)

		var categoryNames []string
		for _, cat := range categories {
			categoryNames = append(categoryNames, cat.CategoryName)
		}

		// 2. Fetch Qualifications
		type qualRow struct {
			ID           int64   `db:"id"`
			TournamentID int64   `db:"tournament_id"`
			CategoryID   int64   `db:"category_id"`
			CategoryName string  `db:"category_name"`
			Rank         int     `db:"rank"`
			TargetLane   *string `db:"target_lane"`
			BIB          *string `db:"bib"`
			AthleteName  string  `db:"athlete_name"`
			ClubCode     *string `db:"club_code"`
			ClubName     *string `db:"club_name"`
			D1Score      int     `db:"d1_score"`
			D2Score      int     `db:"d2_score"`
			D3Score      int     `db:"d3_score"`
			D4Score      int     `db:"d4_score"`
			TotalScore   int     `db:"total_score"`
			TensCount    int     `db:"tens_count"`
			XCount       int     `db:"x_count"`
			IsTeam       bool    `db:"is_team"`
			TeamMembers  *string `db:"team_members"`
		}
		var qualRows []qualRow
		_ = db.Select(&qualRows, `
			SELECT teq.id, teq.tournament_id, teq.category_id, tec.category_name,
			       teq.rank, teq.target_lane, teq.bib, teq.athlete_name, teq.club_code, teq.club_name,
			       teq.d1_score, teq.d2_score, teq.d3_score, teq.d4_score, teq.total_score,
			       teq.tens_count, teq.x_count, teq.is_team, teq.team_members
			FROM tournament_external_qualifications teq
			JOIN tournament_external_categories tec ON teq.category_id = tec.id
			WHERE teq.tournament_id = ?
			ORDER BY tec.category_name ASC, teq.rank ASC, teq.total_score DESC
		`, t.ID)

		qualifications := make(map[string][]map[string]interface{})
		teamQualifications := make(map[string][]map[string]interface{})

		for _, q := range qualRows {
			if q.IsTeam {
				var members []map[string]interface{}
				if q.TeamMembers != nil && *q.TeamMembers != "" {
					_ = json.Unmarshal([]byte(*q.TeamMembers), &members)
				}
				item := map[string]interface{}{
					"rank":        q.Rank,
					"code":        q.ClubCode,
					"club_code":   q.ClubCode,
					"club":        q.ClubName,
					"club_name":   q.ClubName,
					"name":        q.AthleteName,
					"total":       q.TotalScore,
					"total_score": q.TotalScore,
					"tens":        q.TensCount,
					"tens_count":  q.TensCount,
					"xs":          q.XCount,
					"x_count":     q.XCount,
					"members":     members,
				}
				teamQualifications[q.CategoryName] = append(teamQualifications[q.CategoryName], item)
			} else {
				item := map[string]interface{}{
					"rank":        q.Rank,
					"target":      q.TargetLane,
					"target_lane": q.TargetLane,
					"bib":         q.BIB,
					"name":        q.AthleteName,
					"code":        q.ClubCode,
					"club_code":   q.ClubCode,
					"club":        q.ClubName,
					"club_name":   q.ClubName,
					"d1":          q.D1Score,
					"distance_1":  q.D1Score,
					"d2":          q.D2Score,
					"distance_2":  q.D2Score,
					"d3":          q.D3Score,
					"distance_3":  q.D3Score,
					"d4":          q.D4Score,
					"distance_4":  q.D4Score,
					"total":       q.TotalScore,
					"total_score": q.TotalScore,
					"tens":        q.TensCount,
					"tens_count":  q.TensCount,
					"xs":          q.XCount,
					"x_count":     q.XCount,
				}
				qualifications[q.CategoryName] = append(qualifications[q.CategoryName], item)
			}
		}

		// 3. Fetch Matches / Brackets
		type matchRow struct {
			ID               int64   `db:"id"`
			TournamentID     int64   `db:"tournament_id"`
			CategoryID       int64   `db:"category_id"`
			CategoryName     string  `db:"category_name"`
			PhaseName        string  `db:"phase_name"`
			MatchOrder       int     `db:"match_order"`
			IsTeam           bool    `db:"is_team"`
			Athlete1Name     *string `db:"athlete1_name"`
			Athlete1Club     *string `db:"athlete1_club"`
			Athlete1Seed     *int    `db:"athlete1_seed"`
			Athlete1Score    *string `db:"athlete1_score"`
			Athlete1Sets     *string `db:"athlete1_sets"`
			Athlete1IsWinner bool    `db:"athlete1_is_winner"`
			Athlete2Name     *string `db:"athlete2_name"`
			Athlete2Club     *string `db:"athlete2_club"`
			Athlete2Seed     *int    `db:"athlete2_seed"`
			Athlete2Score    *string `db:"athlete2_score"`
			Athlete2Sets     *string `db:"athlete2_sets"`
			Athlete2IsWinner bool    `db:"athlete2_is_winner"`
			WinnerName       *string `db:"winner_name"`
			WinnerClub       *string `db:"winner_club"`
			Status           string  `db:"status"`
		}
		var matchRows []matchRow
		_ = db.Select(&matchRows, `
			SELECT tem.id, tem.tournament_id, tem.category_id, tec.category_name,
			       tem.phase_name, tem.match_order, tem.is_team,
			       tem.athlete1_name, tem.athlete1_club, tem.athlete1_seed, tem.athlete1_score, tem.athlete1_sets, tem.athlete1_is_winner,
			       tem.athlete2_name, tem.athlete2_club, tem.athlete2_seed, tem.athlete2_score, tem.athlete2_sets, tem.athlete2_is_winner,
			       tem.winner_name, tem.winner_club, tem.status
			FROM tournament_external_matches tem
			JOIN tournament_external_categories tec ON tem.category_id = tec.id
			WHERE tem.tournament_id = ?
			ORDER BY tec.category_name ASC, tem.id ASC
		`, t.ID)

		// Group matches into phases per category
		catPhaseMatches := make(map[string]map[string][]map[string]interface{})
		catPhaseOrder := make(map[string][]string)

		for _, m := range matchRows {
			cat := m.CategoryName
			phase := m.PhaseName
			if catPhaseMatches[cat] == nil {
				catPhaseMatches[cat] = make(map[string][]map[string]interface{})
			}
			if _, exists := catPhaseMatches[cat][phase]; !exists {
				catPhaseOrder[cat] = append(catPhaseOrder[cat], phase)
			}

			var a1Sets []string
			if m.Athlete1Sets != nil && *m.Athlete1Sets != "" {
				_ = json.Unmarshal([]byte(*m.Athlete1Sets), &a1Sets)
			}
			var a2Sets []string
			if m.Athlete2Sets != nil && *m.Athlete2Sets != "" {
				_ = json.Unmarshal([]byte(*m.Athlete2Sets), &a2Sets)
			}

			matchItem := map[string]interface{}{
				"id":          m.ID,
				"match_order": m.MatchOrder,
				"is_team":     m.IsTeam,
				"status":      m.Status,
				"winner_name": m.WinnerName,
				"winner_club": m.WinnerClub,
				"athlete_1": map[string]interface{}{
					"name":       m.Athlete1Name,
					"club":       m.Athlete1Club,
					"seed":       m.Athlete1Seed,
					"score":      m.Athlete1Score,
					"set_scores": a1Sets,
					"is_winner":  m.Athlete1IsWinner,
				},
				"athlete_2": map[string]interface{}{
					"name":       m.Athlete2Name,
					"club":       m.Athlete2Club,
					"seed":       m.Athlete2Seed,
					"score":      m.Athlete2Score,
					"set_scores": a2Sets,
					"is_winner":  m.Athlete2IsWinner,
				},
			}
			catPhaseMatches[cat][phase] = append(catPhaseMatches[cat][phase], matchItem)
		}

		brackets := make(map[string]interface{})
		for cat, phaseMap := range catPhaseMatches {
			var phases []map[string]interface{}
			for _, phaseName := range catPhaseOrder[cat] {
				phases = append(phases, map[string]interface{}{
					"phase_name": phaseName,
					"name":       phaseName,
					"matches":    phaseMap[phaseName],
				})
			}
			brackets[cat] = map[string]interface{}{
				"phases": phases,
			}
		}

		// 4. Fetch Final Standings
		type rankRow struct {
			ID              int64   `db:"id"`
			TournamentID    int64   `db:"tournament_id"`
			CategoryID      int64   `db:"category_id"`
			CategoryName    string  `db:"category_name"`
			Rank            int     `db:"rank"`
			ParticipantName string  `db:"participant_name"`
			ClubCode        *string `db:"club_code"`
			ClubName        *string `db:"club_name"`
			Medal           string  `db:"medal"`
			IsTeam          bool    `db:"is_team"`
		}
		var rankRows []rankRow
		_ = db.Select(&rankRows, `
			SELECT tef.id, tef.tournament_id, tef.category_id, tec.category_name,
			       tef.rank, tef.participant_name, tef.club_code, tef.club_name, tef.medal, tef.is_team
			FROM tournament_external_final_ranks tef
			JOIN tournament_external_categories tec ON tef.category_id = tec.id
			WHERE tef.tournament_id = ?
			ORDER BY tec.category_name ASC, tef.rank ASC
		`, t.ID)

		finalStandings := make(map[string][]map[string]interface{})
		for _, r := range rankRows {
			item := map[string]interface{}{
				"rank":      r.Rank,
				"name":      r.ParticipantName,
				"club_code": r.ClubCode,
				"club":      r.ClubName,
				"club_name": r.ClubName,
				"medal":     r.Medal,
				"is_team":   r.IsTeam,
			}
			finalStandings[r.CategoryName] = append(finalStandings[r.CategoryName], item)
		}

		// 5. Fetch Entries / Athletes
		type athleteRow struct {
			ID           int64   `db:"id"`
			TournamentID int64   `db:"tournament_id"`
			CategoryID   *int64  `db:"category_id"`
			CategoryName *string `db:"category_name"`
			BIB          *string `db:"bib"`
			Name         string  `db:"name"`
			ClubCode     *string `db:"club_code"`
			ClubName     *string `db:"club_name"`
			CountryCode  *string `db:"country_code"`
			Gender       string  `db:"gender"`
			TargetLane   *string `db:"target_lane"`
		}
		var athletes []athleteRow
		_ = db.Select(&athletes, `
			SELECT tea.id, tea.tournament_id, tea.category_id, tec.category_name,
			       tea.bib, tea.name, tea.club_code, tea.club_name, tea.country_code, tea.gender, tea.target_lane
			FROM tournament_external_athletes tea
			LEFT JOIN tournament_external_categories tec ON tea.category_id = tec.id
			WHERE tea.tournament_id = ?
			ORDER BY tea.id ASC
		`, t.ID)

		var entries []map[string]interface{}
		for _, a := range athletes {
			entries = append(entries, map[string]interface{}{
				"id":           a.ID,
				"bib":          a.BIB,
				"name":         a.Name,
				"club_code":    a.ClubCode,
				"club":         a.ClubName,
				"club_name":    a.ClubName,
				"country_code": a.CountryCode,
				"gender":       a.Gender,
				"category":     a.CategoryName,
				"target":       a.TargetLane,
			})
		}

		// 6. Fetch Documents
		var documents []models.TournamentExternalDocument
		_ = db.Select(&documents, `
			SELECT id, tournament_id, category_name, doc_type, title, filename, file_url 
			FROM tournament_external_documents 
			WHERE tournament_id = ? 
			ORDER BY id ASC
		`, t.ID)

		var docList []map[string]interface{}
		for _, d := range documents {
			docList = append(docList, map[string]interface{}{
				"filename": d.Filename,
				"title":    d.Title,
				"category": d.CategoryName,
				"type":     d.DocType,
				"url":      d.FileURL,
			})
		}

		// 7. Fetch Schedules
		var schedules []models.TournamentExternalSchedule
		_ = db.Select(&schedules, `
			SELECT id, tournament_id, event_date, time_range, activity, stage, sort_order 
			FROM tournament_external_schedules 
			WHERE tournament_id = ? 
			ORDER BY sort_order ASC, id ASC
		`, t.ID)

		var schedList []map[string]interface{}
		for _, s := range schedules {
			schedList = append(schedList, map[string]interface{}{
				"date":     s.EventDate,
				"time":     s.TimeRange,
				"activity": s.Activity,
				"title":    s.Activity,
				"stage":    s.Stage,
			})
		}

		// Base payload from JSON data if available
		parsedData := make(map[string]interface{})
		if t.DataJSON != "" {
			_ = json.Unmarshal([]byte(t.DataJSON), &parsedData)
		}

		// Augment or fill with relational data if available
		if len(categoryNames) > 0 {
			parsedData["categories"] = categoryNames
		}
		if len(qualifications) > 0 {
			parsedData["qualifications"] = qualifications
		}
		if len(teamQualifications) > 0 {
			parsedData["team_qualifications"] = teamQualifications
		}
		if len(brackets) > 0 {
			parsedData["brackets"] = brackets
		}
		if len(finalStandings) > 0 {
			parsedData["final_standings"] = finalStandings
		}
		if len(entries) > 0 {
			parsedData["entries"] = entries
		}
		if len(docList) > 0 {
			parsedData["documents"] = docList
		}

		// For schedule: if relational schedules have valid non-empty time/date, use them; otherwise keep parsedData["schedule"] from data_json
		hasValidRelationalSched := false
		for _, s := range schedList {
			if (s["time"] != nil && s["time"] != "") || (s["date"] != nil && s["date"] != "") {
				hasValidRelationalSched = true
				break
			}
		}
		if hasValidRelationalSched {
			parsedData["schedule"] = schedList
		}

		c.JSON(http.StatusOK, gin.H{
			"tournament": t,
			"data":       parsedData,
		})
	}
}

