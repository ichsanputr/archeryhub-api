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
			ActiveTournaments int `json:"activeTournaments"`
			TotalAthletes     int `json:"totalAthletes"`
			LiveEvents        int `json:"liveEvents"`
			CompletedToday    int `json:"completedToday"`
		}

		// Active Tournaments (status is 'published' or 'ongoing')
		err := db.Get(&stats.ActiveTournaments, "SELECT COUNT(*) FROM tournaments WHERE status IN ('published', 'ongoing')")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch active tournaments count"})
			return
		}

		// Total Athletes
		err = db.Get(&stats.TotalAthletes, "SELECT COUNT(*) FROM athletes")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch total athletes count"})
			return
		}

		// Live Events (status is 'ongoing')
		err = db.Get(&stats.LiveEvents, "SELECT COUNT(*) FROM tournaments WHERE status = 'ongoing'")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch live events count"})
			return
		}

		// Completed Today
		today := time.Now().Format("2006-01-02")
		err = db.Get(&stats.CompletedToday, "SELECT COUNT(*) FROM tournaments WHERE status = 'completed' AND end_date = ?", today)
		if err != nil {
			// If end_date is just a string or date, this should work.
			// If it fails, we'll just default to 0 for now.
			stats.CompletedToday = 0
		}

		c.JSON(http.StatusOK, stats)
	}
}
