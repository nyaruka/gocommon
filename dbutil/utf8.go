package dbutil

import "strings"

// ToValidUTF8 replaces invalid UTF-8 sequences with � characters and also strips NULL characters, which whilst being
// valid UTF-8, can't be saved to PostgreSQL.
func ToValidUTF8(s string) string {
	return strings.ReplaceAll(strings.ToValidUTF8(s, `�`), "\x00", "")
}
