package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Reference data structure
type RefData struct {
	UUID string `db:"uuid" json:"id"`
	Code string `db:"code" json:"code"`
	Name string `db:"name" json:"name"`
}

// GetDisciplines returns all archery disciplines
func GetDisciplines(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data []RefData
		err := db.Select(&data, "SELECT uuid, code, name FROM ref_disciplines ORDER BY name")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch disciplines"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"disciplines": data, "total": len(data)})
	}
}

// GetBowTypes returns all bow types (formerly divisions)
func GetBowTypes(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data []RefData
		err := db.Select(&data, "SELECT uuid, code, name FROM ref_bow_types ORDER BY name")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bow types"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"bow_types": data, "total": len(data)})
	}
}

// GetEventTypes returns all event types
func GetEventTypes(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data []RefData
		err := db.Select(&data, "SELECT uuid, code, name FROM ref_event_types ORDER BY name")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch event types"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"event_types": data, "total": len(data)})
	}
}

// GetGenderDivisions returns all gender divisions
func GetGenderDivisions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data []RefData
		err := db.Select(&data, "SELECT uuid, code, name FROM ref_gender_divisions ORDER BY name")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch gender divisions"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"gender_divisions": data, "total": len(data)})
	}
}

// GetAgeGroups returns all age groups (formerly categories)
func GetAgeGroups(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data []RefData
		err := db.Select(&data, "SELECT uuid, code, name FROM ref_age_groups ORDER BY name")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch age groups"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"age_groups": data, "total": len(data)})
	}
}
