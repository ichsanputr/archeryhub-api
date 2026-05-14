package handler

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"Archeris-api/utils"
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
	Tags            *string `db:"tags" json:"tags,omitempty"`
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
	Tags            string  `json:"tags"`
	Status          string  `json:"status"`
	MetaTitle       string  `json:"meta_title"`
	MetaDescription string  `json:"meta_description"`
}

// GetNews returns all news for the current user's organization/club
func GetNews(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		limit, offset, page := utils.GetPaginationParams(c)
		var news []News
		var totalCount int
		var err error

		whereClause := ""
		if userType == "organization" {
			whereClause = "WHERE organization_id = (SELECT uuid FROM organizations WHERE uuid = ?)"
		} else if userType == "club" {
			whereClause = "WHERE club_id = (SELECT uuid FROM clubs WHERE uuid = ?)"
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak diizinkan untuk melihat berita"})
			return
		}

		// Count total
		err = db.Get(&totalCount, "SELECT COUNT(*) FROM news "+whereClause, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung berita"})
			return
		}

		// Get data
		query := fmt.Sprintf(`
			SELECT uuid, organization_id, club_id, title, slug, excerpt, image_url, 
				   category, tags, status, views, author_name, published_at, created_at, updated_at
			FROM news 
			%s
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?
		`, whereClause)

		err = db.Select(&news, query, userID, limit, offset)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data berita: " + err.Error()})
			return
		}

		if news == nil {
			news = []News{}
		}

		// Mask URLs
		for i := range news {
			if news[i].ImageURL != nil {
				masked := utils.MaskMediaURL(*news[i].ImageURL)
				news[i].ImageURL = &masked
			}
		}

		meta := utils.CalculatePagination(totalCount, limit, offset, page)
		c.JSON(http.StatusOK, gin.H{"data": news, "meta": meta})
	}
}

// GetNewsPublic returns published news (for public pages)
func GetNewsPublic(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, offset, page := utils.GetPaginationParams(c)
		category := c.Query("category")
		search := c.Query("search")

		whereClause := "WHERE status = 'published'"
		args := []interface{}{}

		if category != "" && category != "all" {
			whereClause += " AND category = ?"
			args = append(args, category)
		}
		if search != "" {
			whereClause += " AND (title LIKE ? OR content LIKE ?)"
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm)
		}

		// Count total
		var totalCount int
		err := db.Get(&totalCount, "SELECT COUNT(*) FROM news "+whereClause, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung data berita"})
			return
		}

		var news []News
		query := fmt.Sprintf(`
			SELECT uuid, organization_id, club_id, title, slug, excerpt, image_url, 
				   category, tags, status, views, author_name, published_at, created_at
			FROM news 
			%s
			ORDER BY published_at DESC
			LIMIT ? OFFSET ?
		`, whereClause)
		queryArgs := append(args, limit, offset)

		err = db.Select(&news, query, queryArgs...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data berita"})
			return
		}

		if news == nil {
			news = []News{}
		}

		// Mask URLs
		for i := range news {
			if news[i].ImageURL != nil {
				masked := utils.MaskMediaURL(*news[i].ImageURL)
				news[i].ImageURL = &masked
			}
		}

		meta := utils.CalculatePagination(totalCount, limit, offset, page)
		c.JSON(http.StatusOK, gin.H{"data": news, "meta": meta})
	}
}

