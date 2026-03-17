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
// @Summary      Start or get chat conversation
// @Description  Archer starts (or gets existing) conversation with seller, optionally by product
// @Tags         Mobile - Chat
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      object{seller_id=string,product_id=string,product_name=string,product_image=string}  true  "Conversation payload"
// @Success      200      {object}  handler.ChatConversation
// @Success      201      {object}  handler.ChatConversation
// @Failure      400      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Router       /mobile/chat/conversations [post]
func MobileStartOrGetConversation(db *sqlx.DB) gin.HandlerFunc {
	return handler.StartOrGetConversation(db)
}

// MobileListConversations godoc
// @Summary      List chat conversations
// @Description  Lists archer/seller conversations including last message, unread counters, and last seen info
// @Tags         Mobile - Chat
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  MobileChatListResponse
// @Failure      403  {object}  ErrorResponse
// @Router       /mobile/chat/conversations [get]
func MobileListConversations(db *sqlx.DB) gin.HandlerFunc {
	return handler.ListConversations(db)
}

// MobileGetConversationMessages godoc
// @Summary      Get conversation messages
// @Description  Returns message list in a conversation and marks opposite side unread as read
// @Tags         Mobile - Chat
// @Produce      json
// @Security     BearerAuth
// @Param        id  path      string  true  "Conversation UUID"
// @Success      200 {object}  MobileChatMessagesResponse
// @Failure      403 {object}  ErrorResponse
// @Failure      404 {object}  ErrorResponse
// @Router       /mobile/chat/conversations/{id}/messages [get]
func MobileGetConversationMessages(db *sqlx.DB) gin.HandlerFunc {
	return handler.GetConversationMessages(db)
}

// MobileSendMessage godoc
// @Summary      Send chat message
// @Description  Sends a message to a conversation as archer or seller participant
// @Tags         Mobile - Chat
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                   true  "Conversation UUID"
// @Param        request  body      object{message=string}  true  "Message payload"
// @Success      201      {object}  handler.ChatMessage
// @Failure      400      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Router       /mobile/chat/conversations/{id}/messages [post]
func MobileSendMessage(db *sqlx.DB) gin.HandlerFunc {
	return handler.SendMessage(db)
}

// MobileGetChatUnreadCount godoc
// @Summary      Get unread chat count
// @Description  Returns total unread chats for current archer/seller
// @Tags         Mobile - Chat
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  MobileChatUnreadResponse
// @Router       /mobile/chat/unread [get]
func MobileGetChatUnreadCount(db *sqlx.DB) gin.HandlerFunc {
	return handler.GetChatUnreadCount(db)
}

// MobileGetPeerLastActive godoc
// @Summary      Get peer last active
// @Description  Returns last_seen_at and online flag for chat peer (seller or archer)
// @Tags         Mobile - Chat
// @Produce      json
// @Security     BearerAuth
// @Param        peer_type  query     string  true  "Peer type: seller or archer"
// @Param        peer_id    query     string  true  "Peer UUID"
// @Success      200        {object}  MobileChatLastActiveResponse
// @Failure      400        {object}  ErrorResponse
// @Failure      404        {object}  ErrorResponse
// @Router       /mobile/chat/last-active [get]
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
