package mobile

import (
	"Archeris-api/handler"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// MobileStartOrGetConversation starts or gets a conversation
// @Summary Start/Get Conversation
// @Description Start a new chat or get existing conversation with a peer
// @Tags Mobile - Chat
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body MobileChatCreateConversationRequest true "Conversation Details"
// @Success 200 {object} ChatConversation
// @Router /mobile/chat/conversations [post]
func MobileStartOrGetConversation(db *sqlx.DB) gin.HandlerFunc {
	return handler.StartOrGetConversation(db)
}

// MobileListConversations returns all conversations for user
// @Summary List Conversations
// @Description Get list of all chat conversations for the authenticated user
// @Tags Mobile - Chat
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MobileConversationListResponse
// @Router /mobile/chat/conversations [get]
func MobileListConversations(db *sqlx.DB) gin.HandlerFunc {
	return handler.ListConversations(db)
}

// MobileGetConversationMessages returns messages in a conversation
// @Summary Get Chat Messages
// @Description Get history of messages for a specific conversation
// @Tags Mobile - Chat
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Conversation UUID"
// @Success 200 {object} MobileConversationMessagesResponse
// @Router /mobile/chat/conversations/{id}/messages [get]
func MobileGetConversationMessages(db *sqlx.DB) gin.HandlerFunc {
	return handler.GetConversationMessages(db)
}

// MobileSendMessage sends a message to a conversation
// @Summary Send Chat Message
// @Description Send a new message to an existing conversation
// @Tags Mobile - Chat
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Conversation UUID"
// @Param request body MobileChatSendMessageRequest true "Message Content"
// @Success 201 {object} ChatMessage
// @Router /mobile/chat/conversations/{id}/messages [post]
func MobileSendMessage(db *sqlx.DB) gin.HandlerFunc {
	return handler.SendMessage(db)
}

// MobileGetChatUnreadCount returns total unread messages
// @Summary Get Unread Count
// @Description Get total count of unread messages across all conversations
// @Tags Mobile - Chat
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MobileChatUnreadCountResponse
// @Router /mobile/chat/unread [get]
func MobileGetChatUnreadCount(db *sqlx.DB) gin.HandlerFunc {
	return handler.GetChatUnreadCount(db)
}

// MobileGetPeerLastActive returns peer's online status
// @Summary Get Peer Status
// @Description Get last seen and online status of a chat peer
// @Tags Mobile - Chat
// @Produce json
// @Security ApiKeyAuth
// @Param peer_type query string true "seller or archer"
// @Param peer_id query string true "Peer UUID"
// @Success 200 {object} MobilePeerLastActiveResponse
// @Router /mobile/chat/last-active [get]
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

