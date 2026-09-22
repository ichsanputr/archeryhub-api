package handler

import (
	"Archeris-api/utils"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// RootTournamentItem represents a tournament item in root list
type RootTournamentItem struct {
	UUID              string     `json:"uuid" db:"uuid"`
	Code              string     `json:"code" db:"code"`
	Name              string     `json:"name" db:"name"`
	Slug              string     `json:"slug" db:"slug"`
	Venue             *string    `json:"venue" db:"venue"`
	City              *string    `json:"city" db:"city"`
	StartDate         *time.Time `json:"start_date" db:"start_date"`
	EndDate           *time.Time `json:"end_date" db:"end_date"`
	Status            string     `json:"status" db:"status"`
	BannerURL         *string    `json:"banner_url" db:"banner_url"`
	LogoURL           *string    `json:"logo_url" db:"logo_url"`
	EntryFee          float64    `json:"entry_fee" db:"entry_fee"`
	QuotaType         string     `json:"quota_type" db:"quota_type"`
	OrganizerID       *string    `json:"organizer_id" db:"organizer_id"`
	OrganizerName     string     `json:"organizer_name" db:"organizer_name"`
	OrganizerEmail    string     `json:"organizer_email" db:"organizer_email"`
	OrganizerAvatar   *string    `json:"organizer_avatar" db:"organizer_avatar"`
	OrganizerSlug     *string    `json:"organizer_slug" db:"organizer_slug"`
	ParticipantCount  int        `json:"participant_count" db:"participant_count"`
	CategoryCount     int        `json:"category_count" db:"category_count"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

// RootListTournaments returns all internal tournaments with organizer info, stats, and filters
func RootListTournaments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		search := strings.TrimSpace(c.Query("search"))
		status := strings.TrimSpace(c.Query("status"))
		quotaType := strings.TrimSpace(c.Query("quota_type"))
		sortBy := strings.TrimSpace(c.Query("sort_by"))
		order := strings.ToUpper(strings.TrimSpace(c.Query("order")))
		limit, offset, page := utils.GetPaginationParams(c)

		whereClauses := []string{"1=1"}
		args := []interface{}{}

		if search != "" {
			searchPattern := "%" + search + "%"
			whereClauses = append(whereClauses, "(t.name LIKE ? OR t.code LIKE ? OR t.slug LIKE ? OR o.name LIKE ? OR o.email LIKE ?)")
			args = append(args, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern)
		}

		if status != "" && status != "all" {
			whereClauses = append(whereClauses, "t.status = ?")
			args = append(args, status)
		}

		if quotaType != "" && quotaType != "all" {
			if quotaType == "free" {
				whereClauses = append(whereClauses, "(t.quota_type = 'free' OR t.quota_type IS NULL OR t.quota_type = '')")
			} else {
				whereClauses = append(whereClauses, "t.quota_type = ?")
				args = append(args, quotaType)
			}
		}

		whereSQL := strings.Join(whereClauses, " AND ")

		// Count total matching
		countQuery := `
			SELECT COUNT(*) 
			FROM tournaments t
			LEFT JOIN organizers o ON t.organizer_id = o.uuid
			WHERE ` + whereSQL

		var total int
		if err := db.Get(&total, countQuery, args...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung jumlah turnamen", "details": err.Error()})
			return
		}

		// Sort column validation
		orderColumn := "t.created_at"
		switch sortBy {
		case "name":
			orderColumn = "t.name"
		case "start_date":
			orderColumn = "t.start_date"
		case "status":
			orderColumn = "t.status"
		case "participants":
			orderColumn = "participant_count"
		case "created_at":
			orderColumn = "t.created_at"
		}

		if order != "ASC" && order != "DESC" {
			order = "DESC"
		}

		query := fmt.Sprintf(`
			SELECT 
				t.uuid,
				COALESCE(t.code, '') as code,
				t.name,
				t.slug,
				t.venue,
				t.city,
				t.start_date,
				t.end_date,
				t.status,
				t.banner_url,
				t.logo_url,
				COALESCE(t.entry_fee, 0) as entry_fee,
				COALESCE(t.quota_type, 'free') as quota_type,
				t.organizer_id,
				COALESCE(o.name, 'Organizer') as organizer_name,
				COALESCE(o.email, '') as organizer_email,
				o.avatar_url as organizer_avatar,
				o.slug as organizer_slug,
				COALESCE((SELECT COUNT(*) FROM tournament_participants tp WHERE tp.tournament_id = t.uuid), 0) as participant_count,
				COALESCE((SELECT COUNT(*) FROM tournament_categories tc WHERE tc.tournament_id = t.uuid), 0) as category_count,
				t.created_at
			FROM tournaments t
			LEFT JOIN organizers o ON t.organizer_id = o.uuid
			WHERE %s
			ORDER BY %s %s
			LIMIT ? OFFSET ?
		`, whereSQL, orderColumn, order)

		queryArgs := append(args, limit, offset)
		var tournaments []RootTournamentItem
		if err := db.Select(&tournaments, query, queryArgs...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar turnamen", "details": err.Error()})
			return
		}

		if tournaments == nil {
			tournaments = []RootTournamentItem{}
		}

		// Global summary stats for cards
		var stats struct {
			TotalTournaments int `db:"total_tournaments"`
			Published        int `db:"published"`
			Draft            int `db:"draft"`
			TotalAthletes    int `db:"total_athletes"`
			CountFree        int `db:"count_free"`
			CountStandard    int `db:"count_standard"`
			CountElite       int `db:"count_elite"`
		}

		statQuery := `
			SELECT 
				COUNT(*) as total_tournaments,
				COALESCE(SUM(CASE WHEN status = 'published' THEN 1 ELSE 0 END), 0) as published,
				COALESCE(SUM(CASE WHEN status = 'draft' THEN 1 ELSE 0 END), 0) as draft,
				COALESCE((SELECT COUNT(*) FROM tournament_participants), 0) as total_athletes,
				COALESCE(SUM(CASE WHEN quota_type = 'free' OR quota_type IS NULL OR quota_type = '' THEN 1 ELSE 0 END), 0) as count_free,
				COALESCE(SUM(CASE WHEN quota_type = 'standard' THEN 1 ELSE 0 END), 0) as count_standard,
				COALESCE(SUM(CASE WHEN quota_type = 'elite' THEN 1 ELSE 0 END), 0) as count_elite
			FROM tournaments
		`
		_ = db.Get(&stats, statQuery)

		meta := utils.CalculatePagination(total, limit, offset, page)
		c.JSON(http.StatusOK, gin.H{
			"tournaments": tournaments,
			"meta":        meta,
			"stats": gin.H{
				"total":          stats.TotalTournaments,
				"published":      stats.Published,
				"draft":          stats.Draft,
				"total_athletes": stats.TotalAthletes,
				"packages": gin.H{
					"free":     stats.CountFree,
					"standard": stats.CountStandard,
					"elite":    stats.CountElite,
				},
			},
		})
	}
}

// RootDeleteTournament deletes an internal tournament and refunds +1 quota credit to the organizer
func RootDeleteTournament(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Resolve uuid or slug
		var eventInfo struct {
			UUID        string         `db:"uuid"`
			Name        string         `db:"name"`
			Status      string         `db:"status"`
			QuotaType   sql.NullString `db:"quota_type"`
			OrganizerID sql.NullString `db:"organizer_id"`
		}

		err := db.Get(&eventInfo, `SELECT uuid, name, status, quota_type, organizer_id FROM tournaments WHERE uuid = ? OR slug = ?`, id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		actualID := eventInfo.UUID

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database"})
			return
		}
		defer tx.Rollback()

		// Comprehensive Clean Cascade Deletion:
		// 1. Elimination arrow scores
		_, _ = tx.Exec(`
			DELETE FROM elimination_match_arrow_scores 
			WHERE match_end_uuid IN (
				SELECT eme.uuid FROM elimination_match_ends eme
				JOIN elimination_matches em ON eme.match_uuid = em.uuid
				JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid
				WHERE eb.tournament_uuid = ?
			)
		`, actualID)

		// 2. Elimination match ends
		_, _ = tx.Exec(`
			DELETE FROM elimination_match_ends 
			WHERE match_uuid IN (
				SELECT em.uuid FROM elimination_matches em
				JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid
				WHERE eb.tournament_uuid = ?
			)
		`, actualID)

		// 3. Elimination matches
		_, _ = tx.Exec(`DELETE FROM elimination_matches WHERE bracket_uuid IN (SELECT uuid FROM elimination_brackets WHERE tournament_uuid = ?)`, actualID)

		// 4. Elimination entries
		_, _ = tx.Exec(`
			DELETE FROM elimination_entries 
			WHERE bracket_uuid IN (SELECT uuid FROM elimination_brackets WHERE tournament_uuid = ?)
			   OR participant_uuid IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
		`, actualID, actualID)

		// 5. Target board elimination
		_, _ = tx.Exec(`
			DELETE FROM target_board_elimination 
			WHERE bracket_uuid IN (SELECT uuid FROM elimination_brackets WHERE tournament_uuid = ?)
			   OR category_uuid IN (SELECT uuid FROM tournament_categories WHERE tournament_id = ?)
		`, actualID, actualID)

		// 6. Elimination brackets
		_, _ = tx.Exec(`DELETE FROM elimination_brackets WHERE tournament_uuid = ?`, actualID)

		// 7. Qualification arrow scores
		_, _ = tx.Exec(`
			DELETE FROM qualification_arrow_scores 
			WHERE end_score_uuid IN (
				SELECT qes.uuid FROM qualification_end_scores qes
				WHERE qes.participant_uuid IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
				   OR qes.session_uuid IN (SELECT qs.uuid FROM qualification_sessions qs WHERE qs.tournament_uuid = ?)
			)
		`, actualID, actualID)

		// 8. Qualification end scores
		_, _ = tx.Exec(`
			DELETE FROM qualification_end_scores 
			WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
			   OR session_uuid IN (SELECT qs.uuid FROM qualification_sessions qs WHERE qs.tournament_uuid = ?)
		`, actualID, actualID)

		// 9. Qualification target assignments
		_, _ = tx.Exec(`
			DELETE FROM qualification_target_assignments 
			WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
			   OR session_uuid IN (SELECT qs.uuid FROM qualification_sessions qs WHERE qs.tournament_uuid = ?)
			   OR target_uuid IN (SELECT uuid FROM tournament_targets WHERE tournament_uuid = ?)
		`, actualID, actualID, actualID)

		// 10. Target board qualification
		_, _ = tx.Exec(`
			DELETE FROM target_board_qualification 
			WHERE session_uuid IN (SELECT qs.uuid FROM qualification_sessions qs WHERE qs.tournament_uuid = ?)
			   OR category_uuid IN (SELECT tc.uuid FROM tournament_categories tc WHERE tc.tournament_id = ?)
		`, actualID, actualID)

		// 11. Qualification session categories
		_, _ = tx.Exec(`
			DELETE FROM qualification_session_categories 
			WHERE session_uuid IN (SELECT qs.uuid FROM qualification_sessions qs WHERE qs.tournament_uuid = ?)
			   OR category_uuid IN (SELECT tc.uuid FROM tournament_categories tc WHERE tc.tournament_id = ?)
		`, actualID, actualID)

		// 12. Qualification sessions
		_, _ = tx.Exec(`DELETE FROM qualification_sessions WHERE tournament_uuid = ?`, actualID)

		// 13. Tournament target boards & physical targets
		_, _ = tx.Exec(`DELETE FROM tournament_target_boards WHERE tournament_uuid = ?`, actualID)
		_, _ = tx.Exec(`DELETE FROM tournament_targets WHERE tournament_uuid = ?`, actualID)

		// 14. Team members & teams
		_, _ = tx.Exec(`
			DELETE FROM team_members 
			WHERE team_id IN (SELECT uuid FROM teams WHERE tournament_id = ? OR event_id = ?)
			   OR participant_id IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
		`, actualID, actualID, actualID)
		_, _ = tx.Exec(`DELETE FROM teams WHERE tournament_id = ? OR event_id = ?`, actualID, actualID)

		// 15. Certificates & upload batches
		_, _ = tx.Exec(`
			DELETE FROM archer_certificates 
			WHERE tournament_id = ? 
			   OR registration_id IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
		`, actualID, actualID)
		_, _ = tx.Exec(`DELETE FROM tournament_certificates WHERE tournament_id = ?`, actualID)
		_, _ = tx.Exec(`DELETE FROM certificate_upload_batches WHERE tournament_id = ?`, actualID)

		// 16. Broadcasts, media, and tournament gallery images
		_, _ = tx.Exec(`DELETE FROM broadcasts WHERE tournament_id = ?`, actualID)
		_, _ = tx.Exec(`DELETE FROM media WHERE tournament_id = ?`, actualID)
		_, _ = tx.Exec(`DELETE FROM tournament_images WHERE tournament_id = ?`, actualID)

		// 17. Schedules & schedule items
		_, _ = tx.Exec(`DELETE FROM tournament_schedule_items WHERE tournament_id = ?`, actualID)
		_, _ = tx.Exec(`DELETE FROM tournament_schedules WHERE tournament_id = ?`, actualID)

		// 18. Assignment history, scorekeeper logs, activity logs, reset codes
		_, _ = tx.Exec(`DELETE FROM assignment_history WHERE tournament_uuid = ?`, actualID)
		_, _ = tx.Exec(`DELETE FROM scorekeeper_logs WHERE tournament_uuid = ?`, actualID)
		_, _ = tx.Exec(`DELETE FROM activity_logs WHERE tournament_id = ?`, actualID)
		_, _ = tx.Exec(`DELETE FROM event_reset_codes WHERE event_id = ?`, actualID)

		// 19. Payment transactions associated with this tournament or participant registrations
		_, _ = tx.Exec(`
			DELETE FROM payment_transactions 
			WHERE tournament_id = ? 
			   OR (registration_id IS NOT NULL AND registration_id != '' AND registration_id IN (
			       SELECT uuid FROM tournament_participants WHERE tournament_id = ?
			   ))
		`, actualID, actualID)

		// 20. Tournament participants (athletes' master records in 'archers' remain 100% intact)
		_, _ = tx.Exec(`DELETE FROM tournament_participants WHERE tournament_id = ?`, actualID)

		// 21. Tournament categories (reference categories in 'ref_*' remain 100% intact)
		_, _ = tx.Exec(`DELETE FROM tournament_categories WHERE tournament_id = ?`, actualID)

		// 22. Delete the tournament record itself
		result, err := tx.Exec(`DELETE FROM tournaments WHERE uuid = ?`, actualID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus record turnamen: " + err.Error()})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan atau sudah dihapus"})
			return
		}

		// Refund quota credit (+1) to the organizer
		quotaTypeStr := "free"
		if eventInfo.QuotaType.Valid && eventInfo.QuotaType.String != "" {
			quotaTypeStr = strings.ToLower(eventInfo.QuotaType.String)
		}

		var orgName string
		if eventInfo.OrganizerID.Valid && eventInfo.OrganizerID.String != "" {
			orgID := eventInfo.OrganizerID.String
			switch quotaTypeStr {
			case "standard":
				_, _ = tx.Exec(`UPDATE organizers SET quota_standard = quota_standard + 1 WHERE uuid = ? OR user_id = ?`, orgID, orgID)
			case "elite":
				_, _ = tx.Exec(`UPDATE organizers SET quota_elite = quota_elite + 1 WHERE uuid = ? OR user_id = ?`, orgID, orgID)
			default: // "free"
				_, _ = tx.Exec(`UPDATE organizers SET quota_free = quota_free + 1 WHERE uuid = ? OR user_id = ?`, orgID, orgID)
			}

			// Get organizer name for response
			_ = tx.Get(&orgName, `SELECT name FROM organizers WHERE uuid = ? OR user_id = ?`, orgID, orgID)
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan transaksi penghapusan"})
			return
		}

		// Audit Log
		adminID, _ := c.Get("user_id")
		adminIDStr := fmt.Sprintf("%v", adminID)
		logMsg := fmt.Sprintf("Root admin deleted tournament '%s' (UUID: %s) and refunded 1 %s package credit to organizer '%s'", eventInfo.Name, actualID, quotaTypeStr, orgName)
		utils.LogActivity(db, adminIDStr, "", "root_delete_tournament", "tournament", actualID, logMsg, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{
			"message":              fmt.Sprintf("Turnamen '%s' berhasil dihapus dan 1 kuota kredit (%s) berhasil dikembalikan ke akun EO", eventInfo.Name, strings.ToUpper(quotaTypeStr)),
			"uuid":                 actualID,
			"refunded_quota_type":  quotaTypeStr,
			"organizer_name":       orgName,
		})
	}
}
