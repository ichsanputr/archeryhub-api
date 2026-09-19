package utils

import (
	"testing"
)

func TestDetectMimeType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"banner.PNG", "image/png"},
		{"graphic.webp", "image/webp"},
		{"guidebook.pdf", "application/pdf"},
		{"rules.doc", "application/msword"},
		{"rules.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"sheet.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"video.mp4", "video/mp4"},
		{"unknown.xyz", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := DetectMimeType(tt.filename)
			if got != tt.expected {
				t.Errorf("DetectMimeType(%q) = %q, expected %q", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestDetermineMediaCategory(t *testing.T) {
	tests := []struct {
		caption  string
		filename string
		mimeType string
		expected string
	}{
		{"Official Banner", "banner-123.jpg", "image/jpeg", "banner"},
		{"Club Logo", "logo.png", "image/png", "logo"},
		{"Guidebook Sleman 2026", "guidebook.pdf", "application/pdf", "document"},
		{"Juknis Event", "juknis.pdf", "application/pdf", "document"},
		{"Podium Photo", "gallery-img.jpg", "image/jpeg", "gallery"},
		{"Random Photo", "image123.jpg", "image/jpeg", "image"},
		{"Scoresheet", "result.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "document"},
	}

	for _, tt := range tests {
		t.Run(tt.caption, func(t *testing.T) {
			got := DetermineMediaCategory(tt.caption, tt.filename, tt.mimeType)
			if got != tt.expected {
				t.Errorf("DetermineMediaCategory(%q, %q, %q) = %q, expected %q", tt.caption, tt.filename, tt.mimeType, got, tt.expected)
			}
		})
	}
}
