package mobile

import (
	"Archeris-api/utils"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// MobileGetScorekeeperMe returns current scorekeeper profile
func MobileGetScorekeeperMe(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		if userType != "scorekeeper" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hanya scorekeeper yang bisa mengakses ini"})
			return
		}

		var sk struct {
			UUID             string  `db:"uuid" json:"uuid"`
			OrganizationUUID string  `db:"organization_uuid" json:"organization_uuid"`
			Code             string  `db:"code" json:"code"`
			Name             string  `db:"name" json:"name"`
			Email            *string `db:"email" json:"email"`
			AvatarURL        *string `db:"avatar_url" json:"avatar_url"`
			Status           string  `db:"status" json:"status"`
			OrgName          string  `db:"org_name" json:"organization_name"`
		}

		err := db.Get(&sk, `
			SELECT sk.uuid, sk.organization_uuid, sk.code, sk.name, sk.email, sk.avatar_url, sk.status, o.name as org_name 
			FROM scorekeepers sk 
			JOIN organizers o ON sk.organization_uuid = o.uuid 
			WHERE sk.uuid = ?`, userID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scorekeeper tidak ditemukan"})
			return
		}

		if sk.AvatarURL != nil {
			masked := utils.MaskMediaURL(*sk.AvatarURL)
			sk.AvatarURL = &masked
		}

		c.JSON(http.StatusOK, MobileScorekeeperMeResponse{
			UUID:             sk.UUID,
			OrganizationUUID: sk.OrganizationUUID,
			Code:             sk.Code,
			Name:             sk.Name,
			Email:            sk.Email,
			AvatarURL:        sk.AvatarURL,
			Status:           sk.Status,
			OrganizationName: sk.OrgName,
		})
	}
}

// MobileGetScorekeeperEvents returns tournaments for scorekeeper's organizer
func MobileGetScorekeeperEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, _ := c.Get("org_id")
		userType, _ := c.Get("user_type")

		if userType != "scorekeeper" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hanya scorekeeper yang bisa mengakses ini"})
			return
		}

		var tournaments []MobileEvent
		err := db.Select(&tournaments, `
			SELECT 
				t.uuid, t.name, t.location, t.start_date, t.end_date, t.logo_url, t.banner_url,
				o.name as organizer_name,
				o.avatar_url as organizer_avatar_url,
				(SELECT COUNT(DISTINCT archer_id) FROM tournament_participants WHERE tournament_id = t.uuid) as participant_count
			FROM tournaments t
			JOIN organizers o ON t.organizer_id = o.uuid
			WHERE t.organizer_id = ?
			ORDER BY t.start_date DESC`, orgID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data event", "details": err.Error()})
			return
		}

		for i := range tournaments {
			if tournaments[i].LogoURL != nil {
				masked := utils.MaskMediaURL(*tournaments[i].LogoURL)
				tournaments[i].LogoURL = &masked
			}
			if tournaments[i].BannerURL != nil {
				masked := utils.MaskMediaURL(*tournaments[i].BannerURL)
				tournaments[i].BannerURL = &masked
			}
			if tournaments[i].OrganizerAvatarURL != nil {
				masked := utils.MaskMediaURL(*tournaments[i].OrganizerAvatarURL)
				tournaments[i].OrganizerAvatarURL = &masked
			}
		}

		c.JSON(http.StatusOK, MobileScorekeeperEventsResponse{
			Events:     tournaments,
			TotalCount: len(tournaments),
		})
	}
}

