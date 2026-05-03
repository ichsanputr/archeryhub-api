package mobile

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// @Summary Get Organization Dashboard Statistics
// @Description Get dashboard statistics, recent participants and payments for organization
// @Tags Mobile - Organization
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MobileOrganizationDashboardResponse
// @Router /mobile/organization/dashboard [get]
func MobileGetOrganizationDashboard(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var dashboard MobileOrganizationDashboardResponse

		// 1. Stats
		// Events
		_ = db.Get(&dashboard.Stats.TotalEvents, "SELECT COUNT(*) FROM events WHERE organizer_id = ?", userID)
		_ = db.Get(&dashboard.Stats.ActiveEvents, "SELECT COUNT(*) FROM events WHERE organizer_id = ? AND status = 'active'", userID)

		// Participants
		_ = db.Get(&dashboard.Stats.TotalParticipants, `
			SELECT COUNT(ep.uuid) 
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			WHERE e.organizer_id = ?
		`, userID)

		// Revenue
		_ = db.Get(&dashboard.Stats.TotalRevenue, `
			SELECT COALESCE(SUM(t.amount), 0)
			FROM payment_transactions t
			JOIN events e ON t.event_id = e.uuid
			WHERE e.organizer_id = ? AND t.status = 'paid' AND t.registration_id IS NOT NULL
		`, userID)

		firstOfMonth := time.Now().AddDate(0, 0, -time.Now().Day()+1).Format("2006-01-02")
		_ = db.Get(&dashboard.Stats.MonthlyRevenue, `
			SELECT COALESCE(SUM(t.amount), 0)
			FROM payment_transactions t
			JOIN events e ON t.event_id = e.uuid
			WHERE e.organizer_id = ? AND t.status = 'paid' AND t.registration_id IS NOT NULL AND t.paid_at >= ?
		`, userID, firstOfMonth)

		_ = db.Get(&dashboard.Stats.PendingRevenue, `
			SELECT COALESCE(SUM(t.amount), 0)
			FROM payment_transactions t
			JOIN events e ON t.event_id = e.uuid
			WHERE e.organizer_id = ? AND t.status = 'pending' AND t.registration_id IS NOT NULL
		`, userID)

		// 2. Recent Participants
		_ = db.Select(&dashboard.RecentParticipants, `
			SELECT a.full_name, e.name as event_name, ep.created_at
			FROM event_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			JOIN events e ON ep.event_id = e.uuid
			WHERE e.organizer_id = ?
			ORDER BY ep.created_at DESC
			LIMIT 5
		`, userID)

		// 3. Recent Payments
		_ = db.Select(&dashboard.RecentPayments, `
			SELECT t.amount, a.full_name, e.name as event_name, t.paid_at, t.status
			FROM payment_transactions t
			JOIN events e ON t.event_id = e.uuid
			LEFT JOIN event_participants ep ON t.registration_id = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			WHERE e.organizer_id = ? AND t.registration_id IS NOT NULL
			ORDER BY t.created_at DESC
			LIMIT 5
		`, userID)

		// 4. Upcoming Deadlines
		_ = db.Select(&dashboard.UpcomingDeadlines, `
			SELECT name, registration_deadline
			FROM events
			WHERE organizer_id = ? AND registration_deadline > NOW() AND status = 'active'
			ORDER BY registration_deadline ASC
			LIMIT 3
		`, userID)

		// Calculate days left
		for i := range dashboard.UpcomingDeadlines {
			dashboard.UpcomingDeadlines[i].DaysLeft = int(time.Until(dashboard.UpcomingDeadlines[i].Deadline).Hours() / 24)
		}

		c.JSON(http.StatusOK, dashboard)
	}
}
