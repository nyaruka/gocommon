package uuids

import "regexp"

var (
	V4Regex = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}`)
	V7Regex = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}`)

	V4OnlyRegex = regexp.MustCompile(`^` + V4Regex.String() + `$`)
	V7OnlyRegex = regexp.MustCompile(`^` + V7Regex.String() + `$`)
)

// IsV4 returns whether the given string contains only a valid v4 UUID
func IsV4(s string) bool {
	return V4OnlyRegex.MatchString(s)
}

// IsV7 returns whether the given string contains only a valid v7 UUID
func IsV7(s string) bool {
	return V7OnlyRegex.MatchString(s)
}
