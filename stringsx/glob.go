package stringsx

import (
	"sort"
	"strings"
)

// GlobMatch does very simple * based glob matching where * can be the entire pattern or start or the end or both.
func GlobMatch(s, pattern string) bool {
	if pattern == "" {
		return s == ""
	}
	if pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		return strings.Contains(s, pattern[1:len(pattern)-1])
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(s, pattern[1:])
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(s, pattern[0:len(pattern)-1])
	}

	return s == pattern
}

// GlobSelect returns the most specific matching pattern from the given set.
func GlobSelect(s string, patterns ...string) string {
	matching := make([]string, 0, len(patterns))
	for _, p := range patterns {
		if GlobMatch(s, p) {
			matching = append(matching, p)
		}
	}

	if len(matching) == 0 {
		return ""
	}

	// return the longest pattern excluding * chars
	sort.SliceStable(matching, func(i, j int) bool {
		return len(strings.Trim(matching[i], "*")) > len(strings.Trim(matching[j], "*"))
	})
	return matching[0]
}
