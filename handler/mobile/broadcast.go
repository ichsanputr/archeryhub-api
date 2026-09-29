package mobile

import (
	"Archeris-api/utils"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type MobileBroadcastItem struct {
	UUID         string    `json:"uuid" db:"uuid"`
	TournamentID string    `json:"tournament_id" db:"tournament_id"`
	EventID      string    `json:"event_id" db:"-"`
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
	Title       string   `json:"title" binding:"required"`
	Message     string   `json:"message" binding:"required"`
	TargetType  string   `json:"target_type"` // 'all', 'paid', 'unpaid', 'pending', 'category', 'recurve', 'compound', 'barebow'
	TargetID    *string  `json:"target_id"`
	TargetLabel *string  `json:"target_label"`
	Priority    string   `json:"priority"` // 'urgent', 'info'
	Channels    []string `json:"channels"` // ['Push Notification', 'Email']
}

type RecipientArcher struct {
	ArcherID string  `db:"archer_id"`
	FullName string  `db:"full_name"`
	Email    *string `db:"email"`
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
		var actualTournamentUUID string
		_ = db.Get(&actualTournamentUUID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID)
		if actualTournamentUUID == "" {
			actualTournamentUUID = eventID
		}

		var broadcasts []MobileBroadcastItem
		query := `SELECT uuid, tournament_id, organizer_id, title, message, target_type, target_id, target_label, sent_count, created_at 
		          FROM broadcasts 
		          WHERE tournament_id = ? AND organizer_id = ? 
		          ORDER BY created_at DESC`
		err := db.Select(&broadcasts, query, actualTournamentUUID, organizationUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar broadcast", "details": err.Error()})
			return
		}

		if broadcasts == nil {
			broadcasts = []MobileBroadcastItem{}
		}
		for i := range broadcasts {
			broadcasts[i].EventID = broadcasts[i].TournamentID
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

		var actualTournamentUUID string
		_ = db.Get(&actualTournamentUUID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID)
		if actualTournamentUUID == "" {
			actualTournamentUUID = eventID
		}

		var broadcast MobileBroadcastItem
		query := `SELECT uuid, tournament_id, organizer_id, title, message, target_type, target_id, target_label, sent_count, created_at 
		          FROM broadcasts 
		          WHERE uuid = ? AND tournament_id = ? AND organizer_id = ? 
		          LIMIT 1`
		err := db.Get(&broadcast, query, broadcastID, actualTournamentUUID, organizationUUID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Detail broadcast tidak ditemukan"})
			return
		}
		broadcast.EventID = broadcast.TournamentID

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
		var actualTournamentUUID string
		var tournamentName string
		var tournamentSlug string

		var tour struct {
			UUID string `db:"uuid"`
			Name string `db:"name"`
			Slug string `db:"slug"`
		}
		err := db.Get(&tour, "SELECT uuid, name, slug FROM tournaments WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID)
		if err == nil && tour.UUID != "" {
			actualTournamentUUID = tour.UUID
			tournamentName = tour.Name
			tournamentSlug = tour.Slug
		} else {
			actualTournamentUUID = eventID
			tournamentName = "Turnamen Panahan"
		}

		var req CreateBroadcastRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data input tidak valid", "details": err.Error()})
			return
		}

		targetType := strings.ToLower(strings.TrimSpace(req.TargetType))
		if targetType == "" {
			targetType = "all"
		}

		// Fetch target archers with their ID, name, and email
		var recipients []RecipientArcher
		baseQuery := `
			SELECT DISTINCT 
				tp.archer_id, 
				COALESCE(a.full_name, 'Archer') as full_name, 
				a.email
			FROM tournament_participants tp
			LEFT JOIN archers a ON (a.id = tp.archer_id OR a.uuid = tp.archer_id)
			LEFT JOIN tournament_categories tc ON tc.uuid = tp.category_id
			WHERE tp.tournament_id = ? AND tp.archer_id IS NOT NULL AND tp.archer_id != ''
		`

		var queryArgs []interface{}
		queryArgs = append(queryArgs, actualTournamentUUID)

		switch targetType {
		case "paid":
			baseQuery += " AND (tp.payment_status = 'settlement' OR tp.payment_status = 'paid' OR tp.payment_status = 'Lunas')"
		case "unpaid", "pending":
			baseQuery += " AND (tp.payment_status = 'pending' OR tp.payment_status = 'Menunggu' OR tp.payment_status = 'unpaid')"
		case "category":
			if req.TargetID != nil && *req.TargetID != "" {
				baseQuery += " AND (tp.category_id = ? OR tc.uuid = ?)"
				queryArgs = append(queryArgs, *req.TargetID, *req.TargetID)
			}
		case "recurve", "compound", "barebow":
			baseQuery += " AND (LOWER(COALESCE(tc.category_name_custom, '')) LIKE ? OR LOWER(COALESCE(tp.category_id, '')) LIKE ?)"
			likePattern := "%" + targetType + "%"
			queryArgs = append(queryArgs, likePattern, likePattern)
		default:
			// "all" - no additional where clause
		}

		err = db.Select(&recipients, baseQuery, queryArgs...)
		if err != nil {
			recipients = []RecipientArcher{}
		}

		sentCount := len(recipients)
		broadcastUUID := uuid.New().String()

		_, err = db.Exec(`
			INSERT INTO broadcasts (uuid, tournament_id, organizer_id, title, message, target_type, target_id, target_label, sent_count, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
		`, broadcastUUID, actualTournamentUUID, organizationUUID, req.Title, req.Message, targetType, req.TargetID, req.TargetLabel, sentCount)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan broadcast", "details": err.Error()})
			return
		}

		// Multi-row batch insert for In-App Notifications
		if sentCount > 0 {
			notifType := "info"
			if strings.ToLower(req.Priority) == "urgent" {
				notifType = "warning"
			}
			notifLink := "/tournaments/" + actualTournamentUUID
			if tournamentSlug != "" {
				notifLink = "/tournaments/" + tournamentSlug
			}

			// Batch insert in chunks of 100 to stay well under query parameter limits
			chunkSize := 100
			for i := 0; i < len(recipients); i += chunkSize {
				end := i + chunkSize
				if end > len(recipients) {
					end = len(recipients)
				}
				chunk := recipients[i:end]

				valueStrings := make([]string, 0, len(chunk))
				valueArgs := make([]interface{}, 0, len(chunk)*5)
				for _, r := range chunk {
					valueStrings = append(valueStrings, "(?, 'archer', ?, ?, ?, ?, 0, NOW())")
					valueArgs = append(valueArgs, r.ArcherID, notifType, req.Title, req.Message, notifLink)
				}

				stmt := fmt.Sprintf("INSERT INTO notifications (user_id, user_role, type, title, message, link, is_read, created_at) VALUES %s", strings.Join(valueStrings, ","))
				_, _ = db.Exec(stmt, valueArgs...)
			}
		}

		// Optional: Dispatch email channel in background goroutine if requested
		shouldSendEmail := false
		for _, ch := range req.Channels {
			if strings.EqualFold(strings.TrimSpace(ch), "email") {
				shouldSendEmail = true
				break
			}
		}

		if shouldSendEmail && sentCount > 0 {
			go func(recs []RecipientArcher, title, msg, tName string) {
				for _, r := range recs {
					if r.Email != nil && *r.Email != "" && strings.Contains(*r.Email, "@") {
						emailContent := fmt.Sprintf(`
							<p style="font-size:15px;color:#111827;font-weight:600;margin-bottom:8px;">Halo %s,</p>
							<p style="font-size:14px;color:#374151;line-height:1.6;white-space:pre-line;margin-bottom:16px;">%s</p>
							<p style="font-size:12px;color:#6b7280;margin-top:20px;border-top:1px solid #e5e7eb;padding-top:12px;">Pengumuman resmi dari panitia turnamen: <strong>%s</strong></p>
						`, r.FullName, msg, tName)
						body := utils.BuildCleanCardEmail(title, emailContent)
						_ = utils.SendEmail(*r.Email, fmt.Sprintf("[%s] %s", tName, title), body)
					}
				}
			}(recipients, req.Title, req.Message, tournamentName)
		}

		// Return saved broadcast item
		c.JSON(http.StatusCreated, gin.H{
			"uuid":          broadcastUUID,
			"event_id":      actualTournamentUUID,
			"tournament_id": actualTournamentUUID,
			"organizer_id":  organizationUUID,
			"title":         req.Title,
			"message":       req.Message,
			"target_type":   targetType,
			"target_id":     req.TargetID,
			"target_label":  req.TargetLabel,
			"sent_count":    sentCount,
			"created_at":    time.Now().Format(time.RFC3339),
		})
	}
}
