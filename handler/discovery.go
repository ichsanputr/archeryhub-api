package handler

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// GetSitemapData returns all public slugs for sitemap generation
func GetSitemapData(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		type SitemapData struct {
			Events        []string `json:"events"`
			Archers       []string `json:"archers"`
			Clubs         []string `json:"clubs"`
			Organizations []string `json:"organizations"`
			Products      []string `json:"products"`
			News          []string `json:"news"`
		}

		var data SitemapData

		// Fetch Event slugs
		err := db.Select(&data.Events, "SELECT slug FROM events WHERE slug IS NOT NULL AND slug != '' AND status IN ('published', 'ongoing', 'completed')")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch event slugs: " + err.Error()})
			return
		}

		// Fetch Archer usernames
		err = db.Select(&data.Archers, "SELECT username FROM archers WHERE username IS NOT NULL AND username != ''")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch archer usernames: " + err.Error()})
			return
		}

		// Fetch Club slugs
		err = db.Select(&data.Clubs, "SELECT slug FROM clubs WHERE slug IS NOT NULL AND slug != '' AND (status = 'active' OR status = 'verified')")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch club slugs: " + err.Error()})
			return
		}

		// Fetch Organization slugs
		err = db.Select(&data.Organizations, "SELECT slug FROM organizations WHERE slug IS NOT NULL AND slug != '' AND verification_status = 'verified'")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch organization slugs: " + err.Error()})
			return
		}

		// Fetch Product slugs
		err = db.Select(&data.Products, "SELECT slug FROM products WHERE slug IS NOT NULL AND slug != '' AND status = 'active'")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product slugs: " + err.Error()})
			return
		}

		// Fetch News slugs
		err = db.Select(&data.News, "SELECT slug FROM news WHERE slug IS NOT NULL AND slug != '' AND status = 'published'")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch news slugs: " + err.Error()})
			return
		}

		// Ensure no nil slices
		if data.Events == nil { data.Events = []string{} }
		if data.Archers == nil { data.Archers = []string{} }
		if data.Clubs == nil { data.Clubs = []string{} }
		if data.Organizations == nil { data.Organizations = []string{} }
		if data.Products == nil { data.Products = []string{} }
		if data.News == nil { data.News = []string{} }

		c.JSON(http.StatusOK, data)
	}
}
