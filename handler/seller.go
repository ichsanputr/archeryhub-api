package handler

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"juno.com/archeryhub/api/models"
)

type SellerProfile struct {
	UUID          string `json:"id" db:"uuid"`
	SellerID      string `json:"seller_id" db:"seller_id"`
	Sections      string `json:"sections" db:"sections"` // JSON string
	CatalogConfig string `json:"catalog_config" db:"catalog_config"` // JSON string
	ThemeColor    string `json:"theme_color" db:"theme_color"`
	BannerText    string `json:"banner_text" db:"banner_text"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

func GetSellerProfile(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		
		var profile SellerProfile
		err := db.QueryRow(`
			SELECT sp.* FROM seller_profiles sp
			JOIN sellers s ON sp.seller_id = s.uuid
			WHERE s.user_id = ?`, userID).Scan(
			&profile.UUID, &profile.SellerID, &profile.Sections, 
			&profile.CatalogConfig, &profile.ThemeColor, &profile.BannerText,
			&profile.CreatedAt, &profile.UpdatedAt,
		)

		if err == sql.ErrNoRows {
			// Return default profile or empty
			c.JSON(http.StatusOK, gin.H{"data": nil})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": profile})
	}
}

func UpdateSellerProfile(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		
		var req struct {
			Sections      interface{} `json:"sections"`
			CatalogConfig interface{} `json:"catalog_config"`
			ThemeColor    string      `json:"theme_color"`
			BannerText    string      `json:"banner_text"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get seller_id first
		var sellerID string
		err := db.QueryRow("SELECT uuid FROM sellers WHERE user_id = ?", userID).Scan(&sellerID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Seller account not found"})
			return
		}

		sectionsJSON := models.ToJSON(req.Sections)
		catalogJSON := models.ToJSON(req.CatalogConfig)

		_, err = db.Exec(`
			INSERT INTO seller_profiles (uuid, seller_id, sections, catalog_config, theme_color, banner_text)
			VALUES (?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE 
				sections = VALUES(sections),
				catalog_config = VALUES(catalog_config),
				theme_color = VALUES(theme_color),
				banner_text = VALUES(banner_text),
				updated_at = CURRENT_TIMESTAMP`,
			uuid.New().String(), sellerID, sectionsJSON, catalogJSON, req.ThemeColor, req.BannerText,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
	}
}
