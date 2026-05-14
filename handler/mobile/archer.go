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

// MobileGetMyEvents returns list of registered events
// @Summary Get My Events
// @Description Get list of all events the archer is registered in
// @Tags         Archer
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MobileMyEventsResponse
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

// MobileGetEventQRCode returns registration QR code
// @Summary Get Event QR Code
// @Description Get the digital ID / QR code for event check-in
// @Tags         Archer
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Event Slug or UUID"
// @Success 200 {object} map[string]interface{}
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

