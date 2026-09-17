package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Archeris-api/utils"
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "File too large. Maximum size is 10MB.", "max_size_mb": 10})
			return
		}

		// Validate file type: images, PDF, and common dashboard docs (Word, Excel)
		allowedTypes := []string{
			"image/jpeg", "image/png", "image/gif", "image/webp",
			"application/pdf",
			"application/msword",                                                      // .doc
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document", // .docx
			"application/vnd.ms-excel",                                                // .xls
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",       // .xlsx
		}
		contentType := header.Header.Get("Content-Type")
		if idx := strings.Index(contentType, ";"); idx >= 0 {
			contentType = strings.TrimSpace(contentType[:idx])
		} else {
			contentType = strings.TrimSpace(contentType)
		}

		// Sniff real magic bytes from the first 512 bytes
		sniffBuf := make([]byte, 512)
		n, _ := file.Read(sniffBuf)
		if seeker, ok := file.(io.ReadSeeker); ok {
			_, _ = seeker.Seek(0, io.SeekStart)
		}

		detectedType := http.DetectContentType(sniffBuf[:n])
		if idx := strings.Index(detectedType, ";"); idx >= 0 {
			detectedType = strings.TrimSpace(detectedType[:idx])
		}

		// Prohibit dangerous executable / script content
		disallowedTypes := []string{
			"application/x-dosexec", "application/x-executable", "application/x-sharedlib",
			"application/javascript", "text/javascript", "text/html", "application/x-sh",
		}
		for _, dt := range disallowedTypes {
			if detectedType == dt {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Format file berbahaya atau tidak diizinkan."})
				return
			}
		}

		isAllowed := false
		for _, t := range allowedTypes {
			if contentType == t || detectedType == t {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File type not allowed. Allowed: JPEG, PNG, GIF, WebP, PDF, DOC, DOCX, XLS, XLSX."})
			return
		}

		// Check tournament storage quota if tournament_id / event_id is provided
		tournamentID := c.PostForm("tournament_id")
		if tournamentID == "" {
			tournamentID = c.PostForm("event_id")
		}

		var targetTourUUID string
		if tournamentID != "" {
			var tour struct {
				UUID            string  `db:"uuid"`
				Name            string  `db:"name"`
				QuotaType       *string `db:"quota_type"`
				QuotaMaxMediaMB *int    `db:"quota_max_media_mb"`
			}
			if err := db.Get(&tour, "SELECT uuid, name, quota_type, quota_max_media_mb FROM tournaments WHERE uuid = ? OR slug = ?", tournamentID, tournamentID); err == nil {
				targetTourUUID = tour.UUID
				var limitMB int64 = 200 // default Free Starter = 200 MB
				if tour.QuotaMaxMediaMB != nil && *tour.QuotaMaxMediaMB > 0 {
					limitMB = int64(*tour.QuotaMaxMediaMB)
				} else if tour.QuotaType != nil {
					switch strings.ToLower(*tour.QuotaType) {
					case "elite":
						limitMB = 10240 // 10 GB
					case "standard":
						limitMB = 3072 // 3 GB
					case "free":
						limitMB = 200 // 200 MB
					}
				}

				var currentUsage int64
				_ = db.Get(&currentUsage, "SELECT COALESCE(SUM(size), 0) FROM media WHERE caption LIKE ?", "%[t:"+tour.UUID+"]%")

				maxBytes := limitMB * 1024 * 1024
				if currentUsage+header.Size > maxBytes {
					c.JSON(http.StatusRequestEntityTooLarge, gin.H{
						"error": fmt.Sprintf("Kapasitas penyimpanan media turnamen telah mencapai batas maksimal paket Anda (%d MB). Silakan tingkatkan paket Anda atau hapus file yang tidak terpakai.", limitMB),
						"code":        "QUOTA_MEDIA_EXCEEDED",
						"limit_mb":    limitMB,
						"used_bytes":  currentUsage,
						"upload_size": header.Size,
					})
					return
				}
			}
		}

		// Generate filename from caption or UUID
		ext := filepath.Ext(header.Filename)
		if ext == "" {
			switch contentType {
			case "image/jpeg":
				ext = ".jpg"
			case "image/png":
				ext = ".png"
			case "image/gif":
				ext = ".gif"
			case "image/webp":
				ext = ".webp"
			case "application/pdf":
				ext = ".pdf"
			case "application/msword":
				ext = ".doc"
			case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
				ext = ".docx"
			case "application/vnd.ms-excel":
				ext = ".xls"
			case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
				ext = ".xlsx"
			default:
				ext = ".bin"
			}
		}

		// Get caption from form
		rawCaption := c.PostForm("caption")
		if rawCaption == "" {
			rawCaption = header.Filename
		}

		dbCaption := rawCaption
		if targetTourUUID != "" {
			dbCaption = "[t:" + targetTourUUID + "] " + rawCaption
		}

		fileID := uuid.New().String()
		var filename string

		if rawCaption != "" {
			// Slugify caption: lowercase, replace spaces with hyphens, remove special chars
			slug := strings.ToLower(rawCaption)
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
		`, fileID, userID, userType, filename, dbCaption, contentType, written)

		if err != nil {
			fmt.Printf("[ERROR] Failed to save media to database: %v\n", err)
			// We don't return error here because the file is already uploaded successfully
		}

		c.JSON(http.StatusCreated, response)
	}
}

// GetMedia serves a media file
// GET /media/:filename or GET /api/v1/media/:filename
func GetMedia(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		filename := c.Param("filename")

		// Handle SVG avatar generator requests: /media/svg?seed=...
		if filename == "svg" {
			seed := c.Query("seed")
			if seed == "" {
				seed = "default"
			}
			c.Redirect(http.StatusFound, fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s", seed))
			return
		}

		// Sanitize filename to prevent directory traversal
		cleanName := filepath.Base(filename)

		// 1. Direct file check on disk
		filePath := filepath.Join("./media", cleanName)
		if _, err := os.Stat(filePath); err == nil {
			c.File(filePath)
			return
		}

		// 2. Try querying media table in database by UUID or URL filename
		if db != nil {
			var dbURL string
			err := db.Get(&dbURL, "SELECT url FROM media WHERE uuid = ? OR url = ? LIMIT 1", cleanName, cleanName)
			if err == nil && dbURL != "" {
				dbClean := filepath.Base(dbURL)
				dbPath := filepath.Join("./media", dbClean)
				if _, err := os.Stat(dbPath); err == nil {
					c.File(dbPath)
					return
				}
			}
		}

		// 3. Fallback: try common extensions (.jpg, .png, .jpeg, .webp)
		exts := []string{".jpg", ".png", ".jpeg", ".webp", ".gif", ".pdf"}
		for _, ext := range exts {
			altPath := filepath.Join("./media", cleanName+ext)
			if _, err := os.Stat(altPath); err == nil {
				c.File(altPath)
				return
			}
		}

		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
	}
}

// DownloadMedia serves a media file as an attachment for download
// GET /media-download/:filename
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

		// Format URL for response
		for i := range mediaFiles {
			mediaFiles[i].URL = utils.MaskMediaURL(mediaFiles[i].URL)
		}

		c.JSON(http.StatusOK, gin.H{"data": mediaFiles})
	}
}

// TournamentMediaFileItem represents an uploaded file item for a tournament
type TournamentMediaFileItem struct {
	ID            string `json:"id"`
	Filename      string `json:"filename"`
	URL           string `json:"url"`
	Size          int64  `json:"size"`
	SizeFormatted string `json:"size_formatted"`
	MimeType      string `json:"mime_type"`
	Category      string `json:"category"`
	CreatedAt     string `json:"created_at"`
}

// TournamentMediaUsageResponse represents media usage information for a tournament
type TournamentMediaUsageResponse struct {
	TournamentID    string                    `json:"tournament_id"`
	TournamentName  string                    `json:"tournament_name"`
	QuotaType       string                    `json:"quota_type"`
	UsedBytes       int64                     `json:"used_bytes"`
	UsedMB          float64                   `json:"used_mb"`
	LimitMB         int64                     `json:"limit_mb"`
	LimitBytes      int64                     `json:"limit_bytes"`
	UsagePercentage float64                   `json:"usage_percentage"`
	TotalFiles      int                       `json:"total_files"`
	Files           []TournamentMediaFileItem `json:"files"`
}

func formatBytes(bytes int64) string {
	if bytes >= 1024*1024*1024 {
		return fmt.Sprintf("%.2f GB", float64(bytes)/(1024*1024*1024))
	} else if bytes >= 1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
	} else if bytes >= 1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%d B", bytes)
}

// GetTournamentMediaUsage returns storage metrics and files for a tournament
// GET /tournaments/:id/media-storage
func GetTournamentMediaUsage(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tournament ID required"})
			return
		}

		var tour struct {
			UUID                  string  `db:"uuid"`
			Name                  string  `db:"name"`
			BannerURL             *string `db:"banner_url"`
			LogoURL               *string `db:"logo_url"`
			TechnicalGuidebookURL *string `db:"technical_guidebook_url"`
			QuotaType             *string `db:"quota_type"`
			QuotaMaxMediaMB       *int    `db:"quota_max_media_mb"`
		}

		if err := db.Get(&tour, "SELECT uuid, name, banner_url, logo_url, technical_guidebook_url, quota_type, quota_max_media_mb FROM tournaments WHERE uuid = ? OR slug = ?", id, id); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tournament not found"})
			return
		}

		var limitMB int64 = 200
		quotaTypeStr := "free"
		if tour.QuotaType != nil && *tour.QuotaType != "" {
			quotaTypeStr = strings.ToLower(*tour.QuotaType)
		}

		if tour.QuotaMaxMediaMB != nil && *tour.QuotaMaxMediaMB > 0 {
			limitMB = int64(*tour.QuotaMaxMediaMB)
		} else {
			switch quotaTypeStr {
			case "elite":
				limitMB = 10240
			case "standard":
				limitMB = 3072
			default:
				limitMB = 200
			}
		}

		type DbMediaItem struct {
			UUID      string    `db:"uuid"`
			URL       string    `db:"url"`
			Caption   string    `db:"caption"`
			MimeType  string    `db:"mime_type"`
			Size      int64     `db:"size"`
			CreatedAt time.Time `db:"created_at"`
		}

		var dbFiles []DbMediaItem
		_ = db.Select(&dbFiles, "SELECT uuid, url, caption, mime_type, size, created_at FROM media WHERE caption LIKE ? ORDER BY created_at DESC", "%[t:"+tour.UUID+"]%")

		var files []TournamentMediaFileItem
		var totalUsedBytes int64 = 0

		for _, item := range dbFiles {
			cleanCaption := strings.ReplaceAll(item.Caption, "[t:"+tour.UUID+"] ", "")
			cleanCaption = strings.ReplaceAll(cleanCaption, "[t:"+tour.UUID+"]", "")

			category := "document"
			mime := strings.ToLower(item.MimeType)
			if strings.HasPrefix(mime, "image/") {
				if strings.Contains(strings.ToLower(cleanCaption), "banner") {
					category = "banner"
				} else if strings.Contains(strings.ToLower(cleanCaption), "logo") {
					category = "logo"
				} else {
					category = "gallery"
				}
			} else if strings.Contains(mime, "pdf") {
				category = "document"
			}

			totalUsedBytes += item.Size
			files = append(files, TournamentMediaFileItem{
				ID:            item.UUID,
				Filename:      cleanCaption,
				URL:           utils.MaskMediaURL(item.URL),
				Size:          item.Size,
				SizeFormatted: formatBytes(item.Size),
				MimeType:      item.MimeType,
				Category:      category,
				CreatedAt:     item.CreatedAt.Format(time.RFC3339),
			})
		}

		limitBytes := limitMB * 1024 * 1024
		usedMB := float64(totalUsedBytes) / (1024 * 1024)
		var usagePct float64 = 0
		if limitBytes > 0 {
			usagePct = float64(totalUsedBytes) / float64(limitBytes) * 100
		}

		c.JSON(http.StatusOK, TournamentMediaUsageResponse{
			TournamentID:    tour.UUID,
			TournamentName:  tour.Name,
			QuotaType:       quotaTypeStr,
			UsedBytes:       totalUsedBytes,
			UsedMB:          usedMB,
			LimitMB:         limitMB,
			LimitBytes:      limitBytes,
			UsagePercentage: usagePct,
			TotalFiles:      len(files),
			Files:           files,
		})
	}
}

// DeleteMedia handles media deletion
// DELETE /api/v1/media/:id
func DeleteMedia(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID, _ := c.Get("user_id")
		role, _ := c.Get("role")

		if userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Get file url from database
		var filename string
		var err error
		if role == "admin" {
			err = db.Get(&filename, "SELECT url FROM media WHERE uuid = ?", id)
		} else {
			err = db.Get(&filename, "SELECT url FROM media WHERE uuid = ? AND (user_id = ? OR caption LIKE ?)", id, userID, "%"+fmt.Sprintf("%v", userID)+"%")
		}

		if err != nil {
			// Fallback: check if media exists
			err = db.Get(&filename, "SELECT url FROM media WHERE uuid = ?", id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
				return
			}
		}

		// Delete from storage
		filePath := filepath.Join("./media", filename)
		_ = os.Remove(filePath)

		// Delete from database
		_, err = db.Exec("DELETE FROM media WHERE uuid = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete media", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Media deleted successfully"})
	}
}
