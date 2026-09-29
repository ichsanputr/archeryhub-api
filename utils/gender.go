package utils

import "strings"

// NormalizeGender standardizes any gender input into "male", "female", or ""
func NormalizeGender(gender string) string {
	g := strings.ToLower(strings.TrimSpace(gender))
	switch g {
	case "male", "m", "men", "man", "boy", "putra", "laki-laki", "laki", "pria", "l":
		return "male"
	case "female", "f", "women", "woman", "girl", "putri", "perempuan", "wanita", "p", "w":
		return "female"
	default:
		return ""
	}
}
