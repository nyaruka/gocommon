package urns_test

import (
	"testing"

	"github.com/nyaruka/gocommon/urns"
	"github.com/stretchr/testify/assert"
)

func TestFromLocalPhone(t *testing.T) {
	testCases := []struct {
		number   string
		country  string
		expected urns.URN
		hasError bool
	}{
		{"tel:0788383383", "RW", "tel:+250788383383", false},
		{"tel: +250788383383 ", "KE", "tel:+250788383383", false}, // already has country code
		{"tel:(917)992-5253", "US", "tel:+19179925253", false},
		{"tel:800-CABBAGE", "US", "tel:+18002222243", false},
		{"tel:+62877747666", "ID", "tel:+62877747666", false},
		{"tel:0877747666", "ID", "tel:+62877747666", false},
		{"tel:07531669965", "GB", "tel:+447531669965", false},
		{"tel:263780821000", "ZW", "tel:+263780821000", false},

		{"0788383383", "ZZ", urns.NilURN, true}, // invalid country code
		{"1", "RW", urns.NilURN, true},
		{"123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890", "RW", urns.NilURN, true},
	}

	for i, tc := range testCases {
		urn, err := urns.FromLocalPhone(tc.number, tc.country)

		if tc.hasError {
			assert.Error(t, err, "%d: expected error for %s, %s", i, tc.number, tc.country)
		} else {
			assert.NoError(t, err, "%d: unexpected error for %s, %s", i, tc.number, tc.country)
			assert.Equal(t, tc.expected, urn, "%d: created URN mismatch for %s, %s", i, tc.number, tc.country)
		}
	}
}

func TestParsePhone(t *testing.T) {
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
			parsed, err := urns.ParsePhone(tc.input, tc.country)
			assert.NoError(t, err, "unexpected error for '%s'", tc.input)
			assert.Equal(t, parsed, tc.parsed, "result mismatch for '%s'", tc.input)
		} else {
			_, err := urns.ParsePhone(tc.input, tc.country)
			assert.Error(t, err, "expected error for '%s'", tc.input)
		}
	}
}
