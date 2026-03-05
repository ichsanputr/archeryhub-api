package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// RequireActivePlan ensures the user has an active or trial subscription
func RequireActivePlan(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			// Some endpoints might use OptionalAuthMiddleware, so we check if userID is present
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			c.Abort()
			return
		}

		userType, _ := c.Get("user_type")
		
		// Admin/Root bypasses subscription checks
		if userType == "root" {
			c.Next()
			return
		}

		// Only Organizations (EO) and Clubs require subscription gating for certain features
		if userType != "organization" && userType != "club" {
			c.Next()
			return
		}

		var status string
		var err error

		if userType == "organization" {
			err = db.Get(&status, "SELECT COALESCE(subscription_status, 'trial') FROM organizations WHERE user_id = ?", userID)
		} else {
			err = db.Get(&status, "SELECT COALESCE(subscription_status, 'trial') FROM clubs WHERE user_id = ?", userID)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify subscription status"})
			c.Abort()
			return
		}

		// status can be 'trial', 'active', 'expired', 'canceled'
		if status != "active" && status != "trial" {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error": "Subscription required",
				"code": "subscription_expired",
				"message": "Fitur ini hanya tersedia untuk akun dengan langganan aktif. Silakan perbarui paket Anda.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
