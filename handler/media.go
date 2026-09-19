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

		var targetTourUUID *string
		if tournamentID != "" {
			var tour struct {
				UUID            string  `db:"uuid"`
				Name            string  `db:"name"`
				QuotaType       *string `db:"quota_type"`
				QuotaMaxMediaMB *int    `db:"quota_max_media_mb"`
			}
			if err := db.Get(&tour, "SELECT uuid, name, quota_type, quota_max_media_mb FROM tournaments WHERE uuid = ? OR slug = ?", tournamentID, tournamentID); err == nil {
				targetTourUUID = &tour.UUID
				var limitMB int64 = 200 // default Free Starter = 200 MB
				if tour.QuotaMaxMediaMB != nil && *tour.QuotaMaxMediaMB > 0 {
					limitMB = int64(*tour.QuotaMaxMediaMB)
				} else if tour.QuotaType != nil {
					switch strings.ToLower(*tour.QuotaType) {
					case "elite":
						limitMB = 10240 // 10 GB
					case "standard":
						limitMB = 3072 // 3 GB
					case "unlimited":
						limitMB = 10240 // 10 GB
					case "free":
						limitMB = 200 // 200 MB
					}
				}

				var currentUsage int64
				_ = db.Get(&currentUsage, "SELECT COALESCE(SUM(size), 0) FROM media WHERE tournament_id = ? OR caption LIKE ?", tour.UUID, "%[t:"+tour.UUID+"]%")

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
			INSERT INTO media (uuid, user_id, tournament_id, user_type, url, caption, mime_type, size)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, fileID, userID, targetTourUUID, userType, filename, rawCaption, contentType, written)

		if err != nil {
			fmt.Printf("[ERROR] Failed to save media to database: %v\n", err)
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
			err := db.Get(&dbURL, "SELECT url FROM media WHERE uuid = ? OR url = ? OR url LIKE ? LIMIT 1", cleanName, cleanName, "%/"+cleanName)
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

		if userID == nil {
			userID = "guest"
		}

		var mediaFiles []MediaListResponse
		query := `
			SELECT 
				uuid as id, 
				COALESCE(NULLIF(caption, ''), url) as filename, 
				url, 
				COALESCE(size, 0) as size, 
				COALESCE(mime_type, 'image/jpeg') as mime_type, 
				created_at 
			FROM media 
			WHERE user_id = ? 
			ORDER BY created_at DESC
		`
		err := db.Select(&mediaFiles, query, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch media library", "details": err.Error()})
			return
		}

		// Format URL and clean filename for response
		for i := range mediaFiles {
			mediaFiles[i].URL = utils.MaskMediaURL(mediaFiles[i].URL)
			// If filename has tour prefix [t:...], clean it for display
			if strings.HasPrefix(mediaFiles[i].Filename, "[t:") {
				if idx := strings.Index(mediaFiles[i].Filename, "] "); idx != -1 {
					mediaFiles[i].Filename = mediaFiles[i].Filename[idx+2:]
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"data": mediaFiles, "files": mediaFiles})
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
			OrganizerID           *string `db:"organizer_id"`
		}

		if err := db.Get(&tour, "SELECT uuid, name, banner_url, logo_url, technical_guidebook_url, quota_type, quota_max_media_mb, organizer_id FROM tournaments WHERE uuid = ? OR slug = ?", id, id); err != nil {
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
			case "unlimited":
				limitMB = 10240
			default:
				limitMB = 200
			}
		}

		type DbMediaItem struct {
			UUID         string    `db:"uuid"`
			URL          string    `db:"url"`
			Caption      *string   `db:"caption"`
			MimeType     *string   `db:"mime_type"`
			Size         *int64    `db:"size"`
			TournamentID *string   `db:"tournament_id"`
			CreatedAt    time.Time `db:"created_at"`
		}

		fileMap := make(map[string]TournamentMediaFileItem)

		// 1. Fetch from media table where tournament_id matches, or legacy caption tags, or matching tournament name
		var dbFiles []DbMediaItem
		_ = db.Select(&dbFiles, `
			SELECT uuid, url, caption, mime_type, size, tournament_id, created_at 
			FROM media 
			WHERE tournament_id = ? 
			   OR caption LIKE ? 
			   OR caption LIKE ?
			ORDER BY created_at DESC
		`, tour.UUID, "%[t:"+tour.UUID+"]%", "%"+tour.Name+"%")

		for _, item := range dbFiles {
			cleanCaption := ""
			if item.Caption != nil {
				cleanCaption = *item.Caption
				cleanCaption = strings.ReplaceAll(cleanCaption, "[t:"+tour.UUID+"] ", "")
				cleanCaption = strings.ReplaceAll(cleanCaption, "[t:"+tour.UUID+"]", "")
			}
			if cleanCaption == "" {
				cleanCaption = utils.ExtractFilename(item.URL)
			}

			rawURL := item.URL
			cleanFilename := utils.ExtractFilename(rawURL)

			var fileSize int64 = 0
			if item.Size != nil && *item.Size > 0 {
				fileSize = *item.Size
			} else {
				if fi, err := os.Stat(filepath.Join("./media", cleanFilename)); err == nil {
					fileSize = fi.Size()
				}
			}

			mimeType := "image/jpeg"
			if item.MimeType != nil && *item.MimeType != "" {
				mimeType = *item.MimeType
			} else {
				mimeType = utils.DetectMimeType(cleanFilename)
			}

			category := utils.DetermineMediaCategory(cleanCaption, cleanFilename, mimeType)

			entry := TournamentMediaFileItem{
				ID:            item.UUID,
				Filename:      cleanCaption,
				URL:           utils.MaskMediaURL(rawURL),
				Size:          fileSize,
				SizeFormatted: formatBytes(fileSize),
				MimeType:      mimeType,
				Category:      category,
				CreatedAt:     item.CreatedAt.Format(time.RFC3339),
			}

			fileMap[cleanFilename] = entry
		}

		// 2. Fetch from tournament_images (gallery)
		type DbTourImage struct {
			UUID      string    `db:"uuid"`
			URL       string    `db:"url"`
			Caption   *string   `db:"caption"`
			AltText   *string   `db:"alt_text"`
			CreatedAt time.Time `db:"created_at"`
		}
		var tourImages []DbTourImage
		_ = db.Select(&tourImages, "SELECT uuid, url, caption, alt_text, created_at FROM tournament_images WHERE tournament_id = ? ORDER BY display_order ASC, created_at DESC", tour.UUID)

		for _, img := range tourImages {
			cleanFilename := utils.ExtractFilename(img.URL)
			if _, exists := fileMap[cleanFilename]; exists {
				continue
			}

			var fileSize int64 = 0
			if fi, err := os.Stat(filepath.Join("./media", cleanFilename)); err == nil {
				fileSize = fi.Size()
			}

			caption := "Gallery Image"
			if img.Caption != nil && *img.Caption != "" {
				caption = *img.Caption
			} else if img.AltText != nil && *img.AltText != "" {
				caption = *img.AltText
			} else {
				caption = cleanFilename
			}

			mimeType := utils.DetectMimeType(cleanFilename)

			entry := TournamentMediaFileItem{
				ID:            img.UUID,
				Filename:      caption,
				URL:           utils.MaskMediaURL(img.URL),
				Size:          fileSize,
				SizeFormatted: formatBytes(fileSize),
				MimeType:      mimeType,
				Category:      "gallery",
				CreatedAt:     img.CreatedAt.Format(time.RFC3339),
			}
			fileMap[cleanFilename] = entry
		}

		// 3. Check direct tournament assets (Banner, Logo, Guidebook)
		type AssetRef struct {
			URL      *string
			Category string
			Name     string
		}
		assets := []AssetRef{
			{URL: tour.BannerURL, Category: "banner", Name: fmt.Sprintf("Banner - %s", tour.Name)},
			{URL: tour.LogoURL, Category: "logo", Name: fmt.Sprintf("Logo - %s", tour.Name)},
			{URL: tour.TechnicalGuidebookURL, Category: "document", Name: fmt.Sprintf("Guidebook - %s", tour.Name)},
		}

		for _, asset := range assets {
			if asset.URL == nil || *asset.URL == "" {
				continue
			}
			cleanFilename := utils.ExtractFilename(*asset.URL)
			if cleanFilename == "" {
				continue
			}
			if _, exists := fileMap[cleanFilename]; exists {
				continue
			}

			var fileSize int64 = 0
			mimeType := utils.DetectMimeType(cleanFilename)
			createdAt := time.Now()

			var existingMedia DbMediaItem
			if err := db.Get(&existingMedia, "SELECT uuid, url, caption, mime_type, size, tournament_id, created_at FROM media WHERE url = ? OR url LIKE ? LIMIT 1", cleanFilename, "%"+cleanFilename); err == nil {
				if existingMedia.Size != nil && *existingMedia.Size > 0 {
					fileSize = *existingMedia.Size
				}
				if existingMedia.MimeType != nil && *existingMedia.MimeType != "" {
					mimeType = *existingMedia.MimeType
				}
				createdAt = existingMedia.CreatedAt
			}

			if fileSize == 0 {
				if fi, err := os.Stat(filepath.Join("./media", cleanFilename)); err == nil {
					fileSize = fi.Size()
					createdAt = fi.ModTime()
				}
			}

			entry := TournamentMediaFileItem{
				ID:            uuid.New().String(),
				Filename:      asset.Name,
				URL:           utils.MaskMediaURL(*asset.URL),
				Size:          fileSize,
				SizeFormatted: formatBytes(fileSize),
				MimeType:      mimeType,
				Category:      asset.Category,
				CreatedAt:     createdAt.Format(time.RFC3339),
			}
			fileMap[cleanFilename] = entry
		}

		// Assemble results list
		var files []TournamentMediaFileItem
		var totalUsedBytes int64 = 0

		for _, file := range fileMap {
			totalUsedBytes += file.Size
			files = append(files, file)
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

		// Find media record
		var mediaItem struct {
			UUID         string  `db:"uuid"`
			URL          string  `db:"url"`
			UserID       string  `db:"user_id"`
			TournamentID *string `db:"tournament_id"`
		}

		err := db.Get(&mediaItem, "SELECT uuid, url, user_id, tournament_id FROM media WHERE uuid = ? OR url = ? OR url LIKE ? LIMIT 1", id, id, "%/"+id)
		if err != nil {
			// Check if it's in tournament_images
			var tourImage struct {
				UUID         string `db:"uuid"`
				TournamentID string `db:"tournament_id"`
				URL          string `db:"url"`
			}
			if err2 := db.Get(&tourImage, "SELECT uuid, tournament_id, url FROM tournament_images WHERE uuid = ? OR url = ?", id, id); err2 == nil {
				filename := utils.ExtractFilename(tourImage.URL)
				_ = os.Remove(filepath.Join("./media", filename))
				_, _ = db.Exec("DELETE FROM tournament_images WHERE uuid = ?", tourImage.UUID)
				c.JSON(http.StatusOK, gin.H{"message": "Gambar turnamen berhasil dihapus"})
				return
			}

			c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
			return
		}

		// Authorization check: admin, media owner, or tournament owner
		isAuthorized := role == "admin" || mediaItem.UserID == userID.(string)
		if !isAuthorized && mediaItem.TournamentID != nil && *mediaItem.TournamentID != "" {
			var tourOwnerID string
			if errOrg := db.Get(&tourOwnerID, "SELECT organizer_id FROM tournaments WHERE uuid = ?", *mediaItem.TournamentID); errOrg == nil {
				if tourOwnerID == userID.(string) {
					isAuthorized = true
				}
			}
		}

		if !isAuthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden to delete this media"})
			return
		}

		// Delete physical file from storage
		filename := utils.ExtractFilename(mediaItem.URL)
		if filename != "" {
			_ = os.Remove(filepath.Join("./media", filename))
		}

		// Delete from database
		_, err = db.Exec("DELETE FROM media WHERE uuid = ?", mediaItem.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete media", "details": err.Error()})
			return
		}

		// Also clean up tournament_images if matching
		_, _ = db.Exec("DELETE FROM tournament_images WHERE url = ? OR url LIKE ?", filename, "%"+filename)

		c.JSON(http.StatusOK, gin.H{"message": "Media deleted successfully"})
	}
}
