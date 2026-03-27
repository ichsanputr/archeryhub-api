package mobile

import (
	"archeryhub-api/utils"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

// MobileRegisterForEvent godoc
// @Summary      Register for an event
// @Description  Allows an authenticated archer to self-register for multiple categories in an event. Returns registration ID and categories.
// @Tags         Events
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string  false  "Event UUID or slug (optional if in body)"
// @Param        request  body      object{event_id=string,event_category_ids=[]string,payment_type=string,payment_amount=float64}  true  "Registration payload"
// @Success      201      {object}  MobileRegisterEventResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Failure      409      {object}  ErrorResponse
// @Router       /archer/events/register [post]
func MobileRegisterForEvent(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		userID, _ := c.Get("user_id")

		var req struct {
			EventID            string   `json:"event_id"`
			AthleteID          string   `json:"athlete_id"`
			EventCategoryID    string   `json:"event_category_id"`
			EventCategoryIDs   []string `json:"event_category_ids"`
			PaymentAmount      float64  `json:"payment_amount"`
			PaymentProofURLs   []string `json:"payment_proof_urls"`
			PaymentStatus      string   `json:"payment_status"`
			RegistrationSource string   `json:"registration_source"`
			PaymentType        string   `json:"payment_type"` // manual, online, gateway
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if eventID == "" {
			eventID = req.EventID
		}

		if eventID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "event_id wajib diisi"})
			return
		}

		allCategoryIDs := []string{}
		if strings.TrimSpace(req.EventCategoryID) != "" {
			allCategoryIDs = append(allCategoryIDs, strings.TrimSpace(req.EventCategoryID))
		}
		for _, id := range req.EventCategoryIDs {
			trimmed := strings.TrimSpace(id)
			if trimmed == "" {
				continue
			}
			exists := false
			for _, existing := range allCategoryIDs {
				if existing == trimmed {
					exists = true
					break
				}
			}
			if !exists {
				allCategoryIDs = append(allCategoryIDs, trimmed)
			}
		}

		if len(allCategoryIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "event_category_ids wajib diisi minimal 1"})
			return
		}

		var archerUUID string
		lookupAthleteID := strings.TrimSpace(req.AthleteID)
		if lookupAthleteID == "" {
			lookupAthleteID = fmt.Sprintf("%v", userID)
		}
		if err := db.Get(&archerUUID, `SELECT uuid FROM archers WHERE uuid = ? OR id = ?`, lookupAthleteID, lookupAthleteID); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Akun atlet tidak ditemukan"})
			return
		}

		if archerUUID != fmt.Sprintf("%v", userID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak diizinkan mendaftarkan atlet lain"})
			return
		}

		var event struct {
			UUID        string `db:"uuid"`
			OrganizerID string `db:"organizer_id"`
		}
		if err := db.Get(&event, `SELECT uuid, organizer_id FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		var orgStatus string
		db.Get(&orgStatus, `
			SELECT COALESCE(s, 'trial') FROM (
				SELECT subscription_status as s FROM organizations WHERE uuid = ?
				UNION ALL
				SELECT 'trial' as s FROM clubs WHERE uuid = ?
			) combined LIMIT 1`, event.OrganizerID, event.OrganizerID)
		if orgStatus != "" && orgStatus != "active" && orgStatus != "trial" {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "Pendaftaran ditutup sementara oleh sistem", "code": "organizer_subscription_expired"})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		paymentStatus := "menunggu acc"
		if req.PaymentType == "gateway" || req.PaymentType == "online" {
			paymentStatus = "unpaid"
		}

		if req.PaymentStatus != "" {
			paymentStatus = req.PaymentStatus
		}

		registrationSource := "mobile_app"
		if strings.TrimSpace(req.RegistrationSource) != "" {
			registrationSource = strings.TrimSpace(req.RegistrationSource)
		}

		proofURLs := ""
		if len(req.PaymentProofURLs) > 0 {
			proofURLs = strings.Join(req.PaymentProofURLs, ",")
		}

		registrationDate := time.Now()

		var registeredCategories []string
		firstParticipantID := ""
		for _, catID := range allCategoryIDs {
			var existingID string
			if err := tx.Get(&existingID, `SELECT uuid FROM event_participants WHERE event_id = ? AND archer_id = ? AND category_id = ?`, event.UUID, archerUUID, catID); err == nil {
				continue
			}
			participantID := uuid.New().String()
			if firstParticipantID == "" {
				firstParticipantID = participantID
			}
			if _, err = tx.Exec(`
				INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, payment_amount, payment_proof_urls, registration_source, registration_date)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, participantID, event.UUID, archerUUID, catID, paymentStatus, req.PaymentAmount, proofURLs, registrationSource, registrationDate); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftar: " + err.Error()})
				return
			}
			registeredCategories = append(registeredCategories, catID)
		}

		if err = tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data"})
			return
		}

		if registeredCategories == nil {
			registeredCategories = []string{}
		}

		if len(registeredCategories) == 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Pemanah sudah terdaftar di semua kategori pilihan pada event ini"})
			return
		}

		c.JSON(http.StatusCreated, MobileRegisterEventResponse{
			Message:              "Pendaftaran berhasil",
			RegistrationID:       firstParticipantID,
			RegisteredCategories: registeredCategories,
			PaymentStatus:        paymentStatus,
		})
	}
}

