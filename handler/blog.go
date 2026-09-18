package handler

import (
	"encoding/json"
	"html"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type BlogComment struct {
	UUID        string        `db:"uuid" json:"id"`
	ArticleSlug string        `db:"article_slug" json:"article_slug"`
	ParentID    *string       `db:"parent_id" json:"parent_id,omitempty"`
	UserID      *string       `db:"user_id" json:"user_id,omitempty"`
	UserType    string        `db:"user_type" json:"user_type"`
	UserName    string        `db:"user_name" json:"user_name"`
	GuestName   *string       `db:"guest_name" json:"guest_name,omitempty"`
	Content     string        `db:"content" json:"content"`
	Status      string        `db:"status" json:"status"`
	CreatedAt   string        `db:"created_at" json:"created_at"`
	Replies     []BlogComment `db:"-" json:"replies"`
}

// ListBlogComments returns all approved comments for a specific blog article slug in 2-level nested structure
func ListBlogComments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")

		var rawComments []BlogComment
		query := `
			SELECT c.uuid, c.article_slug, c.parent_id, c.user_id, c.user_type, c.guest_name, c.content, c.created_at,
				CASE 
					WHEN c.user_type = 'archer' THEN COALESCE((SELECT full_name FROM archers WHERE uuid = c.user_id), 'Archer User')
					WHEN c.user_type = 'organizer' THEN COALESCE((SELECT name FROM organizers WHERE uuid = c.user_id), 'Organizer User')
					WHEN c.user_type = 'seller' THEN COALESCE((SELECT store_name FROM sellers WHERE uuid = c.user_id), 'Seller User')
					ELSE COALESCE(c.guest_name, 'Guest')
				END as user_name
			FROM blog_comments c
			WHERE c.article_slug = ? AND c.status = 'approved'
			ORDER BY c.created_at ASC
		`
		err := db.Select(&rawComments, query, slug)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil komentar", "details": err.Error()})
			return
		}

		commentMap := make(map[string]*BlogComment)
		for i := range rawComments {
			rawComments[i].Replies = []BlogComment{}
			commentMap[rawComments[i].UUID] = &rawComments[i]
		}

		// Attach replies to parents (max 2 levels)
		for _, comment := range rawComments {
			if comment.ParentID == nil || *comment.ParentID == "" {
				continue
			}
			parentUUID := *comment.ParentID
			parent, exists := commentMap[parentUUID]
			if exists {
				if parent.ParentID != nil && *parent.ParentID != "" {
					rootParent, rootExists := commentMap[*parent.ParentID]
					if rootExists {
						rootParent.Replies = append(rootParent.Replies, comment)
						continue
					}
				}
				parent.Replies = append(parent.Replies, comment)
			}
		}

		// Collect root comments in reverse chronological order (newest root first)
		var rootComments []BlogComment
		for i := len(rawComments) - 1; i >= 0; i-- {
			comment := rawComments[i]
			if comment.ParentID == nil || *comment.ParentID == "" {
				if ptr, ok := commentMap[comment.UUID]; ok {
					rootComments = append(rootComments, *ptr)
				} else {
					rootComments = append(rootComments, comment)
				}
			}
		}

		if rootComments == nil {
			rootComments = []BlogComment{}
		}

		c.JSON(http.StatusOK, gin.H{
			"comments": rootComments,
			"count":    len(rawComments),
			"data":     rootComments,
		})
	}
}

