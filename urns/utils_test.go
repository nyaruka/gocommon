package urns_test

import (
	"testing"

	"github.com/nyaruka/gocommon/urns"
	"github.com/stretchr/testify/assert"
)

func TestParseNumber(t *testing.T) {
	tcs := []struct {
		input   string
		country string
		parsed  string
	}{
		{"+250788123123", "", "+250788123123"},    // international number fine without country
		{"+250 788 123-123", "", "+250788123123"}, // fine if not E164 formatted
		{"0788123123", "RW", "+250788123123"},
		{"206 555 1212", "US", "+12065551212"},
		{"12065551212", "US", "+12065551212"}, // country code but no +
		{"5912705", "US", ""},                 // is only possible as a local number so ignored
		{"10000", "US", ""},
	}

	for _, tc := range tcs {
		if tc.parsed != "" {
			parsed, err := urns.ParseNumber(tc.input, tc.country)
			assert.NoError(t, err, "unexpected error for '%s'", tc.input)
			assert.Equal(t, parsed, tc.parsed, "result mismatch for '%s'", tc.input)
		} else {
			_, err := urns.ParseNumber(tc.input, tc.country)
			assert.Error(t, err, "expected error for '%s'", tc.input)
		}
	}
}
