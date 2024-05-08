package urns_test

import (
	"testing"

	"github.com/nyaruka/gocommon/i18n"
	"github.com/nyaruka/gocommon/urns"
	"github.com/stretchr/testify/assert"
)

func TestParsePhone(t *testing.T) {
	testCases := []struct {
		input       string
		country     i18n.Country
		expectedURN urns.URN
		expectedErr string
	}{
		{"+250788123123", "", "tel:+250788123123", ""},    // international number fine without country
		{"+250 788 123-123", "", "tel:+250788123123", ""}, // still fine if not E164 formatted
		{"250788123123", "", "tel:+250788123123", ""},     // still fine without leading + because it's long enough

		{" 0788383383 ", "RW", "tel:+250788383383", ""},   // country code added
		{"+250788383383 ", "RW", "tel:+250788383383", ""}, // already has country code and leading +
		{"250788383383 ", "RW", "tel:+250788383383", ""},  // already has country code and no leading +
		{"+250788383383 ", "KE", "tel:+250788383383", ""}, // already has a different country code
		{"(917)992-5253", "US", "tel:+19179925253", ""},   // punctuation removed
		{"800-CABBAGE", "US", "tel:+18002222243", ""},     // vanity numbers converted to digits
		{"+62877747666", "ID", "tel:+62877747666", ""},
		{"812111005611", "ID", "tel:+62812111005611", ""},
		{"0877747666", "ID", "tel:+62877747666", ""},
		{"07531669965", "GB", "tel:+447531669965", ""},
		{"263780821000", "ZW", "tel:+263780821000", ""},
		{"254791541111", "US", "tel:+254791541111", ""}, // international but missing + and wrong country

		{"1234", "US", "tel:1234", ""},
		{"12345", "US", "tel:12345", ""},
		{"123", "RW", "tel:123", ""},
		{"8080", "EC", "tel:8080", ""},

		// inputs that fail parsing by libphonenumber
		{"1", "RW", "", "the phone number supplied is not a number"},
		{"mtn", "RW", "", "the phone number supplied is not a number"},
		{"1234", "", "", "invalid country code"}, // can't parse short without country

		// inputa that fail checking for possible number or shortcode
		{"99", "EC", "", "not a possible number"},
		{"567-1234", "US", "", "not a possible number"}, // only dialable locally

		{"0788383383", "ZZ", "", "invalid country code"}, // invalid country code
		{"1234567890123456789012345678901234567890123456789012345678901234567890123456789", "RW", "", "the string supplied is too long to be a phone number"}, // too long
	}

	for i, tc := range testCases {
		urn, err := urns.ParsePhone(tc.input, tc.country)

		if tc.expectedErr != "" {
			if assert.EqualError(t, err, tc.expectedErr, "%d: expected error for %s, %s", i, tc.input, tc.country) {
				assert.Equal(t, urns.NilURN, urn)
			}
		} else {
			if assert.NoError(t, err, "%d: unexpected error for %s, %s", i, tc.input, tc.country) {
				assert.Equal(t, tc.expectedURN, urn, "%d: URN mismatch for %s, %s", i, tc.input, tc.country)
			}
		}
	}
}

func TestParseNumber(t *testing.T) {
	testCases := []struct {
		input       string
		country     i18n.Country
		allowShort  bool
		expectedNum string
		expectedErr string
	}{
		{"+250788123123", "", true, "+250788123123", ""},
		{"+250788123123", "", false, "+250788123123", ""},
		{"0788123123", "RW", true, "+250788123123", ""},
		{"0788123123", "RW", false, "+250788123123", ""},
		{"123", "RW", true, "123", ""},
		{"123", "RW", false, "", "not a possible number"},
	}

	for i, tc := range testCases {
		num, err := urns.ParseNumber(tc.input, tc.country, tc.allowShort)

		if tc.expectedErr != "" {
			if assert.EqualError(t, err, tc.expectedErr, "%d: expected error for %s, %s", i, tc.input, tc.country) {
				assert.Equal(t, "", num)
			}
		} else {
			if assert.NoError(t, err, "%d: unexpected error for %s, %s", i, tc.input, tc.country) {
				assert.Equal(t, tc.expectedNum, num, "%d: URN mismatch for %s, %s", i, tc.input, tc.country)
			}
		}
	}
}

func TestToLocalPhone(t *testing.T) {
	tcs := []struct {
		urn      urns.URN
		country  i18n.Country
		expected string
	}{
		{"tel:+250788123123", "", "788123123"},
		{"tel:123123", "", "123123"},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.expected, urns.ToLocalPhone(tc.urn, tc.country), "local mismatch for '%s'", tc.urn)
	}
}
