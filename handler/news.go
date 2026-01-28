package handler

import (
	"net/http"
	"strings"
	"time"

	"archeryhub-api/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// News represents a news article
type News struct {
	UUID            string  `db:"uuid" json:"id"`
	OrganizationID  *string `db:"organization_id" json:"organization_id,omitempty"`
	ClubID          *string `db:"club_id" json:"club_id,omitempty"`
	Title           string  `db:"title" json:"title"`
	Slug            string  `db:"slug" json:"slug"`
	Excerpt         *string `db:"excerpt" json:"excerpt,omitempty"`
	Content         *string `db:"content" json:"content,omitempty"`
	ImageURL        *string `db:"image_url" json:"image_url,omitempty"`
	Category        string  `db:"category" json:"category"`
	Status          string  `db:"status" json:"status"`
	Views           int     `db:"views" json:"views"`
	AuthorName      *string `db:"author_name" json:"author_name,omitempty"`
	AuthorID        *string `db:"author_id" json:"author_id,omitempty"`
	MetaTitle       *string `db:"meta_title" json:"meta_title,omitempty"`
	MetaDescription *string `db:"meta_description" json:"meta_description,omitempty"`
	PublishedAt     *string `db:"published_at" json:"published_at,omitempty"`
	CreatedAt       string  `db:"created_at" json:"created_at"`
	UpdatedAt       string  `db:"updated_at" json:"updated_at"`
}

// CreateNewsRequest represents the request to create news
type CreateNewsRequest struct {
	Title           string  `json:"title" binding:"required"`
	Excerpt         string  `json:"excerpt"`
	Content         string  `json:"content"`
	ImageURL        string  `json:"image_url"`
	Category        string  `json:"category"`
	Status          string  `json:"status"`
	MetaTitle       string  `json:"meta_title"`
	MetaDescription string  `json:"meta_description"`
}

// GetNews returns all news for the current user's organization/club
func GetNews(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		var news []News
		var err error

		// Build query based on user type
		if userType == "organization" {
			err = db.Select(&news, `
				SELECT uuid, organization_id, club_id, title, slug, excerpt, image_url, 
				       category, status, views, author_name, published_at, created_at, updated_at
				FROM news 
				WHERE organization_id = (SELECT uuid FROM organizations WHERE uuid = ?)
				ORDER BY created_at DESC
			`, userID)
		} else if userType == "club" {
			err = db.Select(&news, `
				SELECT uuid, organization_id, club_id, title, slug, excerpt, image_url, 
				       category, status, views, author_name, published_at, created_at, updated_at
				FROM news 
				WHERE club_id = (SELECT uuid FROM clubs WHERE uuid = ?)
				ORDER BY created_at DESC
			`, userID)
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to view news"})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch news: " + err.Error()})
			return
		}

		if news == nil {
			news = []News{}
		}

		c.JSON(http.StatusOK, gin.H{"data": news})
	}
}

// GetNewsPublic returns published news (for public pages)
func GetNewsPublic(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var news []News

		err := db.Select(&news, `
			SELECT uuid, organization_id, club_id, title, slug, excerpt, image_url, 
			       category, status, views, author_name, published_at, created_at
			FROM news 
			WHERE status = 'published'
			ORDER BY published_at DESC
			LIMIT 20
		`)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch news"})
			return
		}

		if news == nil {
			news = []News{}
		}

		c.JSON(http.StatusOK, gin.H{"data": news})
	}
}

// GetNewsByID returns a single news article
func GetNewsByID(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var article News
		err := db.Get(&article, `
			SELECT uuid, organization_id, club_id, title, slug, excerpt, content, image_url, 
			       category, status, views, author_name, author_id, meta_title, meta_description,
			       published_at, created_at, updated_at
			FROM news 
			WHERE uuid = ? OR slug = ?
		`, id, id)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "News not found"})
			return
		}

		// Mask URL
		if article.ImageURL != nil {
			masked := utils.MaskMediaURL(*article.ImageURL)
			article.ImageURL = &masked
		}

		c.JSON(http.StatusOK, gin.H{"data": article})
	}
}

// CreateNews creates a new news article
func CreateNews(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		var req CreateNewsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		newsID := uuid.New().String()
		slug := generateSlug(req.Title)

		// Get author name
		var authorName string
		if userType == "organization" {
			db.Get(&authorName, "SELECT name FROM organizations WHERE uuid = ?", userID)
		} else if userType == "club" {
			db.Get(&authorName, "SELECT name FROM clubs WHERE uuid = ?", userID)
		}

		// Set default values
		if req.Category == "" {
			req.Category = "pengumuman"
		}
		if req.Status == "" {
			req.Status = "draft"
		}

		// Determine which ID to use
		var orgID, clubID *string
		userIDStr := userID.(string)
		if userType == "organization" {
			orgID = &userIDStr
		} else if userType == "club" {
			clubID = &userIDStr
		}

		var publishedAt *time.Time
		if req.Status == "published" {
			now := time.Now()
			publishedAt = &now
		}

		_, err := db.Exec(`
			INSERT INTO news (uuid, organization_id, club_id, title, slug, excerpt, content, image_url, 
			                  category, status, author_name, author_id, meta_title, meta_description, published_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, newsID, orgID, clubID, req.Title, slug, req.Excerpt, req.Content, utils.ExtractFilename(req.ImageURL),
			req.Category, req.Status, authorName, userID, req.MetaTitle, req.MetaDescription, publishedAt)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create news: " + err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "News created successfully",
			"id":      newsID,
		})
	}
}

// UpdateNews updates a news article
func UpdateNews(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		var req CreateNewsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verify ownership
		var ownerID string
		if userType == "organization" {
			db.Get(&ownerID, "SELECT organization_id FROM news WHERE uuid = ?", id)
		} else if userType == "club" {
			db.Get(&ownerID, "SELECT club_id FROM news WHERE uuid = ?", id)
		}

		if ownerID != userID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to update this news"})
			return
		}

		// Check if status changed to published
		var currentStatus string
		db.Get(&currentStatus, "SELECT status FROM news WHERE uuid = ?", id)

		publishedAtUpdate := ""
		if currentStatus != "published" && req.Status == "published" {
			publishedAtUpdate = ", published_at = NOW()"
		}

		_, err := db.Exec(`
			UPDATE news SET 
				title = ?, excerpt = ?, content = ?, image_url = ?, 
				category = ?, status = ?, meta_title = ?, meta_description = ?`+publishedAtUpdate+`
			WHERE uuid = ?
		`, req.Title, req.Excerpt, req.Content, utils.ExtractFilename(req.ImageURL),
			req.Category, req.Status, req.MetaTitle, req.MetaDescription, id)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update news: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "News updated successfully"})
	}
}

// DeleteNews deletes a news article
func DeleteNews(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		// Verify ownership
		var ownerID string
		if userType == "organization" {
			db.Get(&ownerID, "SELECT organization_id FROM news WHERE uuid = ?", id)
		} else if userType == "club" {
			db.Get(&ownerID, "SELECT club_id FROM news WHERE uuid = ?", id)
		}

		if ownerID != userID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this news"})
			return
		}

		_, err := db.Exec("DELETE FROM news WHERE uuid = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete news"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "News deleted successfully"})
	}
}

// generateSlug creates a URL-friendly slug from title
func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove special characters (simple approach)
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	// Add timestamp suffix for uniqueness
	return result.String() + "-" + time.Now().Format("20060102")
}
