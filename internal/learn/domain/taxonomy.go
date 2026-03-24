package domain

import "strings"

// Slugify converts a name to a URL-safe slug. [06-learn §12]
// Used for custom subject slug generation.
func Slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	// Remove anything that isn't alphanumeric or hyphen.
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
		}
	}
	return b.String()
}
