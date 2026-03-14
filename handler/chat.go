package handler

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// ─── Structs ─────────────────────────────────────────────────────────────────

type ChatConversation struct {
	UUID          string    `db:"uuid"            json:"id"`
	ArcherID      string    `db:"archer_id"       json:"archer_id"`
	SellerID      string    `db:"seller_id"       json:"seller_id"`
	ProductID     *string   `db:"product_id"      json:"product_id"`
	ProductName   *string   `db:"product_name"    json:"product_name"`
	ProductImage  *string   `db:"product_image"   json:"product_image"`
	LastMessage   *string   `db:"last_message"    json:"last_message"`
	LastMessageAt time.Time `db:"last_message_at" json:"last_message_at"`
	ArcherUnread  int       `db:"archer_unread"   json:"archer_unread"`
	SellerUnread  int       `db:"seller_unread"   json:"seller_unread"`
	CreatedAt     time.Time `db:"created_at"      json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"      json:"updated_at"`

	// Joined fields — use COALESCE in SQL so these are never NULL
	ArcherName     string     `db:"archer_name"      json:"archer_name"`
	ArcherAvatar   *string    `db:"archer_avatar"    json:"archer_avatar"`
	ArcherLastSeen *time.Time `db:"archer_last_seen" json:"archer_last_seen"`
	SellerName     string     `db:"seller_name"      json:"seller_name"`
	SellerAvatar   *string    `db:"seller_avatar"    json:"seller_avatar"`
	SellerLastSeen *time.Time `db:"seller_last_seen" json:"seller_last_seen"`
}

type ChatMessage struct {
	UUID           string    `db:"uuid"            json:"id"`
	ConversationID string    `db:"conversation_id" json:"conversation_id"`
	SenderType     string    `db:"sender_type"     json:"sender_type"`
	SenderID       string    `db:"sender_id"       json:"sender_id"`
	Message        string    `db:"message"         json:"message"`
	IsRead         bool      `db:"is_read"         json:"is_read"`
	CreatedAt      time.Time `db:"created_at"      json:"created_at"`

	// Joined fields
	SenderName   string  `db:"sender_name"   json:"sender_name"`
	SenderAvatar *string `db:"sender_avatar" json:"sender_avatar"`
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// resolveChatUser extracts user_id, role, and the specific profile uuid
// (archer.uuid / seller.uuid) from the auth context.
func resolveChatUser(c *gin.Context, db *sqlx.DB) (userID, role, profileUUID string, ok bool) {
	userID = c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
		return
	}

	// Safely read role — could be in "role" or fall back to "user_type"
	role = c.GetString("role")
	if role == "" {
		role = c.GetString("user_type")
	}

	switch role {
	case "archer":
		err := db.Get(&profileUUID, "SELECT uuid FROM archers WHERE uuid = ? OR google_id = ? LIMIT 1", userID, userID)
		if err != nil {
			profileUUID = userID
		}
		go db.Exec("UPDATE archers SET last_seen_at = NOW() WHERE uuid = ?", profileUUID)
	case "seller":
		err := db.Get(&profileUUID, "SELECT uuid FROM sellers WHERE uuid = ? OR user_id = ? LIMIT 1", userID, userID)
		if err != nil {
			profileUUID = userID
		}
		go db.Exec("UPDATE sellers SET last_seen_at = NOW() WHERE uuid = ?", profileUUID)
	default:
		profileUUID = userID
	}

	ok = true
	return
}

// convSelectSQL builds the shared SELECT for conversations.
// sellers table uses latin1_swedish_ci so we CONVERT its string columns
// to utf8mb4 before comparing/selecting, avoiding Error 1267 collation mismatch.
// TIMESTAMP columns (last_seen_at) are charset-neutral and need no CONVERT.
const convSelectSQL = `
	SELECT
		cc.uuid, cc.archer_id, cc.seller_id, cc.product_id, cc.product_name, cc.product_image,
		cc.last_message, cc.last_message_at, cc.archer_unread, cc.seller_unread,
		cc.created_at, cc.updated_at,
		COALESCE(a.full_name, '') AS archer_name,
		a.avatar_url AS archer_avatar,
		a.last_seen_at AS archer_last_seen,
		COALESCE(CONVERT(s.store_name USING utf8mb4), '') AS seller_name,
		CONVERT(s.avatar_url USING utf8mb4) AS seller_avatar,
		s.last_seen_at AS seller_last_seen
	FROM chat_conversations cc
	LEFT JOIN archers a ON cc.archer_id = a.uuid
	LEFT JOIN sellers s ON cc.seller_id = CONVERT(s.uuid USING utf8mb4) COLLATE utf8mb4_unicode_ci`

