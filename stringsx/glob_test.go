package stringsx_test

import (
	"testing"

	"github.com/nyaruka/gocommon/stringsx"
	"github.com/stretchr/testify/assert"
)

func TestGlobMatch(t *testing.T) {
	tcs := []struct {
		input   string
		pattern string
		matches bool
	}{
		{"", "", true},
		{"hello", "", false},
		{"hello", "hello", true},
		{"HELLO", "hello", false},
		{"hellohello", "hello", false},
		{"hello", "*hello", true},
		{"hello", "*ello", true},
		{"hello", "*llo", true},
		{"hello", "*lo", true},
		{"hello", "*o", true},
		{"hello", "*", true},
		{"hello", "h*", true},
		{"hello", "he*", true},
		{"hello", "hel*", true},
		{"hello", "hello*", true},
		{"hello", "*hello*", true},
		{"hello", "*ell*", true},
		{"hello", "*e*", true},
		{"", "*", true},
		{"hello", "jam*", false},
		{"hello", "*jam", false},
		{"hello", "*j*", false},
	}

	for _, tc := range tcs {
		if tc.matches {
			assert.True(t, stringsx.GlobMatch(tc.input, tc.pattern), "expected match for %s / %s", tc.input, tc.pattern)
		} else {
			assert.False(t, stringsx.GlobMatch(tc.input, tc.pattern), "unexpected match for %s / %s", tc.input, tc.pattern)
		}
	}
}

func TestGlobSelect(t *testing.T) {
	tcs := []struct {
		input    string
		patterns []string
		selected string
	}{
		{"hello", []string{"*", "*ello", "hello", "hel*"}, "hello"},
		{"hello", []string{"*ello", "hel*"}, "*ello"},
		{"hello", []string{"*", "abc"}, "*"},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.selected, stringsx.GlobSelect(tc.input, tc.patterns...), "select mismatch for %s / %v", tc.input, tc.patterns)
	}
}
