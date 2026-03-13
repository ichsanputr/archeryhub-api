package handler

import (
	"archeryhub-api/utils"
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
			Slug         *string `json:"slug" db:"slug"`
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
			SELECT uuid, store_name, slug, description, avatar_url, banner_url, 
			       phone, email, address, city, province, page_settings
			FROM sellers
			WHERE uuid = ? OR user_id = ?`, userID, userID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Penjual tidak ditemukan"})
			return
		}

		// Prepare data
		data := make(map[string]interface{})
		if seller.PageSettings != nil && *seller.PageSettings != "" {
			json.Unmarshal([]byte(*seller.PageSettings), &data)
		}

		// Mask URLs
		if seller.AvatarURL != nil {
			masked := utils.MaskMediaURL(*seller.AvatarURL)
			seller.AvatarURL = &masked
		}
		if seller.BannerURL != nil {
			masked := utils.MaskMediaURL(*seller.BannerURL)
			seller.BannerURL = &masked
		}

		// Add seller basic info
		data["id"] = seller.UUID
		data["uuid"] = seller.UUID
		data["store_name"] = seller.StoreName
		data["slug"] = seller.Slug
		data["store_slug"] = seller.Slug
		data["description"] = seller.Description
		data["avatar_url"] = seller.AvatarURL
		data["banner_url"] = seller.BannerURL
		data["phone"] = seller.Phone
		data["email"] = seller.Email
		data["address"] = seller.Address
		data["city"] = seller.City
		data["province"] = seller.Province
		data["user_type"] = "seller"

		c.JSON(http.StatusOK, gin.H{"data": data})
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
		err := db.Get(&currentPageSettings, "SELECT page_settings FROM sellers WHERE uuid = ? OR user_id = ?", userID, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Akun penjual tidak ditemukan"})
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
			WHERE uuid = ? OR user_id = ?`,
			string(pageSettingsJSON), userID, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Profil berhasil diperbarui"})
	}
}

// UpdateSellerProfileBasic updates basic seller profile info (name, contact, location)
func UpdateSellerProfileBasic(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		var req struct {
			StoreName   *string `json:"store_name"`
			Email       *string `json:"email"`
			Phone       *string `json:"phone"`
			City        *string `json:"city"`
			Province    *string `json:"province"`
			Address     *string `json:"address"`
			Description *string `json:"description"`
			AvatarURL   *string `json:"avatar_url"`
			BannerURL   *string `json:"banner_url"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		query := "UPDATE sellers SET updated_at = NOW()"
		args := []interface{}{}

		if req.StoreName != nil {
			query += ", store_name = ?"
			args = append(args, *req.StoreName)
		}
		if req.Email != nil {
			query += ", email = ?"
			args = append(args, *req.Email)
		}
		if req.Phone != nil {
			query += ", phone = ?"
			args = append(args, *req.Phone)
		}
		if req.City != nil {
			query += ", city = ?"
			args = append(args, *req.City)
		}
		if req.Province != nil {
			query += ", province = ?"
			args = append(args, *req.Province)
		}
		if req.Address != nil {
			query += ", address = ?"
			args = append(args, *req.Address)
		}
		if req.Description != nil {
			query += ", description = ?"
			args = append(args, *req.Description)
		}
		if req.AvatarURL != nil {
			query += ", avatar_url = ?"
			args = append(args, *req.AvatarURL)
		}
		if req.BannerURL != nil {
			query += ", banner_url = ?"
			args = append(args, *req.BannerURL)
		}

		query += " WHERE uuid = ? OR user_id = ?"
		args = append(args, userID, userID)

		_, err := db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui profil penjual: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Profil toko berhasil diperbarui"})
	}
}
