package mobile

import (
	"archeryhub-api/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type MobileNewsComment struct {
	UUID      string  `db:"uuid" json:"id"`
	NewsID    string  `db:"news_id" json:"news_id"`
	UserID    *string `db:"user_id" json:"user_id,omitempty"`
	UserType  string  `db:"user_type" json:"user_type"`
	GuestName *string `db:"guest_name" json:"guest_name,omitempty"`
	Content   string  `db:"content" json:"content"`
	CreatedAt string  `db:"created_at" json:"created_at"`
}

type MobileNewsItem struct {
	UUID        string  `db:"uuid" json:"id"`
	Title       string  `db:"title" json:"title"`
	Slug        string  `db:"slug" json:"slug"`
	Excerpt     *string `db:"excerpt" json:"excerpt,omitempty"`
	ImageURL    *string `db:"image_url" json:"image_url,omitempty"`
	Category    string  `db:"category" json:"category"`
	AuthorName  *string `db:"author_name" json:"author_name,omitempty"`
	Views       int     `db:"views" json:"views"`
	PublishedAt *string `db:"published_at" json:"published_at,omitempty"`
	CreatedAt   string  `db:"created_at" json:"created_at"`
}

type MobileNewsDetail struct {
	UUID            string  `db:"uuid" json:"id"`
	Title           string  `db:"title" json:"title"`
	Slug            string  `db:"slug" json:"slug"`
	Excerpt         *string `db:"excerpt" json:"excerpt,omitempty"`
	Content         *string `db:"content" json:"content,omitempty"`
	ImageURL        *string `db:"image_url" json:"image_url,omitempty"`
	Category        string  `db:"category" json:"category"`
	Tags            *string `db:"tags" json:"tags,omitempty"`
	AuthorName      *string `db:"author_name" json:"author_name,omitempty"`
	MetaTitle       *string `db:"meta_title" json:"meta_title,omitempty"`
	MetaDescription *string `db:"meta_description" json:"meta_description,omitempty"`
	Views           int     `db:"views" json:"views"`
	PublishedAt     *string `db:"published_at" json:"published_at,omitempty"`
	CreatedAt       string  `db:"created_at" json:"created_at"`
	UpdatedAt       string  `db:"updated_at" json:"updated_at"`
}

// MobileListNews godoc
// @Summary      List published news for mobile
// @Tags         News
// @Produce      json
// @Param        limit   query     int     false  "Limit"
// @Param        offset  query     int     false  "Offset"
// @Param        search  query     string  false  "Search term"
// @Success      200     {object}  map[string]interface{}
// @Router       /news [get]
func MobileListNews(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		search := c.Query("search")

		whereClause := "WHERE status = 'published'"
		args := []interface{}{}
		if search != "" {
			whereClause += " AND (title LIKE ? OR excerpt LIKE ? OR content LIKE ?)"
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm, searchTerm)
		}

		query := `
			SELECT uuid, title, slug, excerpt, image_url, category, author_name,
			       COALESCE(views, 0) as views, published_at, created_at
			FROM news
			` + whereClause + `
			ORDER BY published_at DESC, created_at DESC
			LIMIT ? OFFSET ?
		`
		args = append(args, limit, offset)

		var news []MobileNewsItem
		if err := db.Select(&news, query, args...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data berita", "details": err.Error()})
			return
		}

		if news == nil {
			news = []MobileNewsItem{}
		}

		for i := range news {
			if news[i].ImageURL != nil {
				masked := utils.MaskMediaURL(*news[i].ImageURL)
				news[i].ImageURL = &masked
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"news":        news,
			"total_count": len(news),
		})
	}
}

// MobileGetNewsDetail godoc
// @Summary      Get published news detail for mobile
// @Tags         News
// @Produce      json
// @Param        id  path      string  true  "News UUID or slug"
// @Success      200 {object}  map[string]interface{}
// @Router       /news/{id} [get]
func MobileGetNewsDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var article MobileNewsDetail
		err := db.Get(&article, `
			SELECT uuid, title, slug, excerpt, content, image_url, category, tags,
			       author_name, meta_title, meta_description, COALESCE(views, 0) as views,
			       published_at, created_at, updated_at
			FROM news
			WHERE status = 'published' AND (uuid = ? OR slug = ?)
			LIMIT 1
		`, id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Berita tidak ditemukan"})
			return
		}

		_, _ = db.Exec("UPDATE news SET views = COALESCE(views, 0) + 1 WHERE uuid = ?", article.UUID)

		if article.ImageURL != nil {
			masked := utils.MaskMediaURL(*article.ImageURL)
			article.ImageURL = &masked
		}

		c.JSON(http.StatusOK, gin.H{"news": article})
	}
}

