package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DocsComment struct {
	UUID      string  `db:"uuid" json:"id"`
	DocSlug   string  `db:"doc_slug" json:"doc_slug"`
	UserID    *string `db:"user_id" json:"user_id,omitempty"`
	UserType  string  `db:"user_type" json:"user_type"`
	UserName  string  `db:"user_name" json:"user_name"`
	GuestName *string `db:"guest_name" json:"guest_name,omitempty"`
	Content   string  `db:"content" json:"content"`
	Status    string  `db:"status" json:"status"`
	CreatedAt string  `db:"created_at" json:"created_at"`
	ParentID  *string `db:"parent_id" json:"parent_id,omitempty"`
}

// ListDocsComments returns all approved comments for a specific docs article slug
func ListDocsComments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := normalizeDocSlugParam(c.Param("slug"))

		var comments []DocsComment
		query := `
			SELECT c.uuid, c.doc_slug, c.user_id, c.user_type, c.guest_name, c.content, c.created_at, c.parent_id,
				CASE 
					WHEN c.user_type = 'archer' THEN (SELECT full_name FROM archers WHERE uuid = c.user_id)
					WHEN c.user_type = 'organizer' THEN (SELECT name FROM organizers WHERE uuid = c.user_id)
					WHEN c.user_type = 'seller' THEN (SELECT store_name FROM sellers WHERE uuid = c.user_id)
					ELSE c.guest_name
				END as user_name
			FROM docs_comments c
			WHERE c.doc_slug = ? AND c.status = 'approved'
			ORDER BY c.created_at ASC
		`
		err := db.Select(&comments, query, slug)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments", "details": err.Error()})
			return
		}

		if comments == nil {
			comments = []DocsComment{}
		}

		c.JSON(http.StatusOK, gin.H{
			"comments": comments,
			"count":    len(comments),
		})
	}
}

// AddDocsComment adds a new comment to a docs article slug
func AddDocsComment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := normalizeDocSlugParam(c.Param("slug"))
		
		var req struct {
			GuestName string  `json:"guest_name"`
			Content   string  `json:"content" binding:"required"`
			ParentID  *string `json:"parent_id"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment data"})
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
				c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required for guest comments"})
				return
			}
		}

		commentUUID := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO docs_comments (uuid, doc_slug, user_id, user_type, guest_name, content, status, parent_id)
			VALUES (?, ?, ?, ?, ?, ?, 'approved', ?)
		`, commentUUID, slug, userID, userType, guestName, req.Content, req.ParentID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save comment", "details": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Comment successfully added", "id": commentUUID})
	}
}

type DocTOCItem struct {
	ID    string `json:"id"`
	Level int    `json:"level"`
	Text  string `json:"text"`
}

type DocLocalized struct {
	Title    string       `json:"title"`
	Excerpt  string       `json:"excerpt"`
	Content  string       `json:"content,omitempty"`
	ReadTime string       `json:"readTime"`
	TOC      []DocTOCItem `json:"toc,omitempty"`
}

type DocJSON struct {
	Slug     string       `json:"slug"`
	Icon     string       `json:"icon"`
	Category string       `json:"category"`
	ReadTime string       `json:"readTime"`
	EN       DocLocalized `json:"en"`
}

type DocResponse struct {
	Slug     string       `json:"slug"`
	Icon     string       `json:"icon"`
	Category string       `json:"category"`
	ReadTime string       `json:"readTime"`
	Title    string       `json:"title"`
	Excerpt  string       `json:"excerpt"`
	Content  string       `json:"content,omitempty"`
	TOC      []DocTOCItem `json:"toc,omitempty"`
}

func normalizeDocSlugParam(param string) string {
	// Gin wildcard params (e.g. /*slug) include a leading slash.
	// Normalize both "/archer/archer-profile" and "archer/archer-profile" to "archer/archer-profile".
	slug := strings.TrimSpace(param)
	slug = strings.TrimPrefix(slug, "/")
	slug = strings.TrimSuffix(slug, "/")
	return slug
}

func nestedSlug(category, baseSlug string) string {
	category = strings.TrimSpace(category)
	baseSlug = strings.TrimSpace(baseSlug)
	if category == "" {
		return baseSlug
	}
	return category + "/" + baseSlug
}

// ListDocs returns metadata of all documentation articles
func ListDocs() gin.HandlerFunc {
	return func(c *gin.Context) {
		files, err := os.ReadDir("data/docs")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read docs directory", "details": err.Error()})
			return
		}

		var list []DocResponse
		for _, file := range files {
			if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
				data, err := os.ReadFile(filepath.Join("data/docs", file.Name()))
				if err != nil {
					continue
				}

				var doc DocJSON
				if err := json.Unmarshal(data, &doc); err != nil {
					continue
				}

				localized := doc.EN

				list = append(list, DocResponse{
					Slug:     nestedSlug(doc.Category, doc.Slug),
					Icon:     doc.Icon,
					Category: doc.Category,
					ReadTime: doc.ReadTime,
					Title:    localized.Title,
					Excerpt:  localized.Excerpt,
				})
			}
		}

		c.JSON(http.StatusOK, list)
	}
}

// GetDocDetail returns full details of a specific documentation article
func GetDocDetail() gin.HandlerFunc {
	return func(c *gin.Context) {
		paramSlug := normalizeDocSlugParam(c.Param("slug"))
		// We store docs as flat files using the base slug as the filename.
		// URL can be nested: /docs/{category}/{slug}
		baseSlug := path.Base(paramSlug)
		if baseSlug == "." || baseSlug == "/" || baseSlug == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Documentation article not found"})
			return
		}

		// Backward-compatible aliases for renamed docs.
		// Example: /docs/user-roles -> /docs/dashboard/account-types
		if baseSlug == "user-roles" {
			baseSlug = "account-types"
		}

		filePath := filepath.Join("data/docs", baseSlug+".json")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Documentation article not found"})
			return
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read article", "details": err.Error()})
			return
		}

		var doc DocJSON
		if err := json.Unmarshal(data, &doc); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse article", "details": err.Error()})
			return
		}

		localized := doc.EN

		res := DocResponse{
			Slug:     nestedSlug(doc.Category, doc.Slug),
			Icon:     doc.Icon,
			Category: doc.Category,
			ReadTime: doc.ReadTime,
			Title:    localized.Title,
			Excerpt:  localized.Excerpt,
			Content:  localized.Content,
			TOC:      localized.TOC,
		}

		c.JSON(http.StatusOK, res)
	}
}
