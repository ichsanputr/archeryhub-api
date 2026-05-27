package handler

import (
	"fmt"
	"net/http"

	"Archeris-api/models"
	"Archeris-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// GetClubs returns a list of clubs (data master)
func GetClubs(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, offset, page := utils.GetPaginationParams(c)
		search := c.Query("search")

		whereClause := "WHERE status = 'active'"
		args := []interface{}{}

		if search != "" {
			whereClause += " AND (name LIKE ? OR city LIKE ?)"
			searchParam := "%" + search + "%"
			args = append(args, searchParam, searchParam)
		}

		// Count total
		var total int
		err := db.Get(&total, "SELECT COUNT(*) FROM clubs "+whereClause, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung klub"})
			return
		}

		// Get data
		var clubs []models.Club
		query := fmt.Sprintf(`
			SELECT uuid, slug, name, abbreviation, logo_url, city, status, created_at, updated_at
			FROM clubs %s ORDER BY name ASC LIMIT ? OFFSET ?
		`, whereClause)
		queryArgs := append(args, limit, offset)

		err = db.Select(&clubs, query, queryArgs...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data klub"})
			return
		}

		if clubs == nil {
			clubs = []models.Club{}
		}

		meta := utils.CalculatePagination(total, limit, offset, page)
		c.JSON(http.StatusOK, gin.H{
			"data": clubs,
			"meta": meta,
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Klub tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": club})
	}
}

