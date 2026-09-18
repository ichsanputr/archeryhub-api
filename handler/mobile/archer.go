package mobile

import (
	"Archeris-api/utils"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func buildMobileQRCodeDataURL(qrRaw *string) *string {
	if qrRaw == nil || *qrRaw == "" {
		return nil
	}
	png, err := utils.GenerateQRCode(*qrRaw, 256)
	if err != nil {
		return nil
	}
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	return &dataURL
}


// MobileGetMyRegistration returns registration status for an event
// @Summary Get My Registration
// @Description Get current user's registration and payment status for an event
// @Tags         Archer
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Event Slug or UUID"
// @Success 200 {object} MobileMyRegistrationResponse
// @Router       /archer/tournaments/{id}/registration [get]
func MobileGetMyRegistration(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		userID, _ := c.Get("user_id")

		var archerUUID string
		if err := db.Get(&archerUUID, `SELECT uuid FROM archers WHERE uuid = ?`, fmt.Sprintf("%v", userID)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Atlet tidak ditemukan"})
			return
		}

		var eventUUID string
		if err := db.Get(&eventUUID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		var registrations []MobileRegistrationItem
		err := db.Select(&registrations, `
			SELECT
				ep.uuid,
				COALESCE(ec.category_name_custom, CONCAT(COALESCE(rbt.name,''), ' ', COALESCE(rag.name,''), ' ', COALESCE(rgd.name,''))) as category_name,
				ep.payment_status,
				COALESCE(ep.payment_amount, 0) as payment_amount,
				ep.qr_raw,
				pt.payment_method,
				pt.gateway_reference,
				pt.checkout_url,
				pt.instructions,
				pt.va_number,
				pt.pay_code,
				pt.qr_url,
				ep.registration_date
			FROM tournament_participants ep
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN payment_transactions pt ON pt.registration_id = ep.uuid AND pt.status = 'pending'
			WHERE ep.tournament_id = ? AND ep.archer_id = ? AND ep.payment_status != 'cancelled'
			ORDER BY ep.registration_date DESC
		`, eventUUID, archerUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data pendaftaran"})
			return
		}
		if registrations == nil {
			registrations = []MobileRegistrationItem{}
		}
		for i := range registrations {
			if registrations[i].QRRaw == nil || *registrations[i].QRRaw == "" {
				qrFallback := fmt.Sprintf("AH-%s", registrations[i].ID)
				registrations[i].QRRaw = &qrFallback
			}
			registrations[i].QRCodeDataURL = buildMobileQRCodeDataURL(registrations[i].QRRaw)
		}
		c.JSON(http.StatusOK, MobileMyRegistrationResponse{
			EventID:       eventUUID,
			Registrations: registrations,
		})
	}
}

// MobileGetMyEvents returns list of registered tournaments
// @Summary Get My Events
// @Description Get list of all tournaments the archer is registered in
// @Tags         Archer
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MobileMyEventsResponse
// @Router       /archer/tournaments [get]
func MobileGetMyEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var archerUUID string
		if err := db.Get(&archerUUID, `SELECT uuid FROM archers WHERE uuid = ?`, fmt.Sprintf("%v", userID)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Atlet tidak ditemukan"})
			return
		}

		var tournaments []MobileMyEventItem
		err := db.Select(&tournaments, `
			SELECT
				ep.uuid as registration_id,
				e.uuid as event_uuid, e.name as event_name, e.slug as event_slug,
				COALESCE(NULLIF(e.location, ''), NULLIF(e.venue, ''), NULLIF(e.city, ''), 'Lokasi Belum Diatur') as location,
				e.start_date, e.end_date, e.logo_url, e.banner_url,
				ep.qr_raw,
				COALESCE(ec.category_name_custom, CONCAT(COALESCE(rbt.name,''), ' ', COALESCE(rag.name,''), ' ', COALESCE(rgd.name,''))) as category_name,
				ep.payment_status,
				pt.payment_method,
				COALESCE(pt.pay_code, pt.va_number) as pay_code,
				pt.va_number,
				COALESCE(pt.gateway_reference, pt.reference) as gateway_reference,
				COALESCE(pt.total_amount, pt.amount, ep.payment_amount, 0) as payment_amount,
				ep.registration_date
			FROM tournament_participants ep
			JOIN tournaments e ON ep.tournament_id = e.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN payment_transactions pt ON (ep.payment_id = pt.uuid OR pt.registration_id = ep.uuid)
			WHERE ep.archer_id = ? AND ep.payment_status != 'cancelled'
			ORDER BY e.start_date DESC
		`, archerUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data event"})
			return
		}
		if tournaments == nil {
			tournaments = []MobileMyEventItem{}
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
			if tournaments[i].QRRaw == nil || *tournaments[i].QRRaw == "" {
				qrFallback := fmt.Sprintf("AH-%s", tournaments[i].RegistrationID)
				tournaments[i].QRRaw = &qrFallback
			}
			tournaments[i].QRCodeDataURL = buildMobileQRCodeDataURL(tournaments[i].QRRaw)
		}

		c.JSON(http.StatusOK, MobileMyEventsResponse{
			Events: tournaments,
			Total:  len(tournaments),
		})
	}
}

