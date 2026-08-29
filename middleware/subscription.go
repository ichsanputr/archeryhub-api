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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			c.Abort()
			return
		}

		userType, _ := c.Get("user_type")
		if userType == "root" {
			c.Next()
			return
		}
		if userType == "archer" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Fitur ini hanya dapat diakses oleh akun Penyelenggara (Organizer) atau Klub.",
				"code":  "organizer_role_required",
			})
			c.Abort()
			return
		}

		// Check organizer: either has active monthly subscription OR has quota > 0
		var org struct {
			Status        string `db:"subscription_status"`
			QuotaStandard int    `db:"quota_standard"`
			QuotaElite    int    `db:"quota_elite"`
		}
		err := db.Get(&org, "SELECT COALESCE(subscription_status,'trial') as subscription_status, quota_standard, quota_elite FROM organizers WHERE user_id = ?", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify account status"})
			c.Abort()
			return
		}

		// Allow if: active monthly sub OR has any quota
		hasMonthly := org.Status == "active" || org.Status == "trial"
		hasQuota := org.QuotaStandard > 0 || org.QuotaElite > 0

		if !hasMonthly && !hasQuota {
			// Also allow if they have any published events (already consumed quota)
			var publishedCount int
			_ = db.Get(&publishedCount, "SELECT COUNT(*) FROM events e JOIN organizers o ON e.organizer_id = o.uuid WHERE o.user_id = ? AND e.status = 'published' AND e.quota_type IS NOT NULL", userID)
			if publishedCount == 0 {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error": "Quota required",
					"code": "quota_required",
					"message": "Silakan beli quota event untuk menggunakan fitur ini.",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
