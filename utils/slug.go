package utils

import (
	"regexp"
	"strings"
	"unicode"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// IsValidSlug reports whether s matches the standard Beast challenge slug format:
// lowercase alphanumeric words separated by single hyphens (e.g. 'baby-overflow').
func IsValidSlug(s string) bool {
	return slugRegex.MatchString(s)
}

// Slugify converts an arbitrary string (such as a challenge title) into a valid URL-safe slug.
// E.g., "Buffer Overflow 101!" -> "buffer-overflow-101".
func Slugify(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	lastHyphen := true

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastHyphen = false
		} else if !lastHyphen {
			b.WriteRune('-')
			lastHyphen = true
		}
	}

	return strings.TrimRight(b.String(), "-")
}