// ─── Handlers ────────────────────────────────────────────────────────────────

// StartOrGetConversation godoc
// POST /chat/conversations
// Archer calls this when they click "Chat Penjual" on a product page.
func StartOrGetConversation(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, role, profileUUID, ok := resolveChatUser(c, db)
		_ = userID
		if !ok {
			return
		}

		if role != "archer" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hanya pemanah yang dapat memulai percakapan"})
			return
		}

		var body struct {
			SellerID     string  `json:"seller_id"     binding:"required"`
			ProductID    *string `json:"product_id"`
			ProductName  *string `json:"product_name"`
			ProductImage *string `json:"product_image"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Input tidak valid: " + err.Error()})
			return
		}

		// Verify seller exists
		var sellerExists int
		db.Get(&sellerExists, "SELECT COUNT(*) FROM sellers WHERE uuid = ?", body.SellerID)
		if sellerExists == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Penjual tidak ditemukan"})
			return
		}

		// Normalize product_id: treat empty string as nil so NULL is stored correctly
		if body.ProductID != nil && *body.ProductID == "" {
			body.ProductID = nil
		}

		// Try to find existing conversation
		var existingUUID string
		var queryFind string
		var argsFind []interface{}

		if body.ProductID != nil && *body.ProductID != "" {
			queryFind = "SELECT uuid FROM chat_conversations WHERE archer_id = ? AND seller_id = ? AND product_id = ? LIMIT 1"
			argsFind = []interface{}{profileUUID, body.SellerID, *body.ProductID}
		} else {
			queryFind = "SELECT uuid FROM chat_conversations WHERE archer_id = ? AND seller_id = ? AND product_id IS NULL LIMIT 1"
			argsFind = []interface{}{profileUUID, body.SellerID}
		}

		err := db.Get(&existingUUID, queryFind, argsFind...)
		if err == nil {
			conv, fetchErr := fetchConversation(db, existingUUID)
			if fetchErr != nil {
				logrus.WithError(fetchErr).WithFields(logrus.Fields{
					"handler": "StartOrGetConversation", "conv_id": existingUUID,
				}).Error("[chat] fetchConversation existing failed")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat percakapan"})
				return
			}
			c.JSON(http.StatusOK, conv)
			return
		}

		// Create new conversation
		newUUID := uuid.New().String()
		_, err = db.Exec(`
			INSERT INTO chat_conversations
				(uuid, archer_id, seller_id, product_id, product_name, product_image)
			VALUES (?, ?, ?, ?, ?, ?)`,
			newUUID, profileUUID, body.SellerID, body.ProductID, body.ProductName, body.ProductImage,
		)
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"handler": "StartOrGetConversation", "archer": profileUUID, "seller": body.SellerID,
			}).Error("[chat] INSERT chat_conversations failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat percakapan: " + err.Error()})
			return
		}

		conv, fetchErr := fetchConversation(db, newUUID)
		if fetchErr != nil {
			logrus.WithError(fetchErr).WithFields(logrus.Fields{
				"handler": "StartOrGetConversation", "conv_id": newUUID,
			}).Error("[chat] fetchConversation after insert failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat percakapan baru"})
			return
		}
		c.JSON(http.StatusCreated, conv)
	}
}

// ListConversations returns all conversations for the current user (archer or seller).
// GET /chat/conversations
func ListConversations(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, role, profileUUID, ok := resolveChatUser(c, db)
		if !ok {
			return
		}

		var query string
		var args []interface{}

		switch role {
		case "archer":
			query = convSelectSQL + `
				WHERE cc.archer_id = ?
				ORDER BY cc.last_message_at DESC`
			args = []interface{}{profileUUID}
		case "seller":
			query = convSelectSQL + `
				WHERE cc.seller_id = ?
				ORDER BY cc.last_message_at DESC`
			args = []interface{}{profileUUID}
		default:
			c.JSON(http.StatusForbidden, gin.H{"error": "Role tidak didukung untuk fitur chat"})
			return
		}

		var convs []ChatConversation
		if err := db.Select(&convs, query, args...); err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"handler": "ListConversations", "role": role, "profile": profileUUID,
			}).Error("[chat] SELECT chat_conversations failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat daftar percakapan: " + err.Error()})
			return
		}

		if convs == nil {
			convs = []ChatConversation{}
		}

		unreadTotal := 0
		for _, cv := range convs {
			if role == "archer" {
				unreadTotal += cv.ArcherUnread
			} else {
				unreadTotal += cv.SellerUnread
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"conversations": convs,
			"unread_total":  unreadTotal,
		})
	}
}

// GetConversationMessages returns messages in a conversation.
// GET /chat/conversations/:id/messages
func GetConversationMessages(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, role, profileUUID, ok := resolveChatUser(c, db)
		if !ok {
			return
		}
		convID := c.Param("id")

		var convCheck struct {
			ArcherID string `db:"archer_id"`
			SellerID string `db:"seller_id"`
		}
		if err := db.Get(&convCheck, "SELECT archer_id, seller_id FROM chat_conversations WHERE uuid = ?", convID); err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Percakapan tidak ditemukan"})
			} else {
				logrus.WithError(err).WithField("conv_id", convID).Error("[chat] GetConversationMessages: verify conversation failed")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memverifikasi percakapan"})
			}
			return
		}

		switch role {
		case "archer":
			if convCheck.ArcherID != profileUUID {
				c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak"})
				return
			}
		case "seller":
			if convCheck.SellerID != profileUUID {
				c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak"})
				return
			}
		}

		// Mark messages from the other side as read
		if role == "archer" {
			db.Exec("UPDATE chat_messages SET is_read = 1 WHERE conversation_id = ? AND sender_type = 'seller'", convID)
			db.Exec("UPDATE chat_conversations SET archer_unread = 0 WHERE uuid = ?", convID)
		} else {
			db.Exec("UPDATE chat_messages SET is_read = 1 WHERE conversation_id = ? AND sender_type = 'archer'", convID)
			db.Exec("UPDATE chat_conversations SET seller_unread = 0 WHERE uuid = ?", convID)
		}

		messages := []ChatMessage{}
		err := db.Select(&messages, `
			SELECT
				cm.uuid, cm.conversation_id, cm.sender_type, cm.sender_id,
				cm.message, cm.is_read, cm.created_at,
				CASE cm.sender_type
					WHEN 'archer' THEN COALESCE(a.full_name, '')
					WHEN 'seller' THEN COALESCE(CONVERT(s.store_name USING utf8mb4), '')
					ELSE ''
				END AS sender_name,
				CASE cm.sender_type
					WHEN 'archer' THEN a.avatar_url
					WHEN 'seller' THEN CONVERT(s.avatar_url USING utf8mb4)
					ELSE NULL
				END AS sender_avatar
			FROM chat_messages cm
			LEFT JOIN archers a ON cm.sender_type = 'archer' AND cm.sender_id = a.uuid
			LEFT JOIN sellers s ON cm.sender_type = 'seller'
				AND cm.sender_id = CONVERT(s.uuid USING utf8mb4) COLLATE utf8mb4_unicode_ci
			WHERE cm.conversation_id = ?
			ORDER BY cm.created_at ASC`, convID)
		if err != nil {
			logrus.WithError(err).WithField("conv_id", convID).Error("[chat] GetConversationMessages: SELECT chat_messages failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat pesan"})
			return
		}

		conv, _ := fetchConversation(db, convID)
		c.JSON(http.StatusOK, gin.H{
			"conversation": conv,
			"messages":     messages,
		})
	}
}

// SendMessage sends a new message in a conversation.
// POST /chat/conversations/:id/messages
func SendMessage(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, role, profileUUID, ok := resolveChatUser(c, db)
		if !ok {
			return
		}
		convID := c.Param("id")

		var convCheck struct {
			ArcherID string `db:"archer_id"`
			SellerID string `db:"seller_id"`
		}
		if err := db.Get(&convCheck, "SELECT archer_id, seller_id FROM chat_conversations WHERE uuid = ?", convID); err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Percakapan tidak ditemukan"})
			} else {
				logrus.WithError(err).WithField("conv_id", convID).Error("[chat] SendMessage: verify conversation failed")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memverifikasi percakapan"})
			}
			return
		}

		switch role {
		case "archer":
			if convCheck.ArcherID != profileUUID {
				c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak"})
				return
			}
		case "seller":
			if convCheck.SellerID != profileUUID {
				c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak"})
				return
			}
		default:
			c.JSON(http.StatusForbidden, gin.H{"error": "Role tidak didukung"})
			return
		}

		var body struct {
			Message string `json:"message" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Pesan tidak boleh kosong"})
			return
		}

		msgUUID := uuid.New().String()
		now := time.Now()

		_, err := db.Exec(`
			INSERT INTO chat_messages (uuid, conversation_id, sender_type, sender_id, message)
			VALUES (?, ?, ?, ?, ?)`,
			msgUUID, convID, role, profileUUID, body.Message,
		)
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"handler": "SendMessage", "conv_id": convID, "sender": profileUUID,
			}).Error("[chat] INSERT chat_messages failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengirim pesan"})
			return
		}

		if role == "archer" {
			db.Exec(`UPDATE chat_conversations SET last_message = ?, last_message_at = ?, seller_unread = seller_unread + 1 WHERE uuid = ?`,
				body.Message, now, convID)
		} else {
			db.Exec(`UPDATE chat_conversations SET last_message = ?, last_message_at = ?, archer_unread = archer_unread + 1 WHERE uuid = ?`,
				body.Message, now, convID)
		}

		var msg ChatMessage
		db.Get(&msg, `
			SELECT
				cm.uuid, cm.conversation_id, cm.sender_type, cm.sender_id,
				cm.message, cm.is_read, cm.created_at,
				CASE cm.sender_type
					WHEN 'archer' THEN COALESCE(a.full_name, '')
					WHEN 'seller' THEN COALESCE(CONVERT(s.store_name USING utf8mb4), '')
					ELSE ''
				END AS sender_name,
				CASE cm.sender_type
					WHEN 'archer' THEN a.avatar_url
					WHEN 'seller' THEN CONVERT(s.avatar_url USING utf8mb4)
					ELSE NULL
				END AS sender_avatar
			FROM chat_messages cm
			LEFT JOIN archers a ON cm.sender_type = 'archer' AND cm.sender_id = a.uuid
			LEFT JOIN sellers s ON cm.sender_type = 'seller'
				AND cm.sender_id = CONVERT(s.uuid USING utf8mb4) COLLATE utf8mb4_unicode_ci
			WHERE cm.uuid = ?`, msgUUID)

		c.JSON(http.StatusCreated, msg)
	}
}

// GetChatUnreadCount returns total unread message count for the current user.
// GET /chat/unread
func GetChatUnreadCount(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, role, profileUUID, ok := resolveChatUser(c, db)
		if !ok {
			return
		}

		var count int
		switch role {
		case "archer":
			db.Get(&count, "SELECT COALESCE(SUM(archer_unread), 0) FROM chat_conversations WHERE archer_id = ?", profileUUID)
		case "seller":
			db.Get(&count, "SELECT COALESCE(SUM(seller_unread), 0) FROM chat_conversations WHERE seller_id = ?", profileUUID)
		}

		c.JSON(http.StatusOK, gin.H{"unread": count})
	}
}

// ─── Internal helper ─────────────────────────────────────────────────────────

func fetchConversation(db *sqlx.DB, convID string) (*ChatConversation, error) {
	var conv ChatConversation
	err := db.Get(&conv, convSelectSQL+` WHERE cc.uuid = ?`, convID)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}
