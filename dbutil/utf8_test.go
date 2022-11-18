package dbutil_test

import (
	"testing"

	"github.com/nyaruka/gocommon/dbutil"
	"github.com/stretchr/testify/assert"
)

func TestToValidUTF8(t *testing.T) {
	tcs := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"hello\nworld", "hello\nworld"},
		{"hello\xc3\x28world", "hello�(world"},      // invalid 2 octet sequence
		{"hello\xa0\xa1world", "hello�world"},       // invalid sequence identifier
		{"hello\xe2\x28\xa1world", "hello�(�world"}, // invalid 3 octet sequence
		{"\u0000hello world\x00", "hello world"},    // null character
	}

	for _, tc := range tcs {
		actual := dbutil.ToValidUTF8(tc.input)
		assert.Equal(t, tc.expected, actual)
	}
}
