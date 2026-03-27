package mobile

import (
	"archeryhub-api/handler"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// MobileStartOrGetConversation godoc
func MobileStartOrGetConversation(db *sqlx.DB) gin.HandlerFunc {
	return handler.StartOrGetConversation(db)
}

// MobileListConversations godoc
func MobileListConversations(db *sqlx.DB) gin.HandlerFunc {
	return handler.ListConversations(db)
}

// MobileGetConversationMessages godoc
func MobileGetConversationMessages(db *sqlx.DB) gin.HandlerFunc {
	return handler.GetConversationMessages(db)
}

// MobileSendMessage godoc
func MobileSendMessage(db *sqlx.DB) gin.HandlerFunc {
	return handler.SendMessage(db)
}

// MobileGetChatUnreadCount godoc
func MobileGetChatUnreadCount(db *sqlx.DB) gin.HandlerFunc {
	return handler.GetChatUnreadCount(db)
}

// MobileGetPeerLastActive godoc
func MobileGetPeerLastActive(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		peerType := c.Query("peer_type")
		peerID := c.Query("peer_id")
		if peerType == "" || peerID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "peer_type dan peer_id wajib diisi"})
			return
		}

		var lastSeen *time.Time
		switch peerType {
		case "seller":
			err := db.Get(&lastSeen, "SELECT last_seen_at FROM sellers WHERE uuid = ? LIMIT 1", peerID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Seller tidak ditemukan"})
				return
			}
		case "archer":
			err := db.Get(&lastSeen, "SELECT last_seen_at FROM archers WHERE uuid = ? LIMIT 1", peerID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Archer tidak ditemukan"})
				return
			}
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "peer_type harus seller atau archer"})
			return
		}

		isOnline := false
		if lastSeen != nil {
			isOnline = time.Since(*lastSeen) <= 2*time.Minute
		}

		c.JSON(http.StatusOK, gin.H{
			"peer_type":    peerType,
			"peer_id":      peerID,
			"last_seen_at": lastSeen,
			"is_online":    isOnline,
			"server_time":  fmt.Sprintf("%s", time.Now().UTC().Format(time.RFC3339)),
		})
	}
}