// MobileGetEventQRCode returns registration QR code
// @Summary Get Event QR Code
// @Description Get the digital ID / QR code for event check-in
// @Tags         Archer
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Event Slug or UUID"
// @Success 200 {object} map[string]interface{}
// @Router       /archer/tournaments/{id}/qr [get]
func MobileGetEventQRCode(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		userID, _ := c.Get("user_id")

		var archerUUID string
		if err := db.Get(&archerUUID, `SELECT uuid FROM archers WHERE uuid = ? OR id = ? LIMIT 1`, fmt.Sprintf("%v", userID), fmt.Sprintf("%v", userID)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Atlet tidak ditemukan"})
			return
		}

		var eventUUID string
		if err := db.Get(&eventUUID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		var result struct {
			RegistrationID   string   `db:"registration_id" json:"registration_id"`
			QRRaw            *string  `db:"qr_raw" json:"qr_raw"`
			PaymentStatus    string   `db:"payment_status" json:"payment_status"`
			PaymentMethod    *string  `db:"payment_method" json:"payment_method"`
			PayCode          *string  `db:"pay_code" json:"pay_code"`
			VANumber         *string  `db:"va_number" json:"va_number"`
			GatewayReference *string  `db:"gateway_reference" json:"gateway_reference"`
			PaymentAmount    *float64 `db:"payment_amount" json:"payment_amount"`
			CategoryName     *string  `db:"category_name" json:"category_name"`
			EventName        *string  `db:"event_name" json:"event_name"`
			RegistrationDate string   `db:"registration_date" json:"registration_date"`
		}

		err := db.Get(&result, `
			SELECT 
				ep.uuid as registration_id, 
				ep.qr_raw, 
				ep.payment_status,
				pt.payment_method,
				COALESCE(pt.pay_code, pt.va_number) as pay_code,
				pt.va_number,
				COALESCE(pt.gateway_reference, pt.reference) as gateway_reference,
				COALESCE(pt.total_amount, pt.amount, ep.payment_amount, 0) as payment_amount,
				COALESCE(ec.category_name_custom, CONCAT(COALESCE(rbt.name,''), ' ', COALESCE(rag.name,''), ' ', COALESCE(rgd.name,''))) as category_name,
				e.name as event_name,
				ep.registration_date
			FROM tournament_participants ep
			JOIN tournaments e ON ep.tournament_id = e.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN payment_transactions pt ON (ep.payment_id = pt.uuid OR pt.registration_id = ep.uuid)
			WHERE (ep.tournament_id = ? OR e.slug = ?) AND ep.archer_id = ?
			ORDER BY ep.qr_raw IS NULL ASC, ep.registration_date DESC
			LIMIT 1
		`, eventUUID, eventUUID, archerUUID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Registrasi event tidak ditemukan"})
			return
		}

		qrVal := ""
		if result.QRRaw != nil && *result.QRRaw != "" {
			qrVal = *result.QRRaw
		} else {
			qrVal = fmt.Sprintf("AH-%s", result.RegistrationID)
		}

		c.JSON(http.StatusOK, gin.H{
			"event_id":          eventUUID,
			"event_name":        result.EventName,
			"registration_id":   result.RegistrationID,
			"qr_raw":            qrVal,
			"qr_code_data_url":  buildMobileQRCodeDataURL(&qrVal),
			"payment_status":    result.PaymentStatus,
			"payment_method":    result.PaymentMethod,
			"pay_code":          result.PayCode,
			"va_number":         result.VANumber,
			"gateway_reference": result.GatewayReference,
			"payment_amount":    result.PaymentAmount,
			"category_name":     result.CategoryName,
			"registration_date": result.RegistrationDate,
		})
	}
}