// MobileGetMyRegistration godoc
// @Summary      Get archer's registration for an event
// @Description  Returns registration details including payment status and payment instructions
// @Tags         Archer
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Event UUID or slug"
// @Success      200 {object}  MobileMyRegistrationResponse
// @Failure      404 {object}  ErrorResponse
// @Router       /archer/events/{id}/registration [get]
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
		if err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID); err != nil {
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
				pt.tripay_reference,
				pt.checkout_url,
				pt.instructions,
				pt.va_number,
				pt.pay_code,
				pt.qr_url,
				ep.registration_date
			FROM event_participants ep
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN payment_transactions pt ON pt.registration_id = ep.uuid AND pt.status = 'pending'
			WHERE ep.event_id = ? AND ep.archer_id = ?
			ORDER BY ep.registration_date DESC
		`, eventUUID, archerUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data pendaftaran"})
			return
		}
		if registrations == nil {
			registrations = []MobileRegistrationItem{}
		}
		c.JSON(http.StatusOK, MobileMyRegistrationResponse{
			EventID:       eventUUID,
			Registrations: registrations,
		})
	}
}

// MobileGetMyEvents godoc
// @Summary      List archer's registered events
// @Description  Returns all events the authenticated archer has registered for
// @Tags         Archer
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  MobileMyEventsResponse
// @Router       /archer/events [get]
func MobileGetMyEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var archerUUID string
		if err := db.Get(&archerUUID, `SELECT uuid FROM archers WHERE uuid = ?`, fmt.Sprintf("%v", userID)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Atlet tidak ditemukan"})
			return
		}

		var events []MobileMyEventItem
		err := db.Select(&events, `
			SELECT
				e.uuid as event_uuid, e.name as event_name, e.slug as event_slug,
				e.location, e.start_date, e.end_date, e.logo_url,
				ep.qr_raw,
				COALESCE(ec.category_name_custom, CONCAT(COALESCE(rbt.name,''), ' ', COALESCE(rag.name,''), ' ', COALESCE(rgd.name,''))) as category_name,
				ep.payment_status,
				ep.registration_date
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			WHERE ep.archer_id = ?
			ORDER BY e.start_date DESC
		`, archerUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data event"})
			return
		}
		if events == nil {
			events = []MobileMyEventItem{}
		}

		for i := range events {
			if events[i].LogoURL != nil {
				masked := utils.MaskMediaURL(*events[i].LogoURL)
				events[i].LogoURL = &masked
			}
		}

		c.JSON(http.StatusOK, MobileMyEventsResponse{
			Events: events,
			Total:  len(events),
		})
	}
}

// MobileGetEventQRCode godoc
// @Summary      Get archer event QR
// @Description  Returns archer QR information and payment status for a specific event
// @Tags         Archer
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Event UUID or slug"
// @Success      200 {object} MobileEventQRCodeResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /archer/events/{id}/qr [get]
func MobileGetEventQRCode(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		userID, _ := c.Get("user_id")

		var archerUUID string
		if err := db.Get(&archerUUID, `SELECT uuid FROM archers WHERE uuid = ?`, fmt.Sprintf("%v", userID)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Atlet tidak ditemukan"})
			return
		}

		var eventUUID string
		if err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		var result struct {
			QRRaw            *string `db:"qr_raw" json:"qr_raw"`
			PaymentStatus    string  `db:"payment_status" json:"payment_status"`
			RegistrationDate string  `db:"registration_date" json:"registration_date"`
		}

		err := db.Get(&result, `
			SELECT ep.qr_raw, ep.payment_status, ep.registration_date
			FROM event_participants ep
			WHERE ep.event_id = ? AND ep.archer_id = ?
			ORDER BY ep.qr_raw IS NULL ASC, ep.registration_date DESC
			LIMIT 1
		`, eventUUID, archerUUID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Registrasi event tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"event_id":          eventUUID,
			"qr_raw":            result.QRRaw,
			"qr_code_data_url":  buildMobileQRCodeDataURL(result.QRRaw),
			"payment_status":    result.PaymentStatus,
			"registration_date": result.RegistrationDate,
		})
	}
}
