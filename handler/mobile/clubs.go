package mobile

import (
	"Archeris-api/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type MobileClubItem struct {
	UUID      string  `json:"uuid" db:"uuid"`
	Slug      *string `json:"slug" db:"slug"`
	Name      string  `json:"name" db:"name"`
	LogoURL   *string `json:"logo_url" db:"logo_url"`
	City      *string `json:"city" db:"city"`
	CreatedAt string  `json:"created_at" db:"created_at"`
}

type MobileClubListResponse struct {
	Clubs      []MobileClubItem `json:"clubs"`
	TotalCount int              `json:"total_count"`
}

func MobileListClubs(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		search := c.Query("search")

		whereClause := ""
		args := []interface{}{}
		if search != "" {
			whereClause = "WHERE name LIKE ? OR city LIKE ?"
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm)
		}

		query := `
			SELECT uuid, slug, name, logo_url, city, created_at
			FROM clubs
			` + whereClause + `
			ORDER BY name ASC
			LIMIT ? OFFSET ?
		`
		args = append(args, limit, offset)

		var clubs []MobileClubItem
		if err := db.Select(&clubs, query, args...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data klub", "details": err.Error()})
			return
		}

		if clubs == nil {
			clubs = []MobileClubItem{}
		}

		for i := range clubs {
			if clubs[i].LogoURL != nil {
				masked := utils.MaskMediaURL(*clubs[i].LogoURL)
				clubs[i].LogoURL = &masked
			}
		}

		c.JSON(http.StatusOK, MobileClubListResponse{
			Clubs:      clubs,
			TotalCount: len(clubs),
		})
	}
}
