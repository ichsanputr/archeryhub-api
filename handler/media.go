package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"archeryhub-api/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// MediaUploadResponse represents the response after uploading a file
type MediaUploadResponse struct {
	ID        string `json:"id" db:"uuid"`
	Filename  string `json:"filename" db:"filename"`
	URL       string `json:"url" db:"url"`
	Size      int64  `json:"size" db:"size"`
	MimeType  string `json:"mime_type" db:"mime_type"`
	CreatedAt string `json:"created_at" db:"created_at"`
}

// MediaListResponse represents a media file in the list
type MediaListResponse struct {
	ID        string `json:"id" db:"id"`
	Filename  string `json:"filename" db:"filename"`
	URL       string `json:"url" db:"url"`
	Size      int64  `json:"size" db:"size"`
	MimeType  string `json:"mime_type" db:"mime_type"`
	CreatedAt string `json:"created_at" db:"created_at"`
}

// UploadMedia handles file uploads
// POST /api/v1/media/upload
func UploadMedia(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")
		
		if userID == nil {
			userID = "guest"
		}
		if userType == nil {
			userType = "visitor"
		}

		// Get the file from the request
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided", "details": err.Error()})
			return
		}
		defer file.Close()

		// Validate file size (max 10MB)
		const maxSize = 10 * 1024 * 1024
		if header.Size > maxSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File too large. Maximum size is 10MB"})
			return
		}

		// Validate file type
		allowedTypes := []string{"image/jpeg", "image/png", "image/gif", "image/webp", "application/pdf"}
		contentType := header.Header.Get("Content-Type")
		isAllowed := false
		for _, t := range allowedTypes {
			if strings.HasPrefix(contentType, t) {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File type not allowed. Allowed: jpeg, png, gif, webp, pdf"})
			return
		}

		// Generate filename from caption or UUID
		ext := filepath.Ext(header.Filename)
		if ext == "" {
			// Try to get extension from content type
			switch {
			case strings.HasPrefix(contentType, "image/jpeg"):
				ext = ".jpg"
			case strings.HasPrefix(contentType, "image/png"):
				ext = ".png"
			case strings.HasPrefix(contentType, "image/gif"):
				ext = ".gif"
			case strings.HasPrefix(contentType, "image/webp"):
				ext = ".webp"
			case strings.HasPrefix(contentType, "application/pdf"):
				ext = ".pdf"
			}
		}
		
		// Get caption from form
		caption := c.PostForm("caption")
		fileID := uuid.New().String()
		var filename string
		
		if caption != "" {
			// Slugify caption: lowercase, replace spaces with hyphens, remove special chars
			slug := strings.ToLower(caption)
			slug = strings.ReplaceAll(slug, " ", "-")
			// Remove non-alphanumeric except hyphens
			var cleanSlug strings.Builder
			for _, r := range slug {
				if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
					cleanSlug.WriteRune(r)
				}
			}
			// Add short unique suffix to prevent collisions
			shortID := fileID[:8]
			filename = cleanSlug.String() + "-" + shortID + ext
		} else {
			filename = fileID + ext
		}

		// Ensure media directory exists
		mediaDir := "./media"
		if err := os.MkdirAll(mediaDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create media directory", "details": err.Error()})
			return
		}

		// Create the file
		filePath := filepath.Join(mediaDir, filename)
		out, err := os.Create(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create file", "details": err.Error()})
			return
		}
		defer out.Close()

		// Copy the file content
		written, err := io.Copy(out, file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file", "details": err.Error()})
			return
		}

		response := MediaUploadResponse{
			ID:        fileID,
			Filename:  filename,
			URL:       utils.MaskMediaURL(filename),
			Size:      written,
			MimeType:  contentType,
			CreatedAt: time.Now().Format(time.RFC3339),
		}

		// Save to database (store filename only in url column)
		_, err = db.Exec(`
			INSERT INTO media (uuid, user_id, user_type, url, caption, mime_type, size)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, fileID, userID, userType, filename, caption, contentType, written)
		
		if err != nil {
			fmt.Printf("[ERROR] Failed to save media to database: %v\n", err)
			// We don't return error here because the file is already uploaded successfully
		}

		c.JSON(http.StatusCreated, response)
	}
}

// GetMedia serves a media file
// GET /media/:filename
func GetMedia() gin.HandlerFunc {
	return func(c *gin.Context) {
		filename := c.Param("filename")
		
		// Sanitize filename to prevent directory traversal
		filename = filepath.Base(filename)
		
		filePath := filepath.Join("./media", filename)
		
		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}
		
		c.File(filePath)
	}
}

// DownloadMedia serves a media file as an attachment for download
// GET /media/download/:filename
func DownloadMedia() gin.HandlerFunc {
	return func(c *gin.Context) {
		filename := c.Param("filename")

		// Sanitize filename to prevent directory traversal
		filename = filepath.Base(filename)

		filePath := filepath.Join("./media", filename)

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}

		// Set header for forced download
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Transfer-Encoding", "binary")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.Header("Content-Type", "application/octet-stream")

		c.File(filePath)
	}
}

// ListMedia returns a list of all media files
// GET /api/v1/media
func ListMedia(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		if userID == nil {
			userID = "guest"
		}
		if userType == nil {
			userType = "visitor"
		}

		var mediaFiles []MediaListResponse
		query := `SELECT uuid as id, caption as filename, url, size, mime_type, created_at FROM media WHERE user_id = ? AND user_type = ? ORDER BY created_at DESC`
		err := db.Select(&mediaFiles, query, userID, userType)
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch media library", "details": err.Error()})
			return
		}

		for i := range mediaFiles {
			mediaFiles[i].URL = utils.MaskMediaURL(mediaFiles[i].URL)
		}

		c.JSON(http.StatusOK, gin.H{
			"files": mediaFiles,
			"count": len(mediaFiles),
		})
	}
}

// DeleteMedia deletes a media file
// DELETE /api/v1/media/:id
func DeleteMedia(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID, _ := c.Get("user_id")

		// Get file info from database
		var media struct {
			URL      string `db:"url"`
			UserID   string `db:"user_id"`
		}
		err := db.Get(&media, "SELECT url, user_id FROM media WHERE uuid = ?", id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
			return
		}

		// Security check
		if media.UserID != fmt.Sprintf("%v", userID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this media"})
			return
		}

		// Get filename from URL
		filename := filepath.Base(media.URL)
		filePath := filepath.Join("./media", filename)
		
		// Delete the file from disk if it exists
		if _, err := os.Stat(filePath); err == nil {
			if err := os.Remove(filePath); err != nil {
				fmt.Printf("[ERROR] Failed to delete file from disk: %v\n", err)
			}
		}
		
		// Delete from database
		_, err = db.Exec("DELETE FROM media WHERE uuid = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete media from database"})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{"message": "Media deleted successfully"})
	}
}