// MobileArcherGetEventPerformance returns performance summary for an archer in an event
func MobileArcherGetEventPerformance(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		userID, _ := c.Get("user_id")

		var archerUUID string
		if err := db.Get(&archerUUID, `SELECT uuid FROM archers WHERE uuid = ? OR id = ? OR email = ? LIMIT 1`, fmt.Sprintf("%v", userID), fmt.Sprintf("%v", userID), fmt.Sprintf("%v", userID)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Atlet tidak ditemukan"})
			return
		}

		var event struct {
			UUID      string  `db:"uuid"`
			Name      string  `db:"name"`
			StartDate string  `db:"start_date"`
			EndDate   string  `db:"end_date"`
			Location  string  `db:"location"`
			LogoURL   *string `db:"logo_url"`
		}
		err := db.Get(&event, "SELECT uuid, name, start_date, end_date, location, logo_url FROM tournaments WHERE uuid = ? OR slug = ?", eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		type RegRow struct {
			RegistrationID   string  `db:"registration_id"`
			CategoryID       string  `db:"category_id"`
			CategoryName     string  `db:"category_name"`
			PaymentStatus    string  `db:"payment_status"`
			RegistrationDate string  `db:"registration_date"`
			TargetNumber     *string `db:"target_number"`
		}

		var regRows []RegRow
		err = db.Select(&regRows, `
			SELECT 
				ep.uuid as registration_id,
				COALESCE(ep.category_id, '') as category_id,
				COALESCE(ec.category_name_custom, CONCAT(COALESCE(rbt.name,''), ' ', COALESCE(rag.name,''), ' ', COALESCE(rgd.name,''))) as category_name,
				ep.payment_status,
				COALESCE(ep.registration_date, '') as registration_date,
				MAX(COALESCE(tt.target_name, '')) as target_number
			FROM tournament_participants ep
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN qualification_target_assignments qta ON qta.participant_uuid = ep.uuid
			LEFT JOIN tournament_targets tt ON qta.target_uuid = tt.uuid
			WHERE ep.tournament_id = ? AND ep.archer_id = ? AND ep.payment_status != 'cancelled'
			GROUP BY ep.uuid, ep.category_id, ec.category_name_custom, rbt.name, rag.name, rgd.name, ep.payment_status, ep.registration_date
			ORDER BY ep.registration_date ASC
		`, event.UUID, archerUUID)

		logo := event.LogoURL
		if logo != nil {
			masked := utils.MaskMediaURL(*logo)
			logo = &masked
		}

		if err != nil || len(regRows) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"event_id":          event.UUID,
				"event_name":        event.Name,
				"start_date":        event.StartDate,
				"end_date":          event.EndDate,
				"location":          event.Location,
				"logo_url":          logo,
				"categories":        []gin.H{},
				"registration_id":   "",
				"category_name":     "Belum Terdaftar",
				"payment_status":    "Belum Terdaftar",
				"registration_date": "-",
				"target_number":     "Belum Diatur",
				"total_score":       0,
				"rank":              0,
				"elimination_stage": "Belum Masuk",
				"elimination_win":   "-",
			})
			return
		}

		type SessionBreakdownItem struct {
			SessionUUID string `json:"session_uuid"`
			SessionName string `json:"session_name"`
			Score       int    `json:"score"`
			MaxScore    int    `json:"max_score"`
			TenCount    int    `json:"ten_count"`
			XCount      int    `json:"x_count"`
		}

		type EndScoreItem struct {
			EndNumber int `json:"end_number"`
			Score     int `json:"score"`
			TenCount  int `json:"ten_count"`
			XCount    int `json:"x_count"`
		}

		type CategoryPerformanceItem struct {
			RegistrationID   string                 `json:"registration_id"`
			CategoryID       string                 `json:"category_id"`
			CategoryName     string                 `json:"category_name"`
			PaymentStatus    string                 `json:"payment_status"`
			RegistrationDate string                 `json:"registration_date"`
			TargetNumber     string                 `json:"target_number"`
			TotalScore       int                    `json:"total_score"`
			TotalTen         int                    `json:"total_ten"`
			TotalX           int                    `json:"total_x"`
			ArrowAverage     float64                `json:"arrow_average"`
			Rank             int                    `json:"rank"`
			EliminationStage string                 `json:"elimination_stage"`
			EliminationWin   string                 `json:"elimination_win"`
			Sessions         []SessionBreakdownItem `json:"sessions"`
			Ends             []EndScoreItem         `json:"ends"`
		}

		var catList []CategoryPerformanceItem

		for _, reg := range regRows {
			// Fetch real qualification ends & sessions
			type dbEndRow struct {
				SessionUUID   string `db:"session_uuid"`
				SessionName   string `db:"session_name"`
				EndNumber     int    `db:"end_number"`
				TotalScoreEnd int    `db:"total_score_end"`
				TenCountEnd   int    `db:"ten_count_end"`
				XCountEnd     int    `db:"x_count_end"`
			}
			var dbEnds []dbEndRow
			_ = db.Select(&dbEnds, `
				SELECT 
					qes.session_uuid,
					COALESCE(qs.name, 'Sesi Kualifikasi') as session_name,
					qes.end_number,
					qes.total_score_end,
					qes.ten_count_end,
					qes.x_count_end
				FROM qualification_end_scores qes
				LEFT JOIN qualification_sessions qs ON qes.session_uuid = qs.uuid
				WHERE qes.participant_uuid = ?
				ORDER BY qes.session_uuid ASC, qes.end_number ASC
			`, reg.RegistrationID)

			totalScore := 0
			totalTen := 0
			totalX := 0
			var endItems []EndScoreItem
			sessionMap := make(map[string]*SessionBreakdownItem)
			var sessionOrder []string

			for _, e := range dbEnds {
				totalScore += e.TotalScoreEnd
				totalTen += e.TenCountEnd
				totalX += e.XCountEnd

				endItems = append(endItems, EndScoreItem{
					EndNumber: len(endItems) + 1,
					Score:     e.TotalScoreEnd,
					TenCount:  e.TenCountEnd,
					XCount:    e.XCountEnd,
				})

				if sessionMap[e.SessionUUID] == nil {
					sessionMap[e.SessionUUID] = &SessionBreakdownItem{
						SessionUUID: e.SessionUUID,
						SessionName: e.SessionName,
						MaxScore:    360,
					}
					sessionOrder = append(sessionOrder, e.SessionUUID)
				}
				sessionMap[e.SessionUUID].Score += e.TotalScoreEnd
				sessionMap[e.SessionUUID].TenCount += e.TenCountEnd
				sessionMap[e.SessionUUID].XCount += e.XCountEnd
			}

			var sessionItems []SessionBreakdownItem
			for _, sUUID := range sessionOrder {
				if s, ok := sessionMap[sUUID]; ok {
					sessionItems = append(sessionItems, *s)
				}
			}

			arrowAvg := 0.0
			if len(dbEnds) > 0 {
				totalArrows := len(dbEnds) * 6
				arrowAvg = float64(totalScore) / float64(totalArrows)
			}

			// Fetch Rank in category
			var rank int = 0
			if totalScore > 0 {
				_ = db.Get(&rank, `
					SELECT COUNT(distinct s.participant_uuid) + 1
					FROM (
						SELECT participant_uuid, COALESCE(SUM(total_score_end), 0) as total
						FROM qualification_end_scores
						GROUP BY participant_uuid
					) s
					JOIN tournament_participants ep2 ON ep2.uuid = s.participant_uuid
					WHERE ep2.tournament_id = ? AND ep2.category_id = ? AND s.total > ?
				`, event.UUID, reg.CategoryID, totalScore)
			}

			// Fetch Elimination journey
			type elimMatchRow struct {
				RoundNo         int     `db:"round_no"`
				MatchNo         int     `db:"match_no"`
				WinnerEntryUUID *string `db:"winner_entry_uuid"`
				Status          string  `db:"status"`
				EntryUUID       string  `db:"entry_uuid"`
				Seed            int     `db:"seed"`
			}
			var elimMatches []elimMatchRow
			_ = db.Select(&elimMatches, `
				SELECT 
					em.round_no,
					em.match_no,
					em.winner_entry_uuid,
					em.status,
					ee.uuid as entry_uuid,
					ee.seed
				FROM elimination_entries ee
				JOIN elimination_matches em ON (em.entry_a_uuid = ee.uuid OR em.entry_b_uuid = ee.uuid)
				WHERE ee.participant_uuid = ?
				ORDER BY em.round_no ASC
			`, reg.RegistrationID)

			targetNum := "Belum Diatur"
			if reg.TargetNumber != nil && *reg.TargetNumber != "" {
				targetNum = *reg.TargetNumber
			}

			stage := "Belum Masuk"
			win := "-"

			if len(elimMatches) > 0 {
				topMatch := elimMatches[0] // Lowest round_no (1 is Final, 2 is SF, etc.)
				isWinner := topMatch.WinnerEntryUUID != nil && *topMatch.WinnerEntryUUID == topMatch.EntryUUID

				if topMatch.RoundNo == 1 {
					if isWinner {
						stage = "Final Emas (Gold Medal)"
						win = "Juara 1 (Medali Emas)"
					} else {
						stage = "Final Emas (Gold Medal)"
						win = "Juara 2 (Medali Perak)"
					}
				} else if topMatch.RoundNo == 2 {
					if isWinner {
						stage = "Semifinal"
						win = "Lolos ke Final"
					} else {
						stage = "Semifinal"
						win = "Perebutan Juara 3"
					}
				} else if topMatch.RoundNo == 3 {
					stage = "Perempat Final"
					if isWinner {
						win = "Lolos ke Semifinal"
					} else {
						win = "Gugur Perempat Final"
					}
				} else if topMatch.RoundNo == 4 {
					stage = "Babak 1/8 Final"
					if isWinner {
						win = "Lolos ke Perempat Final"
					} else {
						win = "Gugur 1/8 Final"
					}
				} else if topMatch.RoundNo == 5 {
					stage = "Babak 1/16 Final"
					if isWinner {
						win = "Lolos ke 1/8 Final"
					} else {
						win = "Gugur 1/16 Final"
					}
				} else {
					stage = fmt.Sprintf("Babak %d", topMatch.RoundNo)
					if isWinner {
						win = "Menang"
					} else {
						win = "Kalah"
					}
				}
			} else {
				// Check if entered in elimination brackets but match not yet played
				var seed int
				errSeed := db.Get(&seed, `SELECT seed FROM elimination_entries WHERE participant_uuid = ? LIMIT 1`, reg.RegistrationID)
				if errSeed == nil && seed > 0 {
					stage = fmt.Sprintf("Lolos Eliminasi (Seed #%d)", seed)
					win = "Menunggu Pertandingan"
				}
			}

			catName := reg.CategoryName
			if catName == "" {
				catName = "Kategori Umum"
			}

			catList = append(catList, CategoryPerformanceItem{
				RegistrationID:   reg.RegistrationID,
				CategoryID:       reg.CategoryID,
				CategoryName:     catName,
				PaymentStatus:    reg.PaymentStatus,
				RegistrationDate: reg.RegistrationDate,
				TargetNumber:     targetNum,
				TotalScore:       totalScore,
				TotalTen:         totalTen,
				TotalX:           totalX,
				ArrowAverage:     arrowAvg,
				Rank:             rank,
				EliminationStage: stage,
				EliminationWin:   win,
				Sessions:         sessionItems,
				Ends:             endItems,
			})
		}

		first := catList[0]

		c.JSON(http.StatusOK, gin.H{
			"event_id":          event.UUID,
			"event_name":        event.Name,
			"start_date":        event.StartDate,
			"end_date":          event.EndDate,
			"location":          event.Location,
			"logo_url":          logo,
			"categories":        catList,
			"registration_id":   first.RegistrationID,
			"category_name":     first.CategoryName,
			"payment_status":    first.PaymentStatus,
			"registration_date": first.RegistrationDate,
			"target_number":     first.TargetNumber,
			"total_score":       first.TotalScore,
			"total_ten":         first.TotalTen,
			"total_x":           first.TotalX,
			"arrow_average":     first.ArrowAverage,
			"rank":              first.Rank,
			"elimination_stage": first.EliminationStage,
			"elimination_win":   first.EliminationWin,
			"sessions":          first.Sessions,
			"ends":              first.Ends,
		})
	}
}

