package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MediaUploadResponse represents the response after uploading a file
type MediaUploadResponse struct {
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	URL       string `json:"url"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mime_type"`
	CreatedAt string `json:"created_at"`
}

// MediaListResponse represents a media file in the list
type MediaListResponse struct {
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	URL       string `json:"url"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mime_type"`
	CreatedAt string `json:"created_at"`
}

// UploadMedia handles file uploads
// POST /api/v1/media/upload
func UploadMedia() gin.HandlerFunc {
	return func(c *gin.Context) {
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

		// Build the URL
		baseURL := os.Getenv("API_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8001"
		}
		fileURL := fmt.Sprintf("%s/media/%s", baseURL, filename)

		response := MediaUploadResponse{
			ID:        fileID,
			Filename:  filename,
			URL:       fileURL,
			Size:      written,
			MimeType:  contentType,
			CreatedAt: time.Now().Format(time.RFC3339),
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

// ListMedia returns a list of all media files
// GET /api/v1/media
func ListMedia() gin.HandlerFunc {
	return func(c *gin.Context) {
		mediaDir := "./media"
		
		// Ensure directory exists
		if _, err := os.Stat(mediaDir); os.IsNotExist(err) {
			c.JSON(http.StatusOK, gin.H{
				"files": []MediaListResponse{},
				"count": 0,
			})
			return
		}
		
		files, err := os.ReadDir(mediaDir)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read media directory", "details": err.Error()})
			return
		}
		
		baseURL := os.Getenv("API_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8001"
		}
		
		var mediaFiles []MediaListResponse
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			
			info, err := f.Info()
			if err != nil {
				continue
			}
			
			// Determine mime type from extension
			ext := strings.ToLower(filepath.Ext(f.Name()))
			mimeType := "application/octet-stream"
			switch ext {
			case ".jpg", ".jpeg":
				mimeType = "image/jpeg"
			case ".png":
				mimeType = "image/png"
			case ".gif":
				mimeType = "image/gif"
			case ".webp":
				mimeType = "image/webp"
			case ".pdf":
				mimeType = "application/pdf"
			}
			
			// Extract ID from filename (UUID before extension)
			id := strings.TrimSuffix(f.Name(), ext)
			
			mediaFiles = append(mediaFiles, MediaListResponse{
				ID:        id,
				Filename:  f.Name(),
				URL:       fmt.Sprintf("%s/media/%s", baseURL, f.Name()),
				Size:      info.Size(),
				MimeType:  mimeType,
				CreatedAt: info.ModTime().Format(time.RFC3339),
			})
		}
		
		c.JSON(http.StatusOK, gin.H{
			"files": mediaFiles,
			"count": len(mediaFiles),
		})
	}
}

// DeleteMedia deletes a media file
// DELETE /api/v1/media/:filename
func DeleteMedia() gin.HandlerFunc {
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
		
		// Delete the file
		if err := os.Remove(filePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file", "details": err.Error()})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
	}
}
