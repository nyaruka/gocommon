package urns_test

import (
	"testing"

	"github.com/nyaruka/gocommon/i18n"
	"github.com/nyaruka/gocommon/urns"
	"github.com/stretchr/testify/assert"
)

func TestParsePhone(t *testing.T) {
	testCases := []struct {
		input         string
		country       i18n.Country
		allowShort    bool
		allowSenderID bool
		expectedURN   urns.URN
		expectedErr   string
	}{
		{"+250788123123", "", true, true, "tel:+250788123123", ""},    // international number fine without country
		{"+250 788 123-123", "", true, true, "tel:+250788123123", ""}, // still fine if not E164 formatted
		{"250788123123", "", true, true, "tel:+250788123123", ""},     // still fine without leading + because it's long enough
		{" 0788383383 ", "RW", true, true, "tel:+250788383383", ""},   // country code added
		{"+250788383383 ", "RW", true, true, "tel:+250788383383", ""}, // already has country code and leading +
		{"250788383383 ", "RW", true, true, "tel:+250788383383", ""},  // already has country code and no leading +
		{"+250788383383 ", "KE", true, true, "tel:+250788383383", ""}, // already has a different country code
		{"(917)992-5253", "US", true, true, "tel:+19179925253", ""},   // punctuation removed
		{"800-CABBAGE", "US", true, true, "tel:+18002222243", ""},     // vanity numbers converted to digits
		{"+62877747666", "ID", true, true, "tel:+62877747666", ""},
		{"812111005611", "ID", true, true, "tel:+62812111005611", ""},
		{"0877747666", "ID", true, true, "tel:+62877747666", ""},
		{"07531669965", "GB", true, true, "tel:+447531669965", ""},
		{"263780821000", "ZW", true, true, "tel:+263780821000", ""},
		{"254791541111", "US", true, true, "tel:+254791541111", ""}, // international but missing + and wrong country

		{"123456", "US", true, false, "tel:123456", ""},
		{"12345", "US", true, false, "tel:12345", ""},
		{"1234", "US", true, false, "tel:1234", ""},
		{"1234", "US", false, false, "", "not a possible number"},
		{"1234", "", true, false, "", "not a possible number"}, // can't parse short without country
		{"123", "RW", true, false, "tel:123", ""},
		{"8080", "EC", true, false, "tel:8080", ""},

		{"PRIZES", "RW", true, false, "", "not a possible number"},

		// inputs that fail parsing by libphonenumber
		{"1", "RW", true, false, "", "not a possible number"},

		// input that fails checking for possible number or shortcode
		{"99", "EC", true, false, "", "not a possible number"},
		{"567-1234", "US", true, false, "", "not a possible number"}, // only dialable locally

		{"0788383383", "ZZ", true, false, "", "invalid country code"},                                                                       // invalid country code
		{"1234567890123456789012345678901234567890123456789012345678901234567890123456789", "RW", true, false, "", "not a possible number"}, // too long
	}

	for i, tc := range testCases {
		urn, err := urns.ParsePhone(tc.input, tc.country, tc.allowShort, tc.allowSenderID)

		if tc.expectedErr != "" {
			if assert.EqualError(t, err, tc.expectedErr, "%d: expected error for %s, %s", i, tc.input, tc.country) {
				assert.Equal(t, urns.NilURN, urn)
			}
		} else {
			if assert.NoError(t, err, "%d: unexpected error for %s, %s", i, tc.input, tc.country) {
				assert.Equal(t, tc.expectedURN, urn, "%d: URN mismatch for %s, %s", i, tc.input, tc.country)

				// check the returned URN is valid
				assert.Nil(t, urn.Validate())
			}
		}
	}
}

func TestParseNumber(t *testing.T) {
	testCases := []struct {
		input         string
		country       i18n.Country
		allowShort    bool
		allowSenderID bool
		expectedNum   string
		expectedErr   string
	}{
		{"+250788123123", "", false, false, "+250788123123", ""},
		{"+250788123123", "", true, false, "+250788123123", ""},
		{"+250788123123", "", true, true, "+250788123123", ""},

		{"0788123123", "RW", false, false, "+250788123123", ""},
		{"0788123123", "RW", true, false, "+250788123123", ""},
		{"0788123123", "RW", true, true, "+250788123123", ""},

		{"123", "RW", false, false, "", "not a possible number"},
		{"123", "RW", true, false, "123", ""},
		{"123", "RW", true, true, "123", ""},

		{"PRIZES", "RW", false, false, "", "not a possible number"},
		{"PRIZES", "RW", true, false, "", "not a possible number"},
		{"PRIZES", "RW", true, true, "prizes", ""},
	}

	for i, tc := range testCases {
		num, err := urns.ParseNumber(tc.input, tc.country, tc.allowShort, tc.allowSenderID)

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
