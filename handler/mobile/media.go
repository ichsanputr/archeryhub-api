package mobile

import (
	"Archeris-api/handler"
	"Archeris-api/utils"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// MobileUploadMedia handles file uploads for mobile
// @Summary Upload Media (Mobile)
// @Description Upload a file (image/pdf/doc) and get a masked URL. Optimized for mobile app.
// @Tags Mobile - Media
// @Accept multipart/form-data
// @Produce json
// @Security ApiKeyAuth
// @Param file formData file true "File to upload"
// @Param caption formData string false "Optional caption/title for the file"
// @Success 201 {object} handler.MediaUploadResponse
// @Router /mobile/media/upload [post]
func MobileUploadMedia(db *sqlx.DB) gin.HandlerFunc {
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "File too large. Maximum size is 10MB.", "max_size_mb": 10})
			return
		}

		// Validate file type
		allowedTypes := []string{
			"image/jpeg", "image/png", "image/gif", "image/webp",
			"application/pdf",
			"application/msword",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.ms-excel",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		}
		contentType := header.Header.Get("Content-Type")
		if idx := strings.Index(contentType, ";"); idx >= 0 {
			contentType = strings.TrimSpace(contentType[:idx])
		} else {
			contentType = strings.TrimSpace(contentType)
		}
		
		isAllowed := false
		for _, t := range allowedTypes {
			if contentType == t {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File type not allowed. Allowed: JPEG, PNG, GIF, WebP, PDF, DOC, DOCX, XLS, XLSX."})
			return
		}

		// Generate filename
		ext := filepath.Ext(header.Filename)
		if ext == "" {
			switch contentType {
			case "image/jpeg": ext = ".jpg"
			case "image/png": ext = ".png"
			case "image/gif": ext = ".gif"
			case "image/webp": ext = ".webp"
			case "application/pdf": ext = ".pdf"
			default: ext = ".bin"
			}
		}
		
		caption := c.PostForm("caption")
		fileID := uuid.New().String()
		var filename string
		
		if caption != "" {
			slug := strings.ToLower(caption)
			slug = strings.ReplaceAll(slug, " ", "-")
			var cleanSlug strings.Builder
			for _, r := range slug {
				if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
					cleanSlug.WriteRune(r)
				}
			}
			filename = cleanSlug.String() + "-" + fileID[:8] + ext
		} else {
			filename = fileID + ext
		}

		// Ensure media directory exists
		mediaDir := "./media"
		if err := os.MkdirAll(mediaDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create media directory"})
			return
		}

		// Save the file
		filePath := filepath.Join(mediaDir, filename)
		out, err := os.Create(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create file"})
			return
		}
		defer out.Close()

		written, err := io.Copy(out, file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
			return
		}

		// Save to database
		_, err = db.Exec(`
			INSERT INTO media (uuid, user_id, user_type, url, caption, mime_type, size)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, fileID, userID, userType, filename, caption, contentType, written)
		if err != nil {
			fmt.Printf("[ERROR] Mobile media upload DB save failed: %v\n", err)
		}

		c.JSON(http.StatusCreated, handler.MediaUploadResponse{
			ID:        fileID,
			Filename:  filename,
			URL:       utils.MaskMediaURL(filename),
			Size:      written,
			MimeType:  contentType,
			CreatedAt: time.Now().Format(time.RFC3339),
		})
	}
}

