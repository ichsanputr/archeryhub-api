package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func GetSellerProfile(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		var seller struct {
			UUID         string  `json:"uuid" db:"uuid"`
			StoreName    string  `json:"store_name" db:"store_name"`
			StoreSlug    *string `json:"store_slug" db:"store_slug"`
			Description  *string `json:"description" db:"description"`
			AvatarURL    *string `json:"avatar_url" db:"avatar_url"`
			BannerURL    *string `json:"banner_url" db:"banner_url"`
			Phone        *string `json:"phone" db:"phone"`
			Email        *string `json:"email" db:"email"`
			Address      *string `json:"address" db:"address"`
			City         *string `json:"city" db:"city"`
			Province     *string `json:"province" db:"province"`
			PageSettings *string `json:"page_settings" db:"page_settings"`
		}

		err := db.Get(&seller, `
			SELECT uuid, store_name, store_slug, description, avatar_url, banner_url, 
			       phone, email, address, city, province, page_settings
			FROM sellers
			WHERE user_id = ?`, userID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Seller not found"})
			return
		}

		// Parse page_settings to extract sections, catalog_config, etc.
		var response map[string]interface{}
		if seller.PageSettings != nil && *seller.PageSettings != "" {
			json.Unmarshal([]byte(*seller.PageSettings), &response)
		} else {
			response = make(map[string]interface{})
		}

		// Add seller basic info
		response["uuid"] = seller.UUID
		response["store_name"] = seller.StoreName
		response["store_slug"] = seller.StoreSlug
		response["description"] = seller.Description
		response["avatar_url"] = seller.AvatarURL
		response["banner_url"] = seller.BannerURL
		response["phone"] = seller.Phone
		response["email"] = seller.Email
		response["address"] = seller.Address
		response["city"] = seller.City
		response["province"] = seller.Province

		c.JSON(http.StatusOK, gin.H{"data": response})
	}
}

func UpdateSellerProfile(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		var req struct {
			Sections      interface{} `json:"sections"`
			CatalogConfig interface{} `json:"catalog_config"`
			ThemeColor    string      `json:"theme_color"`
			BannerText    string      `json:"banner_text"`
			PageSettings  interface{} `json:"page_settings"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get current page_settings
		var currentPageSettings *string
		err := db.Get(&currentPageSettings, "SELECT page_settings FROM sellers WHERE user_id = ?", userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Seller account not found"})
			return
		}

		// Build page_settings JSON
		var pageSettingsMap map[string]interface{}
		if currentPageSettings != nil && *currentPageSettings != "" {
			json.Unmarshal([]byte(*currentPageSettings), &pageSettingsMap)
		}
		if pageSettingsMap == nil {
			pageSettingsMap = make(map[string]interface{})
		}

		// Update from request (backward compatibility with old fields)
		if req.Sections != nil {
			pageSettingsMap["sections"] = req.Sections
		}
		if req.CatalogConfig != nil {
			pageSettingsMap["catalog_config"] = req.CatalogConfig
		}
		if req.ThemeColor != "" {
			pageSettingsMap["theme_color"] = req.ThemeColor
		}
		if req.BannerText != "" {
			pageSettingsMap["banner_text"] = req.BannerText
		}

		// If page_settings is provided directly, merge it
		if req.PageSettings != nil {
			var providedSettings map[string]interface{}
			if pageSettingsStr, ok := req.PageSettings.(string); ok {
				json.Unmarshal([]byte(pageSettingsStr), &providedSettings)
			} else {
				pageSettingsBytes, _ := json.Marshal(req.PageSettings)
				json.Unmarshal(pageSettingsBytes, &providedSettings)
			}
			// Merge provided settings
			for k, v := range providedSettings {
				pageSettingsMap[k] = v
			}
		}

		pageSettingsJSON, _ := json.Marshal(pageSettingsMap)

		_, err = db.Exec(`
			UPDATE sellers SET page_settings = ?, updated_at = NOW()
			WHERE user_id = ?`,
			string(pageSettingsJSON), userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
	}
}
