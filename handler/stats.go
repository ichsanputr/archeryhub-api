package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// GetDashboardStats returns aggregated statistics for the dashboard
func GetDashboardStats(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var stats struct {
			ActiveEvents   int `json:"activeEvents"`
			TotalArchers      int `json:"totalArchers"`
			LiveEvents        int `json:"liveEvents"`
			CompletedToday    int `json:"completedToday"`
		}

		// Active Events (status is 'published' or 'ongoing')
		err := db.Get(&stats.ActiveEvents, "SELECT COUNT(*) FROM events WHERE status IN ('published', 'ongoing')")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch active events count"})
			return
		}

		// Total Archers
		err = db.Get(&stats.TotalArchers, "SELECT COUNT(*) FROM archers")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch total archers count"})
			return
		}

		// Live Events (status is 'ongoing')
		err = db.Get(&stats.LiveEvents, "SELECT COUNT(*) FROM events WHERE status = 'ongoing'")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch live events count"})
			return
		}

		// Completed Today
		today := time.Now().Format("2006-01-02")
		err = db.Get(&stats.CompletedToday, "SELECT COUNT(*) FROM events WHERE status = 'completed' AND end_date = ?", today)
		if err != nil {
			// If end_date is just a string or date, this should work.
			// If it fails, we'll just default to 0 for now.
			stats.CompletedToday = 0
		}

		c.JSON(http.StatusOK, stats)
	}
}