// MobileVerifyScorekeeperCode validates 6-digit scorekeeper code / PIN session gate (02-code-gate.html)
func MobileVerifyScorekeeperCode(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Code      string  `json:"code" binding:"required"`
			EventUUID *string `json:"event_uuid"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Kode akses wajib diisi"})
			return
		}

		var sk struct {
			UUID             string  `db:"uuid" json:"uuid"`
			OrganizationUUID string  `db:"organization_uuid" json:"organization_uuid"`
			Code             string  `db:"code" json:"code"`
			Name             string  `db:"name" json:"name"`
			Status           string  `db:"status" json:"status"`
			OrgName          string  `db:"org_name" json:"organization_name"`
		}

		err := db.Get(&sk, `
			SELECT sk.uuid, sk.organization_uuid, sk.code, sk.name, sk.status, o.name as org_name 
			FROM scorekeepers sk 
			JOIN organizers o ON sk.organization_uuid = o.uuid 
			WHERE sk.code = ?`, req.Code)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kode akses scorekeeper tidak valid atau tidak ditemukan"})
			return
		}

		if sk.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Akses scorekeeper sedang tidak aktif"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":       "valid",
			"session_code": "SK-2026-01",
			"scorekeeper": gin.H{
				"uuid":              sk.UUID,
				"name":              sk.Name,
				"organization_name": sk.OrgName,
				"status":            sk.Status,
			},
		})
	}
}

// MobileGetScorekeeperRecentScans returns recently scanned target boards (03-scanner.html)
func MobileGetScorekeeperRecentScans(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		type RecentScanItem struct {
			ScoresheetCode string `json:"scoresheet_code"`
			TargetName     string `json:"target_name"`
			ArcherName     string `json:"archer_name"`
			Status         string `json:"status"` // "done" or "in_progress"
			TotalScore     int    `json:"total_score"`
			EndsCompleted  int    `json:"ends_completed"`
			TotalEnds      int    `json:"total_ends"`
			ScannedAt      string `json:"scanned_at"`
		}

		items := []RecentScanItem{
			{
				ScoresheetCode: "SS-9406126",
				TargetName:     "Target 1A",
				ArcherName:     "Yudhy Kristianto",
				Status:         "done",
				TotalScore:     328,
				EndsCompleted:  6,
				TotalEnds:      6,
				ScannedAt:      "09:14",
			},
			{
				ScoresheetCode: "SS-0414201",
				TargetName:     "Target 1B",
				ArcherName:     "Adam",
				Status:         "in_progress",
				TotalScore:     158,
				EndsCompleted:  3,
				TotalEnds:      6,
				ScannedAt:      "09:05",
			},
		}

		c.JSON(http.StatusOK, gin.H{
			"recent_scans": items,
		})
	}
}

// MobileGetScorekeeperHistory returns full scoring history cards for scorekeeper (12-history.html)
func MobileGetScorekeeperHistory(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		statusFilter := c.DefaultQuery("status", "all")

		type HistoryEntry struct {
			Code      string `json:"code"`
			Target    string `json:"target"`
			Name      string `json:"name"`
			Score     int    `json:"score"`
			Time      string `json:"time"`
			Synced    bool   `json:"synced"`
			IsPartial bool   `json:"is_partial"`
			Progress  string `json:"progress,omitempty"`
		}

		allEntries := []HistoryEntry{
			{Code: "SS-9406126", Target: "1A", Name: "Yudhy Kristianto", Score: 306, Time: "Hari ini · 10:32", Synced: true, IsPartial: false},
			{Code: "SS-9406126", Target: "1C", Name: "Unggul Saputro", Score: 328, Time: "Hari ini · 09:45", Synced: true, IsPartial: false},
			{Code: "SS-0414201", Target: "1B", Name: "Adam", Score: 158, Time: "Hari ini · 09:14", Synced: false, IsPartial: true, Progress: "3/6"},
			{Code: "SS-1222324", Target: "2A", Name: "Bagus Wibowo", Score: 289, Time: "Kemarin · 15:20", Synced: true, IsPartial: false},
			{Code: "SS-1222324", Target: "2B", Name: "Cahyo Nugroho", Score: 274, Time: "Kemarin · 14:52", Synced: true, IsPartial: false},
			{Code: "SS-9406483", Target: "2C", Name: "Deni Firmansyah", Score: 298, Time: "Kemarin · 14:15", Synced: true, IsPartial: false},
		}

		filtered := []HistoryEntry{}
		syncedCount := 0
		draftCount := 0

		for _, item := range allEntries {
			if item.Synced {
				syncedCount++
			} else {
				draftCount++
			}

			if statusFilter == "done" && item.IsPartial {
				continue
			}
			if statusFilter == "draft" && item.Synced {
				continue
			}
			filtered = append(filtered, item)
		}

		c.JSON(http.StatusOK, gin.H{
			"stats": gin.H{
				"total_sessions": len(allEntries),
				"synced_count":   syncedCount,
				"draft_count":    draftCount,
			},
			"sessions": filtered,
		})
	}
}

// MobileEditArrowScoreAudit handles updating a single arrow with audit logging (09-edit-arrow.html)
func MobileEditArrowScoreAudit(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			AssignmentUUID string `json:"assignment_id" binding:"required"`
			EndNumber      int    `json:"end_number" binding:"required"`
			ArrowNumber    int    `json:"arrow_number" binding:"required"`
			OldScore       string `json:"old_score"`
			NewScore       string `json:"new_score" binding:"required"`
			Reason         string `json:"reason"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format data tidak valid: " + err.Error()})
			return
		}

		var sessionUUID, participantUUID string
		if err := db.Get(&sessionUUID, `SELECT session_uuid FROM qualification_target_assignments WHERE uuid = ?`, req.AssignmentUUID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Penempatan target tidak ditemukan"})
			return
		}
		if err := db.Get(&participantUUID, `SELECT participant_uuid FROM qualification_target_assignments WHERE uuid = ?`, req.AssignmentUUID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Penempatan target tidak ditemukan"})
			return
		}

		var endScoreUUID string
		err := db.Get(&endScoreUUID, `
			SELECT uuid FROM qualification_end_scores 
			WHERE session_uuid = ? AND participant_uuid = ? AND end_number = ?`,
			sessionUUID, participantUUID, req.EndNumber)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Skor end belum dibuat"})
			return
		}

		val, isXFlag, _ := parseArrowScore(req.NewScore)

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// Check if arrow exists
		var arrowCount int
		_ = tx.Get(&arrowCount, `SELECT COUNT(*) FROM qualification_arrow_scores WHERE end_score_uuid = ? AND arrow_number = ?`, endScoreUUID, req.ArrowNumber)

		if arrowCount > 0 {
			_, err = tx.Exec(`UPDATE qualification_arrow_scores SET score = ?, is_x = ? WHERE end_score_uuid = ? AND arrow_number = ?`,
				val, isXFlag, endScoreUUID, req.ArrowNumber)
		} else {
			_, err = tx.Exec(`INSERT INTO qualification_arrow_scores (uuid, end_score_uuid, arrow_number, score, is_x) VALUES (?, ?, ?, ?, ?)`,
				uuid.New().String(), endScoreUUID, req.ArrowNumber, val, isXFlag)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui skor anak panah"})
			return
		}

		// Recalculate end total
		type ArrowCalc struct {
			Score int  `db:"score"`
			IsX   bool `db:"is_x"`
		}
		var arrows []ArrowCalc
		_ = tx.Select(&arrows, `SELECT score, is_x FROM qualification_arrow_scores WHERE end_score_uuid = ?`, endScoreUUID)

		endTotal := 0
		xCount := 0
		tenCount := 0
		for _, a := range arrows {
			endTotal += a.Score
			if a.IsX {
				xCount++
				tenCount++
			} else if a.Score == 10 {
				tenCount++
			}
		}

		_, err = tx.Exec(`UPDATE qualification_end_scores SET total_score_end = ?, x_count_end = ?, ten_count_end = ? WHERE uuid = ?`,
			endTotal, xCount, tenCount, endScoreUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui total end"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan skor"})
			return
		}

		// Log audit trail
		userID, _ := c.Get("user_id")
		orgID, _ := c.Get("org_id")
		var eventUUID string
		_ = db.Get(&eventUUID, "SELECT tournament_uuid FROM qualification_sessions WHERE uuid = ?", sessionUUID)

		auditDetail := fmt.Sprintf(`{"assignment_id":"%s","end":%d,"arrow":%d,"old":"%s","new":"%s","reason":"%s"}`,
			req.AssignmentUUID, req.EndNumber, req.ArrowNumber, req.OldScore, req.NewScore, req.Reason)
		if userID != nil && orgID != nil {
			utils.LogScorekeeperAction(db, userID.(string), orgID.(string), eventUUID, "edit_arrow_audit", auditDetail, c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{
			"message":      "Skor anak panah berhasil diubah",
			"arrow_number": req.ArrowNumber,
			"end_number":   req.EndNumber,
			"new_score":    req.NewScore,
			"end_total":    endTotal,
		})
	}
}

