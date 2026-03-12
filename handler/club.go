package handler

import (
	"math"
	"net/http"
	"strconv"

	"archeryhub-api/models"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// GetClubs returns a list of clubs (data master)
func GetClubs(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		search := c.Query("search")

		var clubs []models.Club
		var total int

		query := `SELECT uuid, slug, name, abbreviation, description, banner_url, logo_url, phone, address, city, province, postal_code, established_date, registration_number, organization_id, head_coach_name, head_coach_phone, training_schedule, facilities, website, status, created_at, updated_at
		          FROM clubs WHERE (status = 'active')`
		countQuery := `SELECT COUNT(*) FROM clubs WHERE (status = 'active')`

		if search != "" {
			query += ` AND (name LIKE ? OR city LIKE ?)`
			countQuery += ` AND (name LIKE ? OR city LIKE ?)`
			searchParam := "%" + search + "%"

			err := db.Get(&total, countQuery, searchParam, searchParam)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			err = db.Select(&clubs, query+" LIMIT ? OFFSET ?", searchParam, searchParam, limit, offset)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			err := db.Get(&total, countQuery)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			err = db.Select(&clubs, query+" LIMIT ? OFFSET ?", limit, offset)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"data": clubs,
			"meta": gin.H{
				"total":       total,
				"page":        offset/limit + 1,
				"limit":       limit,
				"total_pages": math.Ceil(float64(total) / float64(limit)),
			},
		})
	}
}

// GetClubByID returns a single club by ID or slug
func GetClubByID(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var club models.Club
		err := db.Get(&club, "SELECT * FROM clubs WHERE (uuid = ? OR slug = ?) AND (status = 'active')", id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Club not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": club})
	}
}
