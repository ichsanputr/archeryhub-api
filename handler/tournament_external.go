package handler

import (
	"encoding/json"
	"net/http"

	"Archeris-api/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// GetExternalTournaments returns list of external scraped tournaments
func GetExternalTournaments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tournaments []models.TournamentExternalListItem
		query := `
			SELECT 
				id, uuid, slug, source_platform, external_id,
				name, short_name, venue, location, city, country,
				start_date, end_date, banner_url, logo_url, status,
				categories_count, participants_count
			FROM tournament_externals
			ORDER BY start_date DESC, created_at DESC
		`
		err := db.Select(&tournaments, query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data turnamen eksternal", "details": err.Error()})
			return
		}

		if tournaments == nil {
			tournaments = []models.TournamentExternalListItem{}
		}

		c.JSON(http.StatusOK, gin.H{
			"tournaments": tournaments,
			"total":       len(tournaments),
		})
	}
}

// GetExternalTournamentDetail returns detail + structured data_json of external tournament
func GetExternalTournamentDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slugOrID := c.Param("slug")
		trimmedID := slugOrID
		if len(slugOrID) > 7 && slugOrID[:7] == "ianseo-" {
			trimmedID = slugOrID[7:]
		}

		var t models.TournamentExternal
		query := `
			SELECT 
				id, uuid, slug, source_platform, external_id, source_url,
				name, short_name, venue, location, city, country,
				start_date, end_date, banner_url, logo_url, status,
				categories_count, participants_count, data_json,
				created_at, updated_at
			FROM tournament_externals
			WHERE slug = ? OR external_id = ? OR external_id = ? OR uuid = ?
			LIMIT 1
		`
		err := db.Get(&t, query, slugOrID, slugOrID, trimmedID, slugOrID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen eksternal tidak ditemukan"})
			return
		}

		var parsedData map[string]interface{}
		if t.DataJSON != "" {
			_ = json.Unmarshal([]byte(t.DataJSON), &parsedData)
		}

		c.JSON(http.StatusOK, gin.H{
			"tournament": t,
			"data":       parsedData,
		})
	}
}
