package mobile

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type MobileBroadcastItem struct {
	UUID         string    `json:"uuid" db:"uuid"`
	EventID      string    `json:"event_id" db:"event_id"`
	OrganizerID  string    `json:"organizer_id" db:"organizer_id"`
	Title        string    `json:"title" db:"title"`
	Message      string    `json:"message" db:"message"`
	TargetType   string    `json:"target_type" db:"target_type"`
	TargetID     *string   `json:"target_id" db:"target_id"`
	TargetLabel  *string   `json:"target_label" db:"target_label"`
	SentCount    int       `json:"sent_count" db:"sent_count"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type CreateBroadcastRequest struct {
	Title       string  `json:"title" binding:"required"`
	Message     string  `json:"message" binding:"required"`
	TargetType  string  `json:"target_type" binding:"required"` // 'all', 'paid', 'unpaid', 'category'
	TargetID    *string `json:"target_id"`
	TargetLabel *string `json:"target_label"`
}

// MobileGetEventBroadcasts lists all broadcasts for a specific event
func MobileGetEventBroadcasts(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMobileUserType(c, "organizer") {
			return
		}
		organizationUUID, ok := getMobileOrganizationUUID(c, db)
		if !ok {
			return
		}

		eventID := c.Param("id")

		var broadcasts []MobileBroadcastItem
		query := `SELECT uuid, event_id, organizer_id, title, message, target_type, target_id, target_label, sent_count, created_at 
		          FROM broadcasts 
		          WHERE event_id = ? AND organizer_id = ? 
		          ORDER BY created_at DESC`
		err := db.Select(&broadcasts, query, eventID, organizationUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar broadcast", "details": err.Error()})
			return
		}

		if broadcasts == nil {
			broadcasts = []MobileBroadcastItem{}
		}

		c.JSON(http.StatusOK, broadcasts)
	}
}

// MobileGetBroadcastDetail returns detail of a specific broadcast
func MobileGetBroadcastDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMobileUserType(c, "organizer") {
			return
		}
		organizationUUID, ok := getMobileOrganizationUUID(c, db)
		if !ok {
			return
		}

		eventID := c.Param("id")
		broadcastID := c.Param("broadcast_id")

		var broadcast MobileBroadcastItem
		query := `SELECT uuid, event_id, organizer_id, title, message, target_type, target_id, target_label, sent_count, created_at 
		          FROM broadcasts 
		          WHERE uuid = ? AND event_id = ? AND organizer_id = ? 
		          LIMIT 1`
		err := db.Get(&broadcast, query, broadcastID, eventID, organizationUUID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Detail broadcast tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, broadcast)
	}
}

// MobileCreateBroadcast creates and sends a new broadcast message
func MobileCreateBroadcast(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMobileUserType(c, "organizer") {
			return
		}
		organizationUUID, ok := getMobileOrganizationUUID(c, db)
		if !ok {
			return
		}

		eventID := c.Param("id")

		var req CreateBroadcastRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data input tidak valid", "details": err.Error()})
			return
		}

		// Fetch target archer IDs
		var archerIDs []string
		if req.TargetType == "category" && req.TargetID != nil {
			query := `SELECT DISTINCT archer_id FROM event_participants WHERE event_id = ? AND category_id = ? AND archer_id IS NOT NULL`
			_ = db.Select(&archerIDs, query, eventID, *req.TargetID)
		} else if req.TargetType == "paid" {
			query := `SELECT DISTINCT archer_id FROM event_participants WHERE event_id = ? AND (payment_status = 'settlement' OR payment_status = 'paid' OR payment_status = 'Lunas') AND archer_id IS NOT NULL`
			_ = db.Select(&archerIDs, query, eventID)
		} else if req.TargetType == "unpaid" {
			query := `SELECT DISTINCT archer_id FROM event_participants WHERE event_id = ? AND (payment_status IS NULL OR payment_status = 'unpaid' OR payment_status = 'Menunggu') AND archer_id IS NOT NULL`
			_ = db.Select(&archerIDs, query, eventID)
		} else {
			query := `SELECT DISTINCT archer_id FROM event_participants WHERE event_id = ? AND archer_id IS NOT NULL`
			_ = db.Select(&archerIDs, query, eventID)
		}

		sentCount := len(archerIDs)
		broadcastUUID := uuid.New().String()

		_, err := db.Exec(`
			INSERT INTO broadcasts (uuid, event_id, organizer_id, title, message, target_type, target_id, target_label, sent_count, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
		`, broadcastUUID, eventID, organizationUUID, req.Title, req.Message, req.TargetType, req.TargetID, req.TargetLabel, sentCount)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan broadcast", "details": err.Error()})
			return
		}

		// Dispatch notifications to target archers
		for _, archerID := range archerIDs {
			_, _ = db.Exec(`
				INSERT INTO notifications (user_id, user_role, type, title, message, link, is_read, created_at)
				VALUES (?, 'archer', 'info', ?, ?, '', 0, NOW())
			`, archerID, req.Title, req.Message)
		}

		// Return the saved broadcast item
		c.JSON(http.StatusCreated, gin.H{
			"uuid":          broadcastUUID,
			"event_id":      eventID,
			"organizer_id":  organizationUUID,
			"title":         req.Title,
			"message":       req.Message,
			"target_type":   req.TargetType,
			"target_id":     req.TargetID,
			"target_label":  req.TargetLabel,
			"sent_count":    sentCount,
			"created_at":    time.Now().Format(time.RFC3339),
		})
	}
}
