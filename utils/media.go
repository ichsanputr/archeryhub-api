package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// mediaBaseURL returns the base URL for media links based on STAGE:
// - development: http://localhost:PORT (default 8001)
// - production: https://api.archeris.net
func mediaBaseURL() string {
	stage := os.Getenv("STAGE")
	if stage == "" {
		stage = os.Getenv("ENV")
	}
	if stage == "production" {
		return "https://api.archeris.net"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}
	return fmt.Sprintf("http://localhost:%s", port)
}

// GetAPIBaseURL returns the API base URL
func GetAPIBaseURL() string {
	return mediaBaseURL()
}

// MaskMediaURL converts a filename stored in the database to a full URL.
// If the input is already a full URL (starts with http), it returns it as is.
// If the input is empty or null, it returns an empty string.
// Base URL is derived from STAGE: development → localhost, production → api.archeris.net.
func MaskMediaURL(filename string) string {
	if filename == "" {
		return ""
	}

	// If it contains localhost/127.0.0.1 from local dev seeds, strip to filename
	if strings.Contains(filename, "localhost:") || strings.Contains(filename, "127.0.0.1:") {
		filename = filepath.Base(filename)
	}

	// If it's an external full URL (e.g. google avatar, cdn), return it as is
	if strings.HasPrefix(filename, "http://") || strings.HasPrefix(filename, "https://") {
		return filename
	}

	baseURL := mediaBaseURL()

	// Clean the filename (extract base if it was a path)
	cleanName := filepath.Base(filename)

	return fmt.Sprintf("%s/media/%s", baseURL, cleanName)
}

// ExtractFilename removes the base URL or path from a string to get only the filename.
// This is used before saving to the database.
func ExtractFilename(url string) string {
	if url == "" {
		return ""
	}

	// If it contains a slash, it's likely a path or URL
	if strings.Contains(url, "/") {
		return filepath.Base(url)
	}

	return url
}

// DownloadAndSaveGoogleAvatar downloads a user's Google profile picture from pictureURL,
// saves it into the local media folder, and returns the stored filename.
func DownloadAndSaveGoogleAvatar(pictureURL string, userID string) (string, error) {
	if pictureURL == "" {
		return "", nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(pictureURL)
	if err != nil {
		return "", fmt.Errorf("failed to download google avatar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("google avatar HTTP status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil || len(data) == 0 {
		return "", fmt.Errorf("failed to read avatar image body: %v", err)
	}

	ext := ".jpg"
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "png") {
		ext = ".png"
	} else if strings.Contains(contentType, "webp") {
		ext = ".webp"
	} else if strings.Contains(contentType, "gif") {
		ext = ".gif"
	}

	mediaDir := "./media"
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create media dir: %w", err)
	}

	shortID := userID
	if len(userID) >= 8 {
		shortID = userID[:8]
	}
	filename := fmt.Sprintf("avatar_google_%s_%d%s", shortID, time.Now().Unix(), ext)
	filePath := filepath.Join(mediaDir, filename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write avatar file: %w", err)
	}

	return filename, nil
}

// DetectMimeType returns a MIME type string based on file extension
func DetectMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".webm":
		return "video/webm"
	default:
		return "application/octet-stream"
	}
}

// DetermineMediaCategory returns the classification category for tournament media files
func DetermineMediaCategory(caption string, filename string, mimeType string) string {
	lowerCap := strings.ToLower(caption)
	lowerName := strings.ToLower(filename)
	lowerMime := strings.ToLower(mimeType)

	if strings.Contains(lowerCap, "banner") || strings.Contains(lowerName, "banner") {
		return "banner"
	}
	if strings.Contains(lowerCap, "logo") || strings.Contains(lowerName, "logo") {
		return "logo"
	}
	if strings.Contains(lowerCap, "guidebook") || strings.Contains(lowerCap, "juknis") || strings.Contains(lowerName, "guidebook") || strings.Contains(lowerName, "juknis") {
		return "document"
	}
	if strings.Contains(lowerCap, "gallery") || strings.Contains(lowerName, "gallery") {
		return "gallery"
	}
	if strings.HasPrefix(lowerMime, "image/") {
		return "image"
	}
	if strings.HasPrefix(lowerMime, "video/") {
		return "video"
	}
	if strings.Contains(lowerMime, "pdf") || strings.Contains(lowerMime, "document") || strings.Contains(lowerMime, "sheet") || strings.Contains(lowerMime, "excel") || strings.Contains(lowerMime, "word") {
		return "document"
	}
	return "document"
}

