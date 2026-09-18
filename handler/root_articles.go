package handler

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// RootArticleRequest defines the payload for creating or updating an article
type RootArticleRequest struct {
	Title        string   `json:"title" binding:"required"`
	Slug         string   `json:"slug"`
	Excerpt      string   `json:"excerpt"`
	Content      string   `json:"content" binding:"required"`
	Category     string   `json:"category" binding:"required"`
	ImageURL     string   `json:"image_url"`
	AuthorName   string   `json:"author_name"`
	AuthorRole   string   `json:"author_role"`
	AuthorAvatar string   `json:"author_avatar"`
	Tags         []string `json:"tags"`
	ReadTime     int      `json:"read_time"`
	Status       string   `json:"status"` // draft, published, archived
	PublishedAt  *string  `json:"published_at"`
}

// generateSlug creates a clean URL slug from title
func generateSlug(title string) string {
	slug := strings.ToLower(title)
	// Replace non-alphanumeric characters with hyphen
	reg := regexp.MustCompile("[^a-z0-9]+")
	slug = reg.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = fmt.Sprintf("article-%d", time.Now().Unix())
	}
	return slug
}

// calculateReadTime calculates read time in minutes from word count
func calculateReadTime(content string) int {
	words := strings.Fields(content)
	wordCount := len(words)
	minutes := int(math.Ceil(float64(wordCount) / 200.0))
	if minutes < 1 {
		return 1
	}
	return minutes
}

// RootListArticles returns all articles for root management with filtering and pagination
func RootListArticles(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 20
		}
		offset := (page - 1) * limit

		search := strings.TrimSpace(c.Query("q"))
		category := strings.TrimSpace(c.Query("category"))
		status := strings.TrimSpace(c.Query("status"))
		sortBy := c.DefaultQuery("sort_by", "created_at")
		order := strings.ToUpper(c.DefaultQuery("order", "DESC"))
		if order != "ASC" && order != "DESC" {
			order = "DESC"
		}

		// Validate sort column
		allowedSorts := map[string]string{
			"id":           "id",
			"title":        "title",
			"category":     "category",
			"status":       "status",
			"views":        "views",
			"published_at": "published_at",
			"created_at":   "created_at",
			"updated_at":   "updated_at",
		}
		sortColumn, ok := allowedSorts[sortBy]
		if !ok {
			sortColumn = "created_at"
		}

		baseQuery := "FROM blog_articles WHERE 1=1"
		var args []interface{}

		if search != "" {
			baseQuery += " AND (title LIKE ? OR excerpt LIKE ? OR slug LIKE ?)"
			term := "%" + search + "%"
			args = append(args, term, term, term)
		}

		if category != "" && category != "All" {
			baseQuery += " AND category = ?"
			args = append(args, category)
		}

		if status != "" && status != "All" {
			baseQuery += " AND status = ?"
			args = append(args, status)
		}

		// Count total matching
		var total int
		countQuery := "SELECT COUNT(*) " + baseQuery
		if err := db.Get(&total, countQuery, args...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung total artikel", "details": err.Error()})
			return
		}

		// Fetch paginated data
		selectQuery := fmt.Sprintf(`
			SELECT id, uuid, slug, title, excerpt, content, category, image_url,
			       author_name, author_role, author_avatar, tags, image_prompts, read_time, views, status,
			       published_at, created_at, updated_at
			%s
			ORDER BY %s %s
			LIMIT ? OFFSET ?
		`, baseQuery, sortColumn, order)

		queryArgs := append(args, limit, offset)

		var dbArticles []BlogArticleDB
		if err := db.Select(&dbArticles, selectQuery, queryArgs...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar artikel", "details": err.Error()})
			return
		}

		articles := make([]BlogArticle, len(dbArticles))
		for i, a := range dbArticles {
			articles[i] = transformArticle(a)
		}

		// Stats
		var stats struct {
			Total     int `db:"total"`
			Published int `db:"published"`
			Draft     int `db:"draft"`
			Archived  int `db:"archived"`
			TotalViews int `db:"total_views"`
		}
		_ = db.Get(&stats, `
			SELECT 
				COUNT(*) as total,
				COALESCE(SUM(CASE WHEN status = 'published' THEN 1 ELSE 0 END), 0) as published,
				COALESCE(SUM(CASE WHEN status = 'draft' THEN 1 ELSE 0 END), 0) as draft,
				COALESCE(SUM(CASE WHEN status = 'archived' THEN 1 ELSE 0 END), 0) as archived,
				COALESCE(SUM(views), 0) as total_views
			FROM blog_articles
		`)

		c.JSON(http.StatusOK, gin.H{
			"data":        articles,
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": int(math.Ceil(float64(total) / float64(limit))),
			"stats": gin.H{
				"total":       stats.Total,
				"published":   stats.Published,
				"draft":       stats.Draft,
				"archived":    stats.Archived,
				"total_views": stats.TotalViews,
			},
		})
	}
}

