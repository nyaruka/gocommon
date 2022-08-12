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
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit-len(ending)]) + ending
}
