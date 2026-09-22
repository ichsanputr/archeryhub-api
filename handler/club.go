package handler

import (
	"fmt"
	"net/http"
	"strings"

	"Archeris-api/models"
	"Archeris-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// GetClubs returns a list of clubs (data master)
func GetClubs(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, offset, page := utils.GetPaginationParams(c)
		search := c.Query("search")

		whereClause := ""
		args := []interface{}{}

		if search != "" {
			whereClause = "WHERE name LIKE ? OR city LIKE ?"
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
			SELECT uuid, slug, name, logo_url, city, created_at, updated_at
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
		err := db.Get(&club, "SELECT * FROM clubs WHERE uuid = ? OR slug = ?", id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Klub tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": club})
	}
}

// CreateClub handles creating a new master club or returning existing if exact name exists
func CreateClub(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name         string `json:"name"`
			Abbreviation string `json:"abbreviation"`
			City         string `json:"city"`
			LogoURL      string `json:"logo_url"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format request tidak valid"})
			return
		}

		name := strings.TrimSpace(req.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nama klub wajib diisi"})
			return
		}

		// Check if club with exact name already exists
		var existing models.Club
		err := db.Get(&existing, "SELECT uuid, slug, name, logo_url, city, created_at, updated_at FROM clubs WHERE LOWER(name) = LOWER(?) LIMIT 1", name)
		if err == nil && existing.UUID != "" {
			c.JSON(http.StatusOK, gin.H{
				"message": "Klub sudah terdaftar",
				"data":    existing,
			})
			return
		}

		newUUID := uuid.New().String()
		slug := utils.CleanSlug(name)
		if slug == "" {
			slug = newUUID[:8]
		}

		// Ensure slug uniqueness
		var slugCount int
		_ = db.Get(&slugCount, "SELECT COUNT(*) FROM clubs WHERE slug = ?", slug)
		if slugCount > 0 {
			slug = fmt.Sprintf("%s-%s", slug, newUUID[:4])
		}

		_, err = db.Exec(`
			INSERT INTO clubs (uuid, slug, name, logo_url, city, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, NOW(), NOW())
		`, newUUID, slug, name, req.LogoURL, req.City)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan klub ke database", "details": err.Error()})
			return
		}

		var created models.Club
		_ = db.Get(&created, "SELECT uuid, slug, name, logo_url, city, created_at, updated_at FROM clubs WHERE uuid = ?", newUUID)

		c.JSON(http.StatusCreated, gin.H{
			"message": "Klub berhasil dibuat",
			"data":    created,
		})
	}
}
