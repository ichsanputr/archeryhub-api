package mobile

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// MobileRecentResults returns recent completed events with top qualification results
func MobileRecentResults(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		type CategoryResult struct {
			CategoryName string  `json:"category_name"`
			Standings    []gin.H `json:"standings"`
		}

		type EventResult struct {
			EventTitle string          `json:"event_title"`
			EventSlug  string          `json:"event_slug"`
			Status     string          `json:"status"`
			EndDate    *string         `json:"end_date"`
			Categories []CategoryResult `json:"categories"`
		}

		var recentEvents []struct {
			UUID    string  `db:"uuid"`
			Name    string  `db:"name"`
			Slug    string  `db:"slug"`
			EndDate *string `db:"end_date"`
		}
		err := db.Select(&recentEvents, `
			SELECT uuid, name, slug, end_date
			FROM events
			WHERE end_date < NOW() AND end_date > DATE_SUB(NOW(), INTERVAL 30 DAY)
			AND status IN ('published', 'active')
			ORDER BY end_date DESC
			LIMIT 5
		`)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"results": []EventResult{}})
			return
		}

		var results []EventResult

		for _, ev := range recentEvents {
			var categories []struct {
				UUID string `db:"uuid"`
				Name string `db:"name"`
			}
			_ = db.Select(&categories, `
				SELECT ec.uuid, COALESCE(ec.category_name_custom, rag.name) as name
				FROM event_categories ec
				LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
				WHERE ec.event_id = ? AND ec.status = 'active'
			`, ev.UUID)

			var catResults []CategoryResult

			for _, cat := range categories {
				type Standing struct {
					Name  string  `db:"name" json:"name"`
					Score float64 `db:"score" json:"score"`
				}
				var standings []Standing
				_ = db.Select(&standings, `
					SELECT a.full_name as name,
						COALESCE(ep.qual_score, ep.payment_amount, 0) as score
					FROM event_participants ep
					JOIN archers a ON ep.archer_id = a.uuid
					WHERE ep.event_id = ? AND ep.category_id = ?
					ORDER BY ep.qual_score DESC
					LIMIT 5
				`, ev.UUID, cat.UUID)

				if len(standings) > 0 {
					standingList := make([]gin.H, 0)
					for _, s := range standings {
						standingList = append(standingList, gin.H{
							"name":  s.Name,
							"score": s.Score,
						})
					}
					catResults = append(catResults, CategoryResult{
						CategoryName: cat.Name,
						Standings:    standingList,
					})
				}
			}

			results = append(results, EventResult{
				EventTitle: ev.Name,
				EventSlug:  ev.Slug,
				Status:     "completed",
				EndDate:    ev.EndDate,
				Categories: catResults,
			})
		}

		if results == nil {
			results = []EventResult{}
		}

		c.JSON(http.StatusOK, gin.H{"results": results})
	}
}
