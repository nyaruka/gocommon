package urns

import (
	"testing"

	"github.com/nyaruka/gocommon/i18n"
	"github.com/stretchr/testify/assert"
)

func TestParsePhone(t *testing.T) {
	testCases := []struct {
		input    string
		country  i18n.Country
		expected URN
	}{
		{" 0788383383 ", "RW", "tel:+250788383383"},
		{"+250788383383 ", "RW", "tel:+250788383383"}, // already has country code and leading +
		{"250788383383 ", "RW", "tel:+250788383383"},  // already has country code and no leading +
		{"+250788383383 ", "KE", "tel:+250788383383"}, // already has a different country code
		{"(917)992-5253", "US", "tel:+19179925253"},
		{"800-CABBAGE", "US", "tel:+18002222243"},
		{"+62877747666", "ID", "tel:+62877747666"},
		{"0877747666", "ID", "tel:+62877747666"},
		{"07531669965", "GB", "tel:+447531669965"},
		{"263780821000", "ZW", "tel:+263780821000"},
		{"254791541111", "US", "tel:+254791541111"}, // international but missing + and wrong country

		{"1", "RW", "tel:1"},
		{"123456", "RW", "tel:123456"},
		{"mtn", "RW", "tel:mtn"},
		{"!mtn!", "RW", "tel:mtn"}, // non tel chars stripped

		{"0788383383", "ZZ", NilURN}, // invalid country code
		{"1234567890123456789012345678901234567890123456789012345678901234567890123456789", "RW", NilURN}, // too long
	}

	for i, tc := range testCases {
		urn, err := ParsePhone(tc.input, tc.country)

		if tc.expected == NilURN {
			assert.Error(t, err, "%d: expected error for %s, %s", i, tc.input, tc.country)
		} else {
			assert.NoError(t, err, "%d: unexpected error for %s, %s", i, tc.input, tc.country)
			assert.Equal(t, tc.expected, urn, "%d: created URN mismatch for %s, %s", i, tc.input, tc.country)
		}
	}
}

func TestParsePhoneOrShortcode(t *testing.T) {
	tcs := []struct {
		input    string
		country  i18n.Country
		expected string
	}{
		{"+250788123123", "", "+250788123123"},    // international number fine without country
		{"+250 788 123-123", "", "+250788123123"}, // still fine if not E164 formatted

		{"0788123123", "RW", "+250788123123"},     // country code added
		{" (206)555-1212 ", "US", "+12065551212"}, // punctiation removed
		{"800-CABBAGE", "US", "+18002222243"},     // letters converted to numbers
		{"12065551212", "US", "+12065551212"},     // country code but no +
		{"10000", "US", "10000"},                  // valid short code for US

		{"5912705", "US", ""}, // is only possible as a local number so ignored
	}

	for _, tc := range tcs {
		parsed, err := parsePhoneOrShortcode(tc.input, tc.country)

		if tc.expected != "" {
			assert.NoError(t, err, "unexpected error for '%s'", tc.input)
			assert.Equal(t, tc.expected, parsed, "result mismatch for '%s'", tc.input)
		} else {
			assert.Error(t, err, "expected error for '%s'", tc.input)
		}
	}
}
func TestToLocalPhone(t *testing.T) {
	tcs := []struct {
		urn      URN
		country  i18n.Country
		expected string
	}{
		{"tel:+250788123123", "", "788123123"},
		{"tel:123123", "", "123123"},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.expected, ToLocalPhone(tc.urn, tc.country), "local mismatch for '%s'", tc.urn)
	}
}
