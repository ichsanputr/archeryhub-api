package mobile

import (
	"archeryhub-api/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

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
// @Description  Returns published news list for mobile read-only pages
// @Tags         News
// @Produce      json
// @Param        limit   query     int     false  "Limit"
// @Param        offset  query     int     false  "Offset"
// @Param        search  query     string  false  "Search term"
// @Success      200     {object}  MobileNewsListResponse
// @Failure      500     {object}  ErrorResponse
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
// @Description  Returns one published news article by UUID or slug
// @Tags         News
// @Produce      json
// @Param        id  path      string  true  "News UUID or slug"
// @Success      200 {object}  MobileNewsDetailResponse
// @Failure      404 {object}  ErrorResponse
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