// RootGetArticle returns a single article by ID or UUID
func RootGetArticle(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idOrUUID := c.Param("id")

		var dbArticle BlogArticleDB
		query := `
			SELECT id, uuid, slug, title, excerpt, content, category, image_url,
			       author_name, author_role, author_avatar, tags, image_prompts, read_time, views, status,
			       published_at, created_at, updated_at
			FROM blog_articles
			WHERE uuid = ? OR CAST(id AS CHAR) = ? OR slug = ?
			LIMIT 1
		`
		err := db.Get(&dbArticle, query, idOrUUID, idOrUUID, idOrUUID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Artikel tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": transformArticle(dbArticle),
		})
	}
}

// RootCreateArticle creates a new blog article
func RootCreateArticle(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RootArticleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data artikel tidak lengkap", "details": err.Error()})
			return
		}

		req.Title = strings.TrimSpace(req.Title)
		if req.Title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Judul artikel wajib diisi"})
			return
		}

		// Slug handling
		slug := strings.TrimSpace(req.Slug)
		if slug == "" {
			slug = generateSlug(req.Title)
		} else {
			slug = generateSlug(slug)
		}

		// Ensure unique slug
		originalSlug := slug
		counter := 1
		for {
			var count int
			err := db.Get(&count, "SELECT COUNT(*) FROM blog_articles WHERE slug = ?", slug)
			if err != nil || count == 0 {
				break
			}
			slug = fmt.Sprintf("%s-%d", originalSlug, counter)
			counter++
		}

		articleUUID := uuid.New().String()

		authorName := strings.TrimSpace(req.AuthorName)
		if authorName == "" {
			authorName = "Archeris Editorial"
		}
		authorRole := strings.TrimSpace(req.AuthorRole)
		if authorRole == "" {
			authorRole = "Archery Specialist & Coach"
		}
		authorAvatar := strings.TrimSpace(req.AuthorAvatar)
		if authorAvatar == "" {
			authorAvatar = "/profile-author.png"
		}

		status := strings.ToLower(strings.TrimSpace(req.Status))
		if status != "draft" && status != "published" && status != "archived" {
			status = "published"
		}

		readTime := req.ReadTime
		if readTime <= 0 {
			readTime = calculateReadTime(req.Content)
		}

		var tagsStr *string
		if len(req.Tags) > 0 {
			cleanedTags := make([]string, 0, len(req.Tags))
			for _, t := range req.Tags {
				t = strings.TrimSpace(t)
				if t != "" {
					cleanedTags = append(cleanedTags, t)
				}
			}
			if len(cleanedTags) > 0 {
				val := strings.Join(cleanedTags, ", ")
				tagsStr = &val
			}
		}

		excerpt := strings.TrimSpace(req.Excerpt)
		if excerpt == "" {
			// Extract plain text snippet from content
			clean := regexp.MustCompile("<[^>]*>").ReplaceAllString(req.Content, " ")
			clean = strings.Join(strings.Fields(clean), " ")
			if len(clean) > 160 {
				excerpt = clean[:157] + "..."
			} else {
				excerpt = clean
			}
		}

		publishedAt := time.Now().Format("2006-01-02 15:04:05")
		if req.PublishedAt != nil && *req.PublishedAt != "" {
			publishedAt = *req.PublishedAt
		}

		insertQuery := `
			INSERT INTO blog_articles (
				uuid, slug, title, excerpt, content, category, image_url,
				author_name, author_role, author_avatar, tags, read_time, views, status,
				published_at, created_at, updated_at
			) VALUES (
				?, ?, ?, ?, ?, ?, ?,
				?, ?, ?, ?, ?, 0, ?,
				?, NOW(), NOW()
			)
		`

		_, err := db.Exec(insertQuery,
			articleUUID, slug, req.Title, excerpt, req.Content, req.Category, req.ImageURL,
			authorName, authorRole, authorAvatar, tagsStr, readTime, status,
			publishedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan artikel baru", "details": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Artikel berhasil dibuat",
			"uuid":    articleUUID,
			"slug":    slug,
		})
	}
}

