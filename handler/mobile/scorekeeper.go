package mobile

import (
	"Archeris-api/utils"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
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
				t.uuid, COALESCE(t.slug, '') as slug, t.name, 
				COALESCE(t.location, '') as location,
				COALESCE(t.city, '') as city,
				COALESCE(t.status, 'published') as status,
				CAST(COALESCE(DATE_FORMAT(t.start_date, '%Y-%m-%d'), '') AS CHAR) as start_date,
				CAST(COALESCE(DATE_FORMAT(t.end_date, '%Y-%m-%d'), '') AS CHAR) as end_date,
				t.logo_url, t.banner_url,
				COALESCE(o.name, '') as organizer_name,
				o.avatar_url as organizer_avatar_url,
				(SELECT COUNT(DISTINCT archer_id) FROM tournament_participants WHERE tournament_id = t.uuid) as participant_count,
				(SELECT COUNT(*) FROM tournament_categories WHERE tournament_id = t.uuid) as category_count,
				COALESCE(t.entry_fee, 0.0) as entry_fee
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


// MobileScorekeeperScanOCR handles scoresheet photo upload and proxies to Python OCR microservice
func MobileScorekeeperScanOCR(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, _ := c.Get("user_type")
		if userType != "scorekeeper" && userType != "organizer" && userType != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hanya juri scorekeeper yang berwenang memindai lembar skor"})
			return
		}

		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File foto lembar skor wajib diunggah (file)"})
			return
		}
		defer file.Close()

		assignmentUUID := c.DefaultPostForm("assignment_uuid", "")
		arrowsPerEndStr := c.DefaultPostForm("arrows_per_end", "6")
		endsPerSessionStr := c.DefaultPostForm("ends_per_session", "6")
		sessionsCountStr := c.DefaultPostForm("sessions_count", "1")
		matchType := c.DefaultPostForm("match_type", "qualification")

		// If assignment UUID provided, lookup default ends & arrows from tournament/session if not explicitly passed
		if assignmentUUID != "" && (arrowsPerEndStr == "6" && endsPerSessionStr == "6") {
			var config struct {
				ArrowsPerEnd  int `db:"arrows_per_end"`
				TotalEnds     int `db:"total_ends"`
			}
			err := db.Get(&config, `
				SELECT 
					COALESCE(c.arrows_per_end, 6) as arrows_per_end,
					COALESCE(c.total_ends, 6) as total_ends
				FROM qualification_target_assignments ta
				JOIN qualification_sessions qs ON ta.session_uuid = qs.uuid
				LEFT JOIN tournament_categories c ON ta.category_uuid = c.uuid
				WHERE ta.uuid = ?`, assignmentUUID)
			if err == nil && config.ArrowsPerEnd > 0 && config.TotalEnds > 0 {
				arrowsPerEndStr = strconv.Itoa(config.ArrowsPerEnd)
				endsPerSessionStr = strconv.Itoa(config.TotalEnds)
			}
		}

		// Prepare multipart request to Python OCR Microservice
		ocrURL := os.Getenv("SCORECARD_OCR_SERVICE_URL")
		if ocrURL == "" {
			ocrURL = "http://localhost:8002/api/v1/scan-scorecard"
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add form fields
		_ = writer.WriteField("arrows_per_end", arrowsPerEndStr)
		_ = writer.WriteField("ends_per_session", endsPerSessionStr)
		_ = writer.WriteField("sessions_count", sessionsCountStr)
		_ = writer.WriteField("match_type", matchType)
		if assignmentUUID != "" {
			_ = writer.WriteField("scoresheet_id", assignmentUUID)
		}

		// Add file
		part, err := writer.CreateFormFile("file", header.Filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyiapkan payload OCR"})
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyalin file foto"})
			return
		}
		_ = writer.Close()

		// Call Python OCR Service with timeout
		client := &http.Client{Timeout: 30 * time.Second}
		req, err := http.NewRequest("POST", ocrURL, body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat request OCR"})
			return
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Layanan OCR tidak dapat dihubungi", "details": err.Error()})
			return
		}
		defer resp.Body.Close()

		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca respons dari OCR"})
			return
		}

		if resp.StatusCode != http.StatusOK {
			c.JSON(resp.StatusCode, gin.H{"error": "Gagal memproses gambar pada mesin OCR", "raw": string(respBytes)})
			return
		}

		var ocrResult map[string]interface{}
		if err := json.Unmarshal(respBytes, &ocrResult); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mem-parsing hasil OCR"})
			return
		}

		// Return formatted response
		c.JSON(http.StatusOK, gin.H{
			"status":             "success",
			"assignment_uuid":    assignmentUUID,
			"tables_detected":    ocrResult["tables_detected"],
			"total_score":        ocrResult["total_score"],
			"total_tens_plus_xs": ocrResult["total_tens_plus_xs"],
			"total_xs":           ocrResult["total_xs"],
			"confidence_score":   ocrResult["metrics"].(map[string]interface{})["average_confidence"],
			"ends":               ocrResult["ends"],
			"sessions":           ocrResult["sessions"],
		})
	}
}

