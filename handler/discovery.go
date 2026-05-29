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

			Organizers []string `json:"organizers"`
			Products      []string `json:"products"`
			News          []string `json:"news"`
		}

		var data SitemapData

		// Fetch Event slugs - include anything that isn't a draft
		err := db.Select(&data.Events, "SELECT slug FROM events WHERE slug IS NOT NULL AND slug != '' AND status != 'draft'")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data slug event: " + err.Error()})
			return
		}

		// Fetch Archer usernames
		err = db.Select(&data.Archers, "SELECT username FROM archers WHERE username IS NOT NULL AND username != ''")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil username atlet: " + err.Error()})
			return
		}



		// Fetch Organizer slugs
		err = db.Select(&data.Organizers, "SELECT slug FROM organizers WHERE slug IS NOT NULL AND slug != '' AND status = 'active'")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data slug organisasi: " + err.Error()})
			return
		}

		// Fetch Product slugs
		err = db.Select(&data.Products, "SELECT slug FROM products WHERE slug IS NOT NULL AND slug != '' AND status = 'active'")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data slug produk: " + err.Error()})
			return
		}

		// Fetch News slugs
		err = db.Select(&data.News, "SELECT slug FROM news WHERE slug IS NOT NULL AND slug != '' AND status = 'published'")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data slug berita: " + err.Error()})
			return
		}

		// Ensure no nil slices
		if data.Events == nil { data.Events = []string{} }
		if data.Archers == nil { data.Archers = []string{} }

		if data.Organizers == nil { data.Organizers = []string{} }
		if data.Products == nil { data.Products = []string{} }
		if data.News == nil { data.News = []string{} }

		c.JSON(http.StatusOK, data)
	}
}
