package mobile

import (
	"archeryhub-api/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// MobileListNews godoc
// @Summary      List published news for mobile
// @Tags         News
// @Produce      json
// @Param        limit   query     int     false  "Limit"
// @Param        offset  query     int     false  "Offset"
// @Param        search  query     string  false  "Search term"
// @Success      200     {object}  MobileNewsListResponse
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

		c.JSON(http.StatusOK, MobileNewsListResponse{
			News:       news,
			TotalCount: len(news),
		})
	}
}

// MobileGetNewsDetail godoc
// @Summary      Get published news detail for mobile
// @Tags         News
// @Produce      json
// @Param        id  path      string  true  "News UUID or slug"
// @Success      200 {object}  MobileNewsDetailResponse
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

		c.JSON(http.StatusOK, MobileNewsDetailResponse{News: article})
	}
}

// MobileListNewsComments godoc
// @Summary      List comments for a news article
// @Tags         News
// @Produce      json
// @Param        id   path      string  true  "News UUID or slug"
// @Success      200  {object}  MobileNewsCommentsResponse
// @Router       /news/{id}/comments [get]
func MobileListNewsComments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var comments []MobileNewsComment
		query := `
			SELECT c.uuid, c.news_id, c.user_id, c.user_type, c.guest_name, c.content, c.created_at,
				CASE 
					WHEN c.user_type = 'archer' THEN (SELECT full_name FROM archers WHERE uuid = c.user_id)
					WHEN c.user_type = 'organization' THEN (SELECT name FROM organizations WHERE uuid = c.user_id)
					WHEN c.user_type = 'seller' THEN (SELECT store_name FROM sellers WHERE uuid = c.user_id)
					ELSE c.guest_name
				END as user_name
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

		c.JSON(http.StatusOK, MobileNewsCommentsResponse{
			Comments: comments,
			Count:    len(comments),
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
// @Success      201      {object}  MessageResponse
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
// @Success      200  {object}  MobileRelatedNewsResponse
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

		c.JSON(http.StatusOK, MobileRelatedNewsResponse{
			News: news,
		})
	}
}