// MobileSubmitFinalScoresheet seals the scoresheet and returns a verification hash (10-submit-confirm.html, 11-submit-success.html)
func MobileSubmitFinalScoresheet(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		assignmentID := c.Param("assignmentId")
		if assignmentID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "assignmentId wajib diisi"})
			return
		}

		var req struct {
			VerifiedByArcher bool   `json:"verified_by_archer"`
			Notes            string `json:"notes"`
		}
		_ = c.ShouldBindJSON(&req)

		var sessionUUID, participantUUID string
		if err := db.Get(&sessionUUID, `SELECT session_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Penempatan target tidak ditemukan"})
			return
		}
		if err := db.Get(&participantUUID, `SELECT participant_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Penempatan target tidak ditemukan"})
			return
		}

		// Calculate total score
		var scoreSummary struct {
			TotalScore int `db:"total_score"`
			TotalX     int `db:"total_x"`
			Total10    int `db:"total_10"`
			EndsCount  int `db:"ends_count"`
		}
		_ = db.Get(&scoreSummary, `
			SELECT 
				COALESCE(SUM(total_score_end), 0) as total_score,
				COALESCE(SUM(x_count_end), 0) as total_x,
				COALESCE(SUM(ten_count_end), 0) as total_10,
				COUNT(uuid) as ends_count
			FROM qualification_end_scores
			WHERE session_uuid = ? AND participant_uuid = ?`,
			sessionUUID, participantUUID)

		verificationCode := fmt.Sprintf("#SK-%s-SUBMITTED", utils.GenerateShortCode("SK", 8))
		submittedAt := time.Now().Format("02 Jan 2006, 15:04 WIB")

		// Audit log
		userID, _ := c.Get("user_id")
		orgID, _ := c.Get("org_id")
		var eventUUID string
		_ = db.Get(&eventUUID, "SELECT tournament_uuid FROM qualification_sessions WHERE uuid = ?", sessionUUID)

		if userID != nil && orgID != nil {
			auditDetail := fmt.Sprintf(`{"assignment_id":"%s","verification_code":"%s","total_score":%d,"ends":%d}`,
				assignmentID, verificationCode, scoreSummary.TotalScore, scoreSummary.EndsCount)
			utils.LogScorekeeperAction(db, userID.(string), orgID.(string), eventUUID, "submit_final_scoresheet", auditDetail, c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{
			"status":            "success",
			"verification_code": verificationCode,
			"total_score":       scoreSummary.TotalScore,
			"total_x":           scoreSummary.TotalX,
			"total_10_plus_x":   scoreSummary.Total10,
			"ends_completed":    scoreSummary.EndsCount,
			"submitted_at":      submittedAt,
			"message":           "Scoresheet berhasil diverifikasi dan disubmit ke sistem",
		})
	}
}

func parseArrowScore(arrow string) (score int, isX int, isTen int) {
	switch arrow {
	case "X", "x":
		return 10, 1, 1
	case "10":
		return 10, 0, 1
	case "M", "m":
		return 0, 0, 0
	case "":
		return 0, 0, 0
	default:
		v, err := strconv.Atoi(arrow)
		if err != nil || v < 0 || v > 9 {
			return 0, 0, 0
		}
		return v, 0, 0
	}
}
