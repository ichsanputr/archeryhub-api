package mobile

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type OptionData struct {
	ID   string `db:"uuid" json:"id"`
	Name string `db:"name" json:"name"`
}

// @Summary Get Club Options
// @Description Get list of clubs for dropdown/selection
// @Tags Mobile - Option
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /mobile/options/clubs [get]
func GetClubOptions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data []OptionData
		err := db.Select(&data, "SELECT uuid, name FROM clubs ORDER BY name ASC")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch clubs"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": data})
	}
}

// @Summary Get Organization Options
// @Description Get list of organizations for dropdown/selection
// @Tags Mobile - Option
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /mobile/options/organizations [get]
func GetOrganizationOptions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data []OptionData
		err := db.Select(&data, "SELECT uuid, name FROM organizations ORDER BY name ASC")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch organizations"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": data})
	}
}

// @Summary Get Discipline Options
// @Description Get list of archery disciplines
// @Tags Mobile - Option
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /mobile/options/disciplines [get]
func GetDisciplineOptions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data []OptionData
		err := db.Select(&data, "SELECT uuid, name FROM ref_disciplines ORDER BY name ASC")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch disciplines"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": data})
	}
}

// @Summary Get Bow Type Options
// @Description Get list of bow types
// @Tags Mobile - Option
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /mobile/options/bow-types [get]
func GetBowTypeOptions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data []OptionData
		err := db.Select(&data, "SELECT uuid, name FROM ref_bow_types ORDER BY name ASC")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bow types"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": data})
	}
}

// @Summary Get Age Group Options
// @Description Get list of age groups
// @Tags Mobile - Option
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /mobile/options/age-groups [get]
func GetAgeGroupOptions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data []OptionData
		err := db.Select(&data, "SELECT uuid, name FROM ref_age_groups ORDER BY name ASC")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch age groups"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": data})
	}
}

// @Summary Get Gender Division Options
// @Description Get list of gender divisions
// @Tags Mobile - Option
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /mobile/options/gender-divisions [get]
func GetGenderDivisionOptions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data []OptionData
		err := db.Select(&data, "SELECT uuid, name FROM ref_gender_divisions ORDER BY name ASC")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch gender divisions"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": data})
	}
}

// @Summary Get City Options
// @Description Get list of Indonesian cities
// @Tags Mobile - Option
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /mobile/options/cities [get]
func GetCityOptions() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := filepath.Join("data", "cities.json")
		file, err := os.ReadFile(path)
		if err != nil {
			path = filepath.Join("api", "data", "cities.json")
			file, err = os.ReadFile(path)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load cities"})
				return
			}
		}

		var cities []interface{}
		json.Unmarshal(file, &cities)
		c.JSON(http.StatusOK, gin.H{"data": cities})
	}
}

// @Summary Get Event Type Options
// @Description Get list of event/team types
// @Tags Mobile - Option
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /mobile/options/event-types [get]
func GetEventTypeOptions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data []OptionData
		err := db.Select(&data, "SELECT uuid, name FROM ref_event_types ORDER BY name ASC")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch event types"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": data})
	}
}