// RootUpdateArticle updates an existing article
func RootUpdateArticle(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idOrUUID := c.Param("id")

		var req RootArticleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data artikel tidak valid", "details": err.Error()})
			return
		}

		var existing struct {
			ID   int    `db:"id"`
			UUID string `db:"uuid"`
			Slug string `db:"slug"`
		}
		err := db.Get(&existing, "SELECT id, uuid, slug FROM blog_articles WHERE uuid = ? OR CAST(id AS CHAR) = ? OR slug = ? LIMIT 1", idOrUUID, idOrUUID, idOrUUID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Artikel tidak ditemukan"})
			return
		}

		slug := strings.TrimSpace(req.Slug)
		if slug == "" {
			slug = generateSlug(req.Title)
		} else {
			slug = generateSlug(slug)
		}

		// Ensure unique slug excluding current article
		if slug != existing.Slug {
			originalSlug := slug
			counter := 1
			for {
				var count int
				err := db.Get(&count, "SELECT COUNT(*) FROM blog_articles WHERE slug = ? AND id != ?", slug, existing.ID)
				if err != nil || count == 0 {
					break
				}
				slug = fmt.Sprintf("%s-%d", originalSlug, counter)
				counter++
			}
		}

		authorName := strings.TrimSpace(req.AuthorName)
		if authorName == "" {
			authorName = "Archeris Editorial"
		}
		authorRole := strings.TrimSpace(req.AuthorRole)
		if authorRole == "" {
			authorRole = "Archery Specialist & Coach"
		}
		authorAvatar := strings.TrimSpace(req.AuthorAvatar)
		if authorAvatar == "" {
			authorAvatar = "/profile-author.png"
		}

		status := strings.ToLower(strings.TrimSpace(req.Status))
		if status != "draft" && status != "published" && status != "archived" {
			status = "published"
		}

		readTime := req.ReadTime
		if readTime <= 0 {
			readTime = calculateReadTime(req.Content)
		}

		var tagsStr *string
		if len(req.Tags) > 0 {
			cleanedTags := make([]string, 0, len(req.Tags))
			for _, t := range req.Tags {
				t = strings.TrimSpace(t)
				if t != "" {
					cleanedTags = append(cleanedTags, t)
				}
			}
			if len(cleanedTags) > 0 {
				val := strings.Join(cleanedTags, ", ")
				tagsStr = &val
			}
		}

		excerpt := strings.TrimSpace(req.Excerpt)
		if excerpt == "" {
			clean := regexp.MustCompile("<[^>]*>").ReplaceAllString(req.Content, " ")
			clean = strings.Join(strings.Fields(clean), " ")
			if len(clean) > 160 {
				excerpt = clean[:157] + "..."
			} else {
				excerpt = clean
			}
		}

		updateQuery := `
			UPDATE blog_articles
			SET slug = ?, title = ?, excerpt = ?, content = ?, category = ?, image_url = ?,
			    author_name = ?, author_role = ?, author_avatar = ?, tags = ?, read_time = ?,
			    status = ?, updated_at = NOW()
		`
		var updateArgs []interface{}
		updateArgs = append(updateArgs, slug, req.Title, excerpt, req.Content, req.Category, req.ImageURL,
			authorName, authorRole, authorAvatar, tagsStr, readTime, status)

		if req.PublishedAt != nil && *req.PublishedAt != "" {
			updateQuery += ", published_at = ?"
			updateArgs = append(updateArgs, *req.PublishedAt)
		}

		updateQuery += " WHERE id = ?"
		updateArgs = append(updateArgs, existing.ID)

		_, err = db.Exec(updateQuery, updateArgs...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui artikel", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Artikel berhasil diperbarui",
			"uuid":    existing.UUID,
			"slug":    slug,
		})
	}
}

// RootDeleteArticle permanently removes an article
func RootDeleteArticle(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idOrUUID := c.Param("id")

		result, err := db.Exec("DELETE FROM blog_articles WHERE uuid = ? OR CAST(id AS CHAR) = ? OR slug = ?", idOrUUID, idOrUUID, idOrUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus artikel", "details": err.Error()})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Artikel tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Artikel berhasil dihapus"})
	}
}

// RootToggleArticleStatus changes the status of an article (draft <-> published <-> archived)
func RootToggleArticleStatus(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idOrUUID := c.Param("id")

		var req struct {
			Status string `json:"status" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Status wajib ditentukan"})
			return
		}

		status := strings.ToLower(strings.TrimSpace(req.Status))
		if status != "draft" && status != "published" && status != "archived" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Status harus berupa draft, published, atau archived"})
			return
		}

		result, err := db.Exec("UPDATE blog_articles SET status = ?, updated_at = NOW() WHERE uuid = ? OR CAST(id AS CHAR) = ? OR slug = ?", status, idOrUUID, idOrUUID, idOrUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengubah status artikel", "details": err.Error()})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Artikel tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Status artikel berhasil diubah",
			"status":  status,
		})
	}
}

// RootUploadArticleImage handles image upload for articles
func RootUploadArticleImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File gambar tidak ditemukan"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" && ext != ".gif" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format gambar harus berupa JPG, PNG, WEBP, atau GIF"})
		return
	}

	// 5MB limit
	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ukuran gambar maksimal 5MB"})
		return
	}

	uploadDir := "public/uploads/articles"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat direktori upload"})
		return
	}

	newFilename := fmt.Sprintf("article_%d_%s%s", time.Now().Unix(), uuid.New().String()[:8], ext)
	dstPath := filepath.Join(uploadDir, newFilename)

	dst, err := os.Create(dstPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan file"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menulis file"})
		return
	}

	imageURL := "/uploads/articles/" + newFilename
	c.JSON(http.StatusOK, gin.H{
		"message":   "Gambar berhasil diunggah",
		"image_url": imageURL,
		"url":       imageURL,
	})
}
