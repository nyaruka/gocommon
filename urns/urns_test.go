package urns_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/nyaruka/gocommon/urns"
	"github.com/stretchr/testify/assert"
)

func TestIsValidScheme(t *testing.T) {
	assert.True(t, urns.IsValidScheme("tel"))
	assert.False(t, urns.IsValidScheme("xyz"))
}

func TestURNProperties(t *testing.T) {
	testCases := []struct {
		urn      urns.URN
		format   string
		display  string
		rawQuery string
		query    url.Values
	}{
		{"tel:+250788383383", "0788 383 383", "", "", map[string][]string{}},
		{"tel:+250788383383#my-phone", "my-phone", "my-phone", "", map[string][]string{}},
		{"twitter:85114#billy_bob", "billy_bob", "billy_bob", "", map[string][]string{}},
		{"twitter:billy_bob", "billy_bob", "", "", map[string][]string{}},
		{"tel:not-a-number", "not-a-number", "", "", map[string][]string{}},
		{"instagram:billy_bob", "billy_bob", "", "", map[string][]string{}},
		{"instagram:22114?foo=bar#foobar", "foobar", "foobar", "foo=bar", map[string][]string{"foo": {"bar"}}},
		{"facebook:ref:12345?foo=bar&foo=zap", "ref:12345", "", "foo=bar&foo=zap", map[string][]string{"foo": {"bar", "zap"}}},
		{"tel:+250788383383", "0788 383 383", "", "", map[string][]string{}},
		{"twitter:85114?foo=bar#foobar", "foobar", "foobar", "foo=bar", map[string][]string{"foo": {"bar"}}},
		{"webchat:123456789012345678901234", "123456789012345678901234", "", "", map[string][]string{}},
	}
	for _, tc := range testCases {
		assert.Equal(t, string(tc.urn), tc.urn.String())
		assert.Equal(t, tc.format, tc.urn.Format(), "format mismatch for %s", tc.urn)
		assert.Equal(t, tc.display, tc.urn.Display(), "display mismatch for %s", tc.urn)
		assert.Equal(t, tc.rawQuery, tc.urn.RawQuery(), "raw query mismatch for %s", tc.urn)

		query, _ := tc.urn.Query()
		assert.Equal(t, tc.query, query, "parsed query mismatch for %s", tc.urn)
	}
}

func TestNewFromParts(t *testing.T) {
	testCases := []struct {
		scheme   *urns.Scheme
		path     string
		query    url.Values
		display  string
		expected urns.URN
		identity urns.URN
		hasError bool
	}{
		{urns.External, " Aa123 \t\n", nil, "", "ext:Aa123", "ext:Aa123", false}, // whitespace trimmed
		{urns.External, "12345", url.Values{"id": []string{"2"}}, "cool", "ext:12345?id=2#cool", "ext:12345", false},
		{urns.Email, "BoB@NYARUKA.com", nil, "", "mailto:bob@nyaruka.com", "mailto:bob@nyaruka.com", false}, // emails lowercased
		{urns.Phone, "+250788383383", nil, "", "tel:+250788383383", "tel:+250788383383", false},
		{urns.Twitter, "1234", nil, "bob", "twitter:1234#bob", "twitter:1234", false},
		{urns.Facebook, "12345", nil, "", "facebook:12345", "facebook:12345", false},
		{urns.Instagram, "12345", nil, "", "instagram:12345", "instagram:12345", false},
		{urns.Telegram, "12345", nil, "Jane", "telegram:12345#Jane", "telegram:12345", false},
		{urns.WhatsApp, "12345", nil, "", "whatsapp:12345", "whatsapp:12345", false},
		{urns.WebChat, "123456789012345678901234", nil, "", "webchat:123456789012345678901234", "webchat:123456789012345678901234", false},
		{urns.WebChat, "123456789012345678901234", nil, "bob@nyaruka.com", "webchat:123456789012345678901234#bob@nyaruka.com", "webchat:123456789012345678901234", false},

		{urns.Viber, "", nil, "", urns.NilURN, ":", true},
	}

	for _, tc := range testCases {
		urn, err := urns.NewFromParts(tc.scheme.Prefix, tc.path, tc.query, tc.display)
		identity := urn.Identity()

		assert.Equal(t, tc.expected, urn, "from parts mismatch for: %s, %s, %s", tc.scheme, tc.path, tc.display)
		assert.Equal(t, tc.identity, identity, "identity mismatch for: %s, %s, %s", tc.scheme, tc.path, tc.display)

		if tc.hasError {
			assert.Error(t, err, "expected error for: %s, %s, %s", tc.scheme, tc.path, tc.display)
		} else {
			assert.NoError(t, err, "unexpected error for: %s, %s, %s", tc.scheme, tc.path, tc.display)
		}
	}

	// test New shortcut
	urn, err := urns.New(urns.Phone, "+250788383383")
	assert.NoError(t, err)
	assert.Equal(t, "tel:+250788383383", urn.String())
}