// MobileArcherGetCertificates returns certificates earned by the archer
func MobileArcherGetCertificates(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		type CertificateItem struct {
			ID            string `json:"id" db:"id"`
			Title         string `json:"title" db:"title"`
			EventName     string `json:"event_name" db:"event_name"`
			Category      string `json:"category" db:"category"`
			IssueDate     string `json:"issue_date" db:"issue_date"`
			PdfURL        string `json:"pdf_url" db:"pdf_url"`
			CertificateNo string `json:"certificate_no" db:"certificate_no"`
		}

		var certs []CertificateItem
		apiBase := utils.GetAPIBaseURL()
		err := db.Select(&certs, `
			SELECT 
				ep.uuid as id,
				CONCAT('Sertifikat Partisipasi - ', e.name) as title,
				e.name as event_name,
				COALESCE(cat.name, 'Kategori Umum') as category,
				DATE_FORMAT(ep.registration_date, '%d %b %Y') as issue_date,
				COALESCE(ep.certificate_url, CONCAT(?, '/api/certificates/', ep.uuid, '/pdf')) as pdf_url,
				CONCAT('CERT-', UPPER(SUBSTRING(ep.uuid, 1, 8))) as certificate_no
			FROM tournament_participants ep
			JOIN tournaments e ON ep.tournament_id = e.uuid
			LEFT JOIN tournament_categories cat ON ep.event_category_id = cat.uuid
			WHERE ep.archer_id = ? AND ep.payment_status IN ('paid', 'lunas', 'settlement', 'completed', 'confirmed')
			ORDER BY ep.registration_date DESC
		`, apiBase, userID)

		if err != nil || certs == nil {
			certs = []CertificateItem{}
		}

		c.JSON(http.StatusOK, certs)
	}
}