// MobileListNewsComments godoc
// @Summary      List comments for a news article
// @Tags         News
// @Produce      json
// @Param        id   path      string  true  "News UUID or slug"
// @Success      200  {object}  map[string]interface{}
// @Router       /news/{id}/comments [get]
func MobileListNewsComments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var comments []MobileNewsComment
		query := `
			SELECT c.uuid, c.news_id, c.user_id, c.user_type, c.guest_name, c.content, c.created_at
			FROM news_comments c
			JOIN news n ON c.news_id = n.uuid
			WHERE (n.uuid = ? OR n.slug = ?) AND c.status = 'approved'
			ORDER BY c.created_at DESC
		`
		err := db.Select(&comments, query, id, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil komentar", "details": err.Error()})
			return
		}

		if comments == nil {
			comments = []MobileNewsComment{}
		}

		c.JSON(http.StatusOK, gin.H{
			"comments": comments,
			"count":    len(comments),
		})
	}
}

// MobileAddNewsComment godoc
// @Summary      Post a comment on a news article
// @Tags         News
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "News UUID or slug"
// @Param        request  body      object{guest_name=string,content=string}  true  "Comment payload"
// @Success      201      {object}  map[string]interface{}
// @Router       /news/{id}/comments [post]
func MobileAddNewsComment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		
		var req struct {
			GuestName string `json:"guest_name"`
			Content   string `json:"content" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data komentar tidak valid"})
			return
		}

		// Resolve news ID
		var newsID string
		err := db.Get(&newsID, "SELECT uuid FROM news WHERE uuid = ? OR slug = ?", id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Berita tidak ditemukan"})
			return
		}

		userIDInterface, exists := c.Get("user_id")
		userTypeInterface, _ := c.Get("user_type")

		var userID *string
		userType := "guest"
		guestName := &req.GuestName

		if exists && userIDInterface != nil {
			uid := userIDInterface.(string)
			userID = &uid
			userType = userTypeInterface.(string)
			guestName = nil
		} else {
			if req.GuestName == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Nama harus diisi untuk komentar tamu"})
				return
			}
		}

		commentUUID := uuid.New().String()
		_, err = db.Exec(`
			INSERT INTO news_comments (uuid, news_id, user_id, user_type, guest_name, content, status)
			VALUES (?, ?, ?, ?, ?, ?, 'approved')
		`, commentUUID, newsID, userID, userType, guestName, req.Content)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan komentar", "details": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Komentar berhasil ditambahkan"})
	}
}

// MobileListRelatedNews godoc
// @Summary      List related news
// @Description  Returns a list of related news articles based on the same category
// @Tags         News
// @Produce      json
// @Param        id   path      string  true  "News UUID or slug"
// @Success      200  {object}  map[string]interface{}
// @Router       /news/{id}/related [get]
func MobileListRelatedNews(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// 1. Get current news category
		var current struct {
			UUID     string `db:"uuid"`
			Category string `db:"category"`
		}
		err := db.Get(&current, "SELECT uuid, category FROM news WHERE uuid = ? OR slug = ? LIMIT 1", id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Berita tidak ditemukan"})
			return
		}

		// 2. Get related news (same category, different UUID)
		var news []MobileNewsItem
		query := `
			SELECT uuid, title, slug, excerpt, image_url, category, author_name,
			       COALESCE(views, 0) as views, published_at, created_at
			FROM news
			WHERE status = 'published' AND category = ? AND uuid != ?
			ORDER BY published_at DESC, created_at DESC
			LIMIT 5
		`
		if err := db.Select(&news, query, current.Category, current.UUID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil berita terkait"})
			return
		}

		if news == nil {
			news = []MobileNewsItem{}
		}

		for i := range news {
			if news[i].ImageURL != nil {
				masked := utils.MaskMediaURL(*news[i].ImageURL)
				news[i].ImageURL = &masked
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"news": news,
		})
	}
}
