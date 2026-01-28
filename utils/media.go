package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MaskMediaURL converts a filename stored in the database to a full URL.
// If the input is already a full URL (starts with http), it returns it as is.
// If the input is empty or null, it returns an empty string.
func MaskMediaURL(filename string) string {
	if filename == "" {
		return ""
	}

	// If it's already a full URL, return it
	if strings.HasPrefix(filename, "http://") || strings.HasPrefix(filename, "https://") {
		return filename
	}

	// Get base URL from environment
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8001"
	}

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
