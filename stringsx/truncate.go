package stringsx

// Truncate truncates the given string to ensure it's less than limit characters
func Truncate(s string, limit int) string {
	return truncate(s, limit, "")
}

// TruncateEllipsis truncates the given string and adds ellipsis where the input is cut
func TruncateEllipsis(s string, limit int) string {
	return truncate(s, limit, "...")
}

func truncate(s string, limit int, ending string) string {
	if limit < 0 {
		limit = 0
	}
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	// not enough room to fit the ending, so just hard-truncate to the limit
	if limit <= len(ending) {
		return string(runes[:limit])
	}
	return string(runes[:limit-len(ending)]) + ending
}
