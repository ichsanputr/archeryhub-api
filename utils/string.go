package utils

import (
	"fmt"
	"strings"
	"unicode"
)

// DiceBearAvatar returns a DiceBear avatar URL using the given seed.
// The seed ensures the same user always gets the same avatar.
func DiceBearAvatar(seed string) string {
	return fmt.Sprintf("https://api.dicebear.com/9.x/avataaars/svg?seed=%s", seed)
}

// CleanUsername cleans a username by converting to lowercase, 
// replacing spaces/underscores with dashes, and removing non-alphanumeric characters.
func CleanUsername(username string) string {
	// Convert to lowercase
	username = strings.ToLower(username)
	
	// Replace spaces and underscores with dashes
	username = strings.ReplaceAll(username, " ", "-")
	username = strings.ReplaceAll(username, "_", "-")
	
	// Remove multiple dashes
	for strings.Contains(username, "--") {
		username = strings.ReplaceAll(username, "--", "-")
	}
	
	// Build result with only allowed characters (a-z, 0-9, -)
	var result strings.Builder
	for _, r := range username {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	
	// Trim dashes from start and end
	return strings.Trim(result.String(), "-")
}

// IsValidUsername checks if a username contains only allowed characters.
func IsValidUsername(username string) bool {
	if len(username) == 0 {
		return false
	}
	
	for _, r := range username {
		if !unicode.IsDigit(r) && !unicode.IsLower(r) && r != '-' {
			return false
		}
	}
	
	// Must not start or end with a dash
	if username[0] == '-' || username[len(username)-1] == '-' {
		return false
	}
	
	return true
}