// AddBlogComment adds a new comment or reply to a blog article slug
func AddBlogComment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")
		
		var req struct {
			ParentID    *string `json:"parent_id"`
			GuestName   string  `json:"guest_name"`
			AuthorName  string  `json:"author_name"`
			AuthorEmail string  `json:"author_email"`
			Content     string  `json:"content" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data komentar tidak valid"})
			return
		}

		userIDInterface, exists := c.Get("user_id")
		userTypeInterface, _ := c.Get("user_type")

		var userID *string
		var guestName *string
		userType := "guest"
		safeGuestName := ""
		
		if req.GuestName != "" {
			safeGuestName = req.GuestName
		} else if req.AuthorName != "" {
			safeGuestName = req.AuthorName
		}

		safeGuestName = html.EscapeString(strings.TrimSpace(safeGuestName))
		if len(safeGuestName) > 100 {
			safeGuestName = safeGuestName[:100]
		}

		if exists && userIDInterface != nil {
			uid := userIDInterface.(string)
			userID = &uid
			userType = userTypeInterface.(string)
			guestName = nil
		} else {
			if safeGuestName == "" {
				safeGuestName = "Archery Enthusiast"
			}
			guestName = &safeGuestName
		}

		safeContent := html.EscapeString(strings.TrimSpace(req.Content))
		if safeContent == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Isi komentar tidak boleh kosong"})
			return
		}
		if len(safeContent) > 2000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Isi komentar terlalu panjang (maksimal 2000 karakter)"})
			return
		}

		// Verify parent_id if provided
		var parentID *string
		if req.ParentID != nil && strings.TrimSpace(*req.ParentID) != "" {
			trimmedParentID := strings.TrimSpace(*req.ParentID)
			var parentComment struct {
				UUID        string  `db:"uuid"`
				ParentID    *string `db:"parent_id"`
				ArticleSlug string  `db:"article_slug"`
			}
			err := db.Get(&parentComment, "SELECT uuid, parent_id, article_slug FROM blog_comments WHERE uuid = ? AND article_slug = ? AND status = 'approved' LIMIT 1", trimmedParentID, slug)
			if err == nil {
				// If parent is already a reply, attach to root parent to enforce 2 levels max
				if parentComment.ParentID != nil && *parentComment.ParentID != "" {
					parentID = parentComment.ParentID
				} else {
					parentID = &parentComment.UUID
				}
			}
		}

		commentUUID := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO blog_comments (uuid, article_slug, parent_id, user_id, user_type, guest_name, content, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'approved')
		`, commentUUID, slug, parentID, userID, userType, guestName, safeContent)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan komentar", "details": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":     "Komentar berhasil ditambahkan",
			"id":          commentUUID,
			"author_name": safeGuestName,
			"content":     safeContent,
		})
	}
}

// BlogArticleDB represents raw article row in database
type BlogArticleDB struct {
	ID           int     `db:"id"`
	UUID         string  `db:"uuid"`
	Slug         string  `db:"slug"`
	Title        string  `db:"title"`
	Excerpt      string  `db:"excerpt"`
	Content      string  `db:"content"`
	Category     string  `db:"category"`
	ImageURL     string  `db:"image_url"`
	AuthorName   string  `db:"author_name"`
	AuthorRole   string  `db:"author_role"`
	AuthorAvatar string  `db:"author_avatar"`
	Tags         *string `db:"tags"`
	ImagePrompts *string `db:"image_prompts"`
	ReadTime     int     `db:"read_time"`
	Views        int     `db:"views"`
	Status       string  `db:"status"`
	PublishedAt  string  `db:"published_at"`
	CreatedAt    string  `db:"created_at"`
	UpdatedAt    string  `db:"updated_at"`
}

