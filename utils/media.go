package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// mediaBaseURL returns the base URL for media links based on STAGE:
// - development: http://localhost:PORT (default 8001)
// - production: https://api.archeryhub.id
func mediaBaseURL() string {
	stage := os.Getenv("STAGE")
	if stage == "" {
		stage = os.Getenv("ENV")
	}
	if stage == "production" {
		return "https://api.archeryhub.id"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}
	return fmt.Sprintf("http://localhost:%s", port)
}

// MaskMediaURL converts a filename stored in the database to a full URL.
// If the input is already a full URL (starts with http), it returns it as is.
// If the input is empty or null, it returns an empty string.
// Base URL is derived from STAGE: development → localhost, production → api.archeryhub.id.
func MaskMediaURL(filename string) string {
	if filename == "" {
		return ""
	}

	// If it's already a full URL, return it
	if strings.HasPrefix(filename, "http://") || strings.HasPrefix(filename, "https://") {
		return filename
	}

	baseURL := mediaBaseURL()

	// Clean the filename (extract base if it was a path)
	cleanName := filepath.Base(filename)

	return fmt.Sprintf("%s/api/v1/media/%s", baseURL, cleanName)
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
