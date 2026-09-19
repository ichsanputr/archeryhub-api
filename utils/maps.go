package utils

import (
	"regexp"
	"strings"
)

var (
	iframeSrcRegex   = regexp.MustCompile(`(?i)<iframe[^>]+src=["']([^"']+)["']`)
	gmapsDomainRegex = regexp.MustCompile(`(?i)^(https?:\/\/)?(www\.)?(google\.[a-z.]+|maps\.google\.[a-z.]+|maps\.app\.goo\.gl|goo\.gl)\/.*`)
)

// NormalizeGmapsEmbed extracts the embed URL if the input is an <iframe> HTML tag,
// or returns the trimmed string if it's already a URL.
func NormalizeGmapsEmbed(input *string) *string {
	if input == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*input)
	if trimmed == "" {
		return nil
	}

	// 1. Check if input contains an <iframe> tag and extract src
	if matches := iframeSrcRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
		src := strings.TrimSpace(matches[1])
		return &src
	}

	// 2. Return trimmed URL
	return &trimmed
}