func TestNormalize(t *testing.T) {
	testCases := []struct {
		rawURN   urns.URN
		expected urns.URN
	}{
		// tel numbers re-parsed
		{"tel:+250788383383", "tel:+250788383383"},
		{"tel:250788383383", "tel:+250788383383"}, // + added
		{"tel:1(800)CABBAGE", "tel:+18002222243"},
		{"tel:+62877747666", "tel:+62877747666"},
		{"tel:+2203693333", "tel:+2203693333"},

		// or left as they are if not valid
		{"tel:000", "tel:000"},
		{"tel:mtn", "tel:mtn"},
		{"tel:+12345678901234567890", "tel:+12345678901234567890"},

		// twitter handles remove @
		{"twitter: @jimmyJO", "twitter:jimmyjo"},
		{"twitterid:12345#jimmyJO", "twitterid:12345#jimmyJO"},

		// email addresses
		{"mailto: nAme@domAIN.cOm ", "mailto:name@domain.com"},

		// external ids are case sensitive
		{"ext: eXterNAL123 ", "ext:eXterNAL123"},
	}

	for _, tc := range testCases {
		normalized := tc.rawURN.Normalize()
		assert.Equal(t, tc.expected, normalized, "normalize mismatch for '%s'", tc.rawURN)

		// check we're idempotent
		normalized = normalized.Normalize()
		assert.Equal(t, tc.expected, normalized, "re-normalize mismatch for '%s'", tc.rawURN)
	}
}

func TestParse(t *testing.T) {
	testCases := []struct {
		input         string
		urn           urns.URN
		expectedError string
	}{
		{"xxxx", urns.NilURN, "path cannot be empty"},
		{"tel:", urns.NilURN, "path cannot be empty"},
		{":xxxx", urns.NilURN, "scheme cannot be empty"},
		{"tel:46362#rrh#gege", urns.NilURN, "fragment component can only come after path or query components"},

		// no semantic validation
		{"xyz:abc", "xyz:abc", ""},
		{"tel:****", "tel:****", ""},
	}

	for _, tc := range testCases {
		actual, err := urns.Parse(tc.input)

		if tc.expectedError != "" {
			assert.EqualError(t, err, tc.expectedError, "error mismatch for %s", tc.input)
		} else {
			assert.NoError(t, err, "unexpected error for %s", tc.input)
			assert.Equal(t, tc.urn, actual, "parsed URN mismatch for %s", tc.input)
		}
	}
}

func TestValidate(t *testing.T) {
	testCases := []struct {
		urn           urns.URN
		expectedError string
	}{
		{"xxxx", "scheme or path cannot be empty"}, // un-parseable URNs don't validate
		{"xyz:abc", "unknown URN scheme"},          // nor do unknown schemes
		{"tel:", "scheme or path cannot be empty"},

		// valid tel numbers
		{"tel:+250788383383", ""},
		{"tel:+250788383383", ""},
		{"tel:250123", ""},
		{"tel:1337", ""},
		{"tel:1", ""},
		{"tel:prizes", ""},

		// invalid tel numbers
		{"tel:", "cannot be empty"},                   // need a path
		{"tel:07883 83383", "invalid path component"}, // can't have spaces
		{"tel:PRIZES", "invalid path component"},      // we allow letters but we always lowercase
		{"tel:+123", "invalid path component"},        // too short to have a +
		{"tel:+prizes", "invalid path component"},     // sender ids don't have +

		// twitter handles
		{"twitter:jimmyjo", ""},
		{"twitter:billy_bob", ""},
		{"twitter:jimmyjo!@", "invalid path component"},
		{"twitter:billy bob", "invalid path component"},

		// twitterid urns
		{"twitterid:12345#jimmyjo", ""},
		{"twitterid:12345#1234567", ""},
		{"twitterid:jimmyjo#1234567", "invalid path component"},

		// email addresses
		{"mailto:abcd+label@x.y.z.com", ""},
		{"mailto:@@@", "invalid path component"},

		// facebook and telegram URN paths must be integers
		{"telegram:12345678901234567", ""},
		{"telegram:abcdef", "invalid path component"},
		{"facebook:12345678901234567", ""},
		{"facebook:abcdef", "invalid path component"},
		{"instagram:12345678901234567", ""},
		{"instagram:abcdef", "invalid path component"},

		// facebook refs can be anything
		{"facebook:ref:facebookRef", ""},

		// jiochat IDs
		{"jiochat:12345", ""},
		{"jiochat:123de", "invalid path component"},

		// WeChat Open IDs
		{"wechat:o6_bmjrPTlm6_2sgVt7hMZOPfL2M", ""},

		// line IDs
		{"line:Uasd224", ""},
		{"line:Uqw!123", "invalid path component"},

		// viber needs to be alphanum
		{"viber:asdf12354", ""},
		{"viber:asdf!12354", "invalid path component"},
		{"viber:xy5/5y6O81+/kbWHpLhBoA==", ""},

		// whatsapp needs to be integers
		{"whatsapp:12354", ""},
		{"whatsapp:abcde", "invalid path component"},
		{"whatsapp:+12067799294", "invalid path component"},

		// freschat has to be two uuids separated by a colon
		{"freshchat:6a2f41a3-c54c-fce8-32d2-0324e1c32e22/6a2f41a3-c54c-fce8-32d2-0324e1c32e22", ""},
		{"freshchat:6a2f41a3-c54c-fce8-32d2-0324e1c32e22", "invalid path component"},
		{"freshchat:+12067799294", "invalid path component"},

		{"slack:U0123ABCDEF", ""},

		{"webchat:aA3456789012345678901234", ""},
		{"webchat:aA3456789012345678901234:bob@nyaruka.com", ""},
		{"webchat:1234567890123456789", "invalid path component"},
		{"webchat:12345678901234567890123$", "invalid path component"},
		{"webchat:aA3456789012345678901234:@@$", "invalid path component"},
	}

	for _, tc := range testCases {
		err := tc.urn.Validate()
		if tc.expectedError != "" {
			assert.Error(t, err, "expected error for %s", tc.urn)

			if err != nil && !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("Failed wrong error, '%s' not found in '%s' for '%s'", tc.expectedError, err.Error(), string(tc.urn))
			}
		} else {
			assert.NoError(t, err, "unexpected error validating %s", tc.urn)
		}
	}
}
