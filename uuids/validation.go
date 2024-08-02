package uuids

import (
	"regexp"
	"strconv"
)

var (
	Regex = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-([1-7])[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}`)

	onlyRegex = regexp.MustCompile(`^` + Regex.String() + `$`)
)

// Is returns whether the given string contains only a valid v4 UUID
func Is(s string) bool {
	return onlyRegex.MatchString(s)
}

func Version(s string) int {
	m := onlyRegex.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	v, _ := strconv.Atoi(m[1])
	return v
}