// GetNewsByID returns a single news article
func GetNewsByID(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var article News
		err := db.Get(&article, `
			SELECT uuid, organization_id, club_id, title, slug, excerpt, content, image_url, 
			       category, tags, status, views, author_name, author_id, meta_title, meta_description,
			       published_at, created_at, updated_at
			FROM news 
			WHERE uuid = ? OR slug = ?
		`, id, id)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Berita tidak ditemukan"})
			return
		}

		// Increment views
		db.Exec("UPDATE news SET views = COALESCE(views, 0) + 1 WHERE uuid = ?", article.UUID)

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
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hanya organisasi dan klub yang bisa memposting berita"})
			return
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
			                  category, tags, status, author_name, author_id, meta_title, meta_description, published_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, newsID, orgID, clubID, req.Title, slug, req.Excerpt, req.Content, utils.ExtractFilename(req.ImageURL),
			req.Category, req.Tags, req.Status, authorName, userID, req.MetaTitle, req.MetaDescription, publishedAt)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat berita: " + err.Error()})
			return
		}

		// Notify subscribers if published
		if req.Status == "published" {
			article := News{
				Title:   req.Title,
				Slug:    slug,
				Excerpt: &req.Excerpt,
			}
			notifySubscribers(db, article)
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Berita berhasil dibuat",
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

		// Verify ownership and get UUID
		var article struct {
			UUID           string  `db:"uuid"`
			OrganizationID *string `db:"organization_id"`
			ClubID         *string `db:"club_id"`
			Status         string  `db:"status"`
		}

		err := db.Get(&article, "SELECT uuid, organization_id, club_id, status FROM news WHERE uuid = ? OR slug = ?", id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Berita tidak ditemukan"})
			return
		}

		isOwner := false
		if userType == "organization" && article.OrganizationID != nil && *article.OrganizationID == userID.(string) {
			isOwner = true
		} else if userType == "club" && article.ClubID != nil && *article.ClubID == userID.(string) {
			isOwner = true
		}

		if !isOwner {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak diizinkan untuk mengubah berita ini"})
			return
		}

		// Check if status changed to published
		publishedAtUpdate := ""
		if article.Status != "published" && req.Status == "published" {
			publishedAtUpdate = ", published_at = NOW()"
		}

		_, err = db.Exec(`
			UPDATE news SET 
				title = ?, excerpt = ?, content = ?, image_url = ?, 
				category = ?, tags = ?, status = ?, meta_title = ?, meta_description = ?`+publishedAtUpdate+`
			WHERE uuid = ?
		`, req.Title, req.Excerpt, req.Content, utils.ExtractFilename(req.ImageURL),
			req.Category, req.Tags, req.Status, req.MetaTitle, req.MetaDescription, article.UUID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui berita: " + err.Error()})
			return
		}

		// Notify subscribers if status changed to published
		if article.Status != "published" && req.Status == "published" {
			var slug string
			err := db.Get(&slug, "SELECT slug FROM news WHERE uuid = ?", article.UUID)
			if err == nil {
				notifySubscribers(db, News{
					Title:   req.Title,
					Slug:    slug,
					Excerpt: &req.Excerpt,
				})
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Berita berhasil diperbarui"})
	}
}

// DeleteNews deletes a news article
func DeleteNews(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		// Verify ownership
		var article struct {
			UUID           string  `db:"uuid"`
			OrganizationID *string `db:"organization_id"`
			ClubID         *string `db:"club_id"`
		}

		err := db.Get(&article, "SELECT uuid, organization_id, club_id FROM news WHERE uuid = ? OR slug = ?", id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Berita tidak ditemukan"})
			return
		}

		isOwner := false
		if userType == "organization" && article.OrganizationID != nil && *article.OrganizationID == userID.(string) {
			isOwner = true
		} else if userType == "club" && article.ClubID != nil && *article.ClubID == userID.(string) {
			isOwner = true
		}

		if !isOwner {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak diizinkan untuk menghapus berita ini"})
			return
		}

		_, err = db.Exec("DELETE FROM news WHERE uuid = ?", article.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus berita"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Berita berhasil dihapus"})
	}
}

// SubscribeNews handles new email subscriptions
func SubscribeNews(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email string `json:"email" binding:"required,email"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Alamat email tidak valid"})
			return
		}

		_, err := db.Exec(`
			INSERT INTO news_subscribers (email) 
			VALUES (?) 
			ON DUPLICATE KEY UPDATE is_active = 1
		`, req.Email)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal berlangganan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Terima kasih telah berlangganan!"})
	}
}

func notifySubscribers(db *sqlx.DB, article News) {
	var subscribers []string
	err := db.Select(&subscribers, "SELECT email FROM news_subscribers WHERE is_active = 1")
	if err != nil {
		return
	}

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "https://archeris.net"
	}

	subject := "Berita Baru: " + article.Title
	
	excerpt := ""
	if article.Excerpt != nil {
		excerpt = *article.Excerpt
	}

	body := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #eee; border-radius: 10px;">
			<h2 style="color: #333;">%s</h2>
			<p style="color: #666; line-height: 1.6;">%s</p>
			<div style="margin-top: 30px;">
				<a href="%s/news/%s" 
				   style="background-color: #007bff; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; font-weight: bold;">
				   Baca Selengkapnya
				</a>
			</div>
			<hr style="margin-top: 40px; border: 0; border-top: 1px solid #eee;">
			<p style="font-size: 12px; color: #999;">
				Anda menerima email ini karena Anda berlangganan berita di archeris.net.
			</p>
		</div>
	`, article.Title, excerpt, appURL, article.Slug)

	for _, email := range subscribers {
		go func(to string) {
			utils.SendEmail(to, subject, body)
		}(email)
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

// IncrementNewsViews increments the view count of a news article
func IncrementNewsViews(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		_, err := db.Exec("UPDATE news SET views = COALESCE(views, 0) + 1 WHERE uuid = ? OR slug = ?", id, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui data analitik"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Analitik diperbarui"})
	}
}