// BlogArticle represents a published blog article with structured tags and image_prompts array
type BlogArticle struct {
	ID           int      `json:"id"`
	UUID         string   `json:"uuid"`
	Slug         string   `json:"slug"`
	Title        string   `json:"title"`
	Excerpt      string   `json:"excerpt"`
	Content      string   `json:"content"`
	Category     string   `json:"category"`
	ImageURL     string   `json:"image"`
	AuthorName   string   `json:"author_name"`
	AuthorRole   string   `json:"author_role"`
	AuthorAvatar string   `json:"author_avatar"`
	Tags         []string `json:"tags"`
	ImagePrompts []string `json:"image_prompts"`
	ReadTime     int      `json:"read_time"`
	Views        int      `json:"views"`
	Status       string   `json:"status"`
	PublishedAt  string   `json:"published_at"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

func parseStringArray(raw *string) []string {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return []string{}
	}
	s := strings.TrimSpace(*raw)
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		var list []string
		if err := json.Unmarshal([]byte(s), &list); err == nil && len(list) > 0 {
			return list
		}
	}
	parts := strings.Split(s, ",")
	var list []string
	for _, p := range parts {
		cleaned := strings.TrimSpace(p)
		if cleaned != "" {
			list = append(list, cleaned)
		}
	}
	if list == nil {
		list = []string{}
	}
	return list
}

func parseTags(raw *string) []string {
	return parseStringArray(raw)
}

func transformArticle(dbArticle BlogArticleDB) BlogArticle {
	avatar := dbArticle.AuthorAvatar
	if avatar == "" || strings.Contains(avatar, "placeholder") || strings.Contains(avatar, "dicebear") {
		avatar = "/media/profile-author.png"
	}

	imageUrl := dbArticle.ImageURL
	if imageUrl != "" && !strings.HasPrefix(imageUrl, "http://") && !strings.HasPrefix(imageUrl, "https://") {
		if !strings.HasPrefix(imageUrl, "/") {
			imageUrl = "/" + imageUrl
		}
	}

	return BlogArticle{
		ID:           dbArticle.ID,
		UUID:         dbArticle.UUID,
		Slug:         dbArticle.Slug,
		Title:        dbArticle.Title,
		Excerpt:      dbArticle.Excerpt,
		Content:      dbArticle.Content,
		Category:     dbArticle.Category,
		ImageURL:     imageUrl,
		AuthorName:   dbArticle.AuthorName,
		AuthorRole:   dbArticle.AuthorRole,
		AuthorAvatar: avatar,
		Tags:         parseTags(dbArticle.Tags),
		ImagePrompts: parseStringArray(dbArticle.ImagePrompts),
		ReadTime:     dbArticle.ReadTime,
		Views:        dbArticle.Views,
		Status:       dbArticle.Status,
		PublishedAt:  dbArticle.PublishedAt,
		CreatedAt:    dbArticle.CreatedAt,
		UpdatedAt:    dbArticle.UpdatedAt,
	}
}

// ListBlogArticles returns all published articles from database
func ListBlogArticles(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		category := c.Query("category")
		search := c.Query("q")

		query := `
			SELECT id, uuid, slug, title, excerpt, content, category, image_url, 
			       author_name, author_role, author_avatar, tags, image_prompts, read_time, views, status, 
			       published_at, created_at, updated_at
			FROM blog_articles
			WHERE status = 'published'
		`
		var args []interface{}

		if category != "" && category != "All" {
			cleanSlug := strings.ToLower(strings.TrimSpace(category))
			query += ` AND (
				category = ? 
				OR LOWER(category) = ? 
				OR LOWER(REPLACE(REPLACE(REPLACE(category, '&', 'and'), ' ', '-'), '--', '-')) = ?
				OR LOWER(REPLACE(REPLACE(REPLACE(category, '&', ''), ' ', '-'), '--', '-')) = ?
			)`
			args = append(args, category, cleanSlug, cleanSlug, cleanSlug)
		}

		if search != "" {
			query += " AND (title LIKE ? OR excerpt LIKE ?)"
			term := "%" + search + "%"
			args = append(args, term, term)
		}

		query += " ORDER BY published_at DESC"

		var dbArticles []BlogArticleDB
		err := db.Select(&dbArticles, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar artikel", "details": err.Error()})
			return
		}

		articles := make([]BlogArticle, len(dbArticles))
		for i, a := range dbArticles {
			articles[i] = transformArticle(a)
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  articles,
			"count": len(articles),
		})
	}
}

// GetBlogArticle returns a single published article by slug
func GetBlogArticle(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")

		var dbArticle BlogArticleDB
		query := `
			SELECT id, uuid, slug, title, excerpt, content, category, image_url, 
			       author_name, author_role, author_avatar, tags, image_prompts, read_time, views, status, 
			       published_at, created_at, updated_at
			FROM blog_articles
			WHERE slug = ? AND status = 'published'
			LIMIT 1
		`
		err := db.Get(&dbArticle, query, slug)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Artikel tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": transformArticle(dbArticle),
		})
	}
}

// IncrementBlogArticleViews (generic view tracker since the article is static, we track it by slug)
func IncrementBlogArticleViews(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")
		
		_, _ = db.Exec("UPDATE blog_articles SET views = views + 1 WHERE slug = ?", slug)
		
		c.JSON(http.StatusOK, gin.H{"message": "Analitik diperbarui untuk " + slug})
	}
}

// SubscribeNewsletter registers an email address for newsletter updates
func SubscribeNewsletter(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email string `json:"email"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format data tidak valid"})
			return
		}

		email := strings.ToLower(strings.TrimSpace(req.Email))
		if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") || len(email) < 5 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format email tidak valid. Masukkan alamat email yang benar."})
			return
		}

		_, err := db.Exec(`
			INSERT INTO news_subscribers (email, is_active, created_at, updated_at)
			VALUES (?, 1, NOW(), NOW())
			ON DUPLICATE KEY UPDATE is_active = 1, updated_at = NOW()
		`, email)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses langganan newsletter", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Terima kasih! Email Anda telah terdaftar untuk menerima newsletter Archeris.",
			"email":   email,
		})
	}
}
