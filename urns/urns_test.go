package urns

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURNProperties(t *testing.T) {
	testCases := []struct {
		urn      URN
		format   string
		display  string
		rawQuery string
		query    url.Values
	}{
		{"tel:+250788383383", "0788 383 383", "", "", map[string][]string{}},
		{"twitter:85114#billy_bob", "billy_bob", "billy_bob", "", map[string][]string{}},
		{"twitter:billy_bob", "billy_bob", "", "", map[string][]string{}},
		{"tel:not-a-number", "not-a-number", "", "", map[string][]string{}},
		{"instagram:billy_bob", "billy_bob", "", "", map[string][]string{}},
		{"instagram:22114?foo=bar#foobar", "foobar", "foobar", "foo=bar", map[string][]string{"foo": {"bar"}}},
		{"facebook:ref:12345?foo=bar&foo=zap", "ref:12345", "", "foo=bar&foo=zap", map[string][]string{"foo": {"bar", "zap"}}},
		{"tel:+250788383383", "0788 383 383", "", "", map[string][]string{}},
		{"twitter:85114?foo=bar#foobar", "foobar", "foobar", "foo=bar", map[string][]string{"foo": {"bar"}}},
		{"discord:732326982863421591", "732326982863421591", "", "", map[string][]string{}},
		{"webchat:123456@foo", "123456@foo", "", "", map[string][]string{}},
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

func TestIsFacebookRef(t *testing.T) {
	testCases := []struct {
		urn           URN
		isFacebookRef bool
		facebookRef   string
	}{
		{"facebook:ref:12345", true, "12345"},
		{"facebook:12345", false, ""},

		{"tel:25078838383", false, ""},
		{"discord:732326982863421591", false, ""},
		{"discord:foo", false, ""},
	}
	for _, tc := range testCases {
		assert.Equal(t, tc.isFacebookRef, tc.urn.IsFacebookRef(), "is facebook ref mismatch for %s", tc.urn)
		assert.Equal(t, tc.facebookRef, tc.urn.FacebookRef(), "facebook ref mismatch for %s", tc.urn)
	}
}

func TestFromParts(t *testing.T) {
	testCases := []struct {
		scheme   string
		path     string
		display  string
		expected URN
		identity URN
		hasError bool
	}{
		{"tel", "+250788383383", "", URN("tel:+250788383383"), URN("tel:+250788383383"), false},
		{"twitter", "hello", "", URN("twitter:hello"), URN("twitter:hello"), false},
		{"facebook", "12345", "", URN("facebook:12345"), URN("facebook:12345"), false},
		{"instagram", "12345", "", URN("instagram:12345"), URN("instagram:12345"), false},
		{"telegram", "12345", "Jane", URN("telegram:12345#Jane"), URN("telegram:12345"), false},
		{"whatsapp", "12345", "", URN("whatsapp:12345"), URN("whatsapp:12345"), false},
		{"viber", "", "", NilURN, ":", true},
		{"discord", "732326982863421591", "", URN("discord:732326982863421591"), URN("discord:732326982863421591"), false},
		{"webchat", "12345@foo", "", URN("webchat:12345@foo"), URN("webchat:12345@foo"), false},
	}

	for _, tc := range testCases {
		urn, err := NewURNFromParts(tc.scheme, tc.path, "", tc.display)
		identity := urn.Identity()

		assert.Equal(t, tc.expected, urn, "from parts mismatch for: %s, %s, %s", tc.scheme, tc.path, tc.display)
		assert.Equal(t, tc.identity, identity, "identity mismatch for: %s, %s, %s", tc.scheme, tc.path, tc.display)

		if tc.hasError {
			assert.Error(t, err, "expected error for: %s, %s, %s", tc.scheme, tc.path, tc.display)
		} else {
			assert.NoError(t, err, "unexpected error for: %s, %s, %s", tc.scheme, tc.path, tc.display)
		}
	}
}

func TestNormalize(t *testing.T) {
	testCases := []struct {
		rawURN   URN
		country  string
		expected URN
	}{
		// valid tel numbers
		{"tel:0788383383", "RW", "tel:+250788383383"},
		{"tel: +250788383383 ", "KE", "tel:+250788383383"},
		{"tel:+250788383383", "", "tel:+250788383383"},
		{"tel:250788383383", "", "tel:+250788383383"},
		{"tel:2.50788383383E+11", "", "tel:+250788383383"},
		{"tel:2.50788383383E+12", "", "tel:+250788383383"},
		{"tel:(917)992-5253", "US", "tel:+19179925253"},
		{"tel:19179925253", "", "tel:+19179925253"},
		{"tel:+62877747666", "", "tel:+62877747666"},
		{"tel:62877747666", "ID", "tel:+62877747666"},
		{"tel:0877747666", "ID", "tel:+62877747666"},
		{"tel:07531669965", "GB", "tel:+447531669965"},
		{"tel:22658125926", "", "tel:+22658125926"},
		{"tel:263780821000", "ZW", "tel:+263780821000"},
		{"tel:+2203693333", "", "tel:+2203693333"},

		// un-normalizable tel numbers
		{"tel:12345", "RW", "tel:12345"},
		{"tel:0788383383", "", "tel:0788383383"},
		{"tel:0788383383", "ZZ", "tel:0788383383"},
		{"tel:MTN", "RW", "tel:mtn"},
		{"tel:+12345678901234567890", "", "tel:12345678901234567890"},

		// twitter handles remove @
		{"twitter: @jimmyJO", "", "twitter:jimmyjo"},
		{"twitterid:12345#@jimmyJO", "", "twitterid:12345#jimmyjo"},

		// email addresses
		{"mailto: nAme@domAIN.cOm ", "", "mailto:name@domain.com"},

		// external ids are case sensitive
		{"ext: eXterNAL123 ", "", "ext:eXterNAL123"},
	}

	for _, tc := range testCases {
		normalized := tc.rawURN.Normalize(tc.country)
		assert.Equal(t, tc.expected, normalized, "normalize mismatch for '%s' with country '%s'", tc.rawURN, tc.country)
	}
}

func TestLocalize(t *testing.T) {
	testCases := []struct {
		input    URN
		country  string
		expected URN
	}{
		// valid tel numbers
		{"tel:+250788383383", "RW", URN("tel:788383383")},
		{"tel:+447531669965", "GB", URN("tel:7531669965")},
		{"tel:+19179925253", "US", URN("tel:9179925253")},

		// un-localizable tel numbers
		{"tel:12345", "RW", URN("tel:12345")},
		{"tel:0788383383", "", URN("tel:0788383383")},
		{"tel:0788383383", "ZZ", URN("tel:0788383383")},
		{"tel:MTN", "RW", URN("tel:MTN")},

		// other schemes left as is
		{"twitter:jimmyjo", "RW", URN("twitter:jimmyjo")},
		{"twitterid:12345#jimmyjo", "RW", URN("twitterid:12345#jimmyjo")},
		{"mailto:bob@example.com", "", URN("mailto:bob@example.com")},
	}

	for _, tc := range testCases {
		localized := tc.input.Localize(tc.country)

		assert.Equal(t, tc.expected, localized, "localize mismatch for %s in country", tc.input, tc.country)
	}
}

func TestParse(t *testing.T) {
	testCases := []struct {
		input         string
		urn           URN
		expectedError string
	}{
		{"xxxx", NilURN, "path cannot be empty"},
		{"tel:", NilURN, "path cannot be empty"},
		{":xxxx", NilURN, "scheme cannot be empty"},
		{"tel:46362#rrh#gege", NilURN, "fragment component can only come after path or query components"},

		// no semantic validation
		{"xyz:abc", URN("xyz:abc"), ""},
		{"tel:****", URN("tel:****"), ""},
	}

	for _, tc := range testCases {
		actual, err := Parse(tc.input)

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
		urn           URN
		expectedError string
	}{
		{"xxxx", "scheme or path cannot be empty"}, // un-parseable URNs don't validate
		{"xyz:abc", "invalid scheme"},              // nor do unknown schemes
		{"tel:", "scheme or path cannot be empty"},

		// valid tel numbers
		{"tel:+250788383383", ""},
		{"tel:+250788383383", ""},
		{"tel:+250123", ""},
		{"tel:1337", ""},
		{"tel:1", ""}, // one digit shortcodes are a thing
		{"tel:PRIZES", ""},
		{"tel:cellbroadcastchannel50", ""},

		// invalid tel numbers
		{"tel:07883 83383", "invalid tel number"}, // can't have spaces
		{"tel:", "cannot be empty"},               // need a path

		// twitter handles
		{"twitter:jimmyjo", ""},
		{"twitter:billy_bob", ""},
		{"twitter:jimmyjo!@", "invalid twitter handle"},
		{"twitter:billy bob", "invalid twitter handle"},

		// twitterid urns
		{"twitterid:12345#jimmyjo", ""},
		{"twitterid:12345#1234567", ""},
		{"twitterid:jimmyjo#1234567", "invalid twitter id"},
		{"twitterid:123#a.!f", "invalid twitter handle"},

		// email addresses
		{"mailto:abcd+label@x.y.z.com", ""},
		{"mailto:@@@", "invalid email"},

		// facebook and telegram URN paths must be integers
		{"telegram:12345678901234567", ""},
		{"telegram:abcdef", "invalid telegram id"},
		{"facebook:12345678901234567", ""},
		{"facebook:abcdef", "invalid facebook id"},
		{"instagram:12345678901234567", ""},
		{"instagram:abcdef", "invalid instagram id"},

		// facebook refs can be anything
		{"facebook:ref:facebookRef", ""},

		// jiochat IDs
		{"jiochat:12345", ""},
		{"jiochat:123de", "invalid jiochat id"},

		// WeChat Open IDs
		{"wechat:o6_bmjrPTlm6_2sgVt7hMZOPfL2M", ""},

		// line IDs
		{"line:Uasd224", ""},
		{"line:Uqw!123", "invalid line id"},

		// viber needs to be alphanum
		{"viber:asdf12354", ""},
		{"viber:asdf!12354", "invalid viber id"},
		{"viber:xy5/5y6O81+/kbWHpLhBoA==", ""},

		// whatsapp needs to be integers
		{"whatsapp:12354", ""},
		{"whatsapp:abcde", "invalid whatsapp id"},
		{"whatsapp:+12067799294", "invalid whatsapp id"},

		// freschat has to be two uuids separated by a colon
		{"freshchat:6a2f41a3-c54c-fce8-32d2-0324e1c32e22/6a2f41a3-c54c-fce8-32d2-0324e1c32e22", ""},
		{"freshchat:6a2f41a3-c54c-fce8-32d2-0324e1c32e22", "invalid freshchat id"},
		{"freshchat:+12067799294", "invalid freshchat id"},

		{"slack:U0123ABCDEF", ""},
	}

	for _, tc := range testCases {
		err := tc.urn.Validate()
		if tc.expectedError != "" {
			assert.Error(t, err, "expected error for %s", tc.urn)

			if err != nil && !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("Failed wrong error, '%s' not found in '%s' for '%s'", tc.expectedError, err.Error(), string(tc.urn))
			}
		} else {
			assert.NoError(t, err, "unspected error validating %s", tc.urn)
		}
	}
}

func TestTelURNs(t *testing.T) {
	testCases := []struct {
		number   string
		country  string
		expected URN
		hasError bool
	}{
		{"0788383383", "RW", URN("tel:+250788383383"), false},
		{" +250788383383 ", "KE", URN("tel:+250788383383"), false},
		{"+250788383383", "", URN("tel:+250788383383"), false},
		{"250788383383", "", URN("tel:+250788383383"), false},
		{"(917)992-5253", "US", URN("tel:+19179925253"), false},
		{"(917) 992 - 5253", "US", URN("tel:+19179925253"), false},
		{"19179925253", "", URN("tel:+19179925253"), false},
		{"+62877747666", "", URN("tel:+62877747666"), false},
		{"62877747666", "ID", URN("tel:+62877747666"), false},
		{"0877747666", "ID", URN("tel:+62877747666"), false},
		{"07531669965", "GB", URN("tel:+447531669965"), false},
		{"12345", "RW", URN("tel:12345"), false},
		{"0788383383", "", URN("tel:0788383383"), false},
		{"0788383383", "ZZ", URN("tel:0788383383"), false},
		{"PRIZES", "RW", URN("tel:prizes"), false},
		{"PRIZES!", "RW", URN("tel:prizes"), false},
		{"1", "RW", URN("tel:1"), false},
		{"123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890", "RW", NilURN, true},
	}

	for _, tc := range testCases {
		urn, err := NewTelURNForCountry(tc.number, tc.country)

		if tc.hasError {
			assert.Error(t, err, "expected error for %s, %s", tc.number, tc.country)
		} else {
			assert.NoError(t, err, "unexpected error for %s, %s", tc.number, tc.country)
			assert.Equal(t, tc.expected, urn, "created URN mismatch for %s, %s", tc.number, tc.country)
		}
	}
}

func TestTelegramURNs(t *testing.T) {
	testCases := []struct {
		identifier int64
		display    string
		expected   URN
		hasError   bool
	}{
		{12345, "", URN("telegram:12345"), false},
		{12345, "Sarah", URN("telegram:12345#Sarah"), false},
	}

	for _, tc := range testCases {
		urn, err := NewTelegramURN(tc.identifier, tc.display)

		if tc.hasError {
			assert.Error(t, err, "expected error for %s", tc.identifier)
		} else {
			assert.NoError(t, err, "unexpected error for %s", tc.identifier)
			assert.Equal(t, tc.expected, urn, "created URN mismatch for %s", tc.identifier)
		}
	}
}

func TestWhatsAppURNs(t *testing.T) {
	testCases := []struct {
		identifier string
		expected   URN
		hasError   bool
	}{
		{"12345", URN("whatsapp:12345"), false},
		{"+12345", NilURN, true},
	}

	for _, tc := range testCases {
		urn, err := NewWhatsAppURN(tc.identifier)

		if tc.hasError {
			assert.Error(t, err, "expected error for %s", tc.identifier)
		} else {
			assert.NoError(t, err, "unexpected error for %s", tc.identifier)
			assert.Equal(t, tc.expected, urn, "created URN mismatch for %s", tc.identifier)
		}
	}
}

func TestFacebookURNs(t *testing.T) {
	testCases := []struct {
		identifier string
		expected   URN
		hasError   bool
	}{
		{"12345", URN("facebook:12345"), false},
		{"invalid", NilURN, true},
	}

	for _, tc := range testCases {
		urn, err := NewFacebookURN(tc.identifier)

		if tc.hasError {
			assert.Error(t, err, "expected error for %s", tc.identifier)
		} else {
			assert.NoError(t, err, "unexpected error for %s", tc.identifier)
			assert.Equal(t, tc.expected, urn, "created URN mismatch for %s", tc.identifier)
		}
	}
}

func TestInstagramURNs(t *testing.T) {
	testCases := []struct {
		identifier string
		expected   URN
		hasError   bool
	}{
		{"12345", URN("instagram:12345"), false},
		{"invalid", NilURN, true},
	}

	for _, tc := range testCases {
		urn, err := NewInstagramURN(tc.identifier)

		if tc.hasError {
			assert.Error(t, err, "expected error for %s", tc.identifier)
		} else {
			assert.NoError(t, err, "unexpected error for %s", tc.identifier)
			assert.Equal(t, tc.expected, urn, "created URN mismatch for %s", tc.identifier)
		}
	}
}

func TestFirebaseURNs(t *testing.T) {
	testCases := []struct {
		identifier string
		expected   URN
		hasError   bool
	}{
		{"12345", URN("fcm:12345"), false},
		{"asdf", URN("fcm:asdf"), false},
		{"", NilURN, true},
	}

	for _, tc := range testCases {
		urn, err := NewFirebaseURN(tc.identifier)

		if tc.hasError {
			assert.Error(t, err, "expected error for %s", tc.identifier)
		} else {
			assert.NoError(t, err, "unexpected error for %s", tc.identifier)
			assert.Equal(t, tc.expected, urn, "created URN mismatch for %s", tc.identifier)
		}
	}
}

func TestDiscordURNs(t *testing.T) {
	testCases := []struct {
		identifier string
		expected   URN
		hasError   bool
	}{
		{"732326982863421591", URN("discord:732326982863421591"), false},
		{"notadiscordID", URN("discord:notadiscordID"), true},
		{"", NilURN, true},
	}
	for _, tc := range testCases {
		urn, err := NewDiscordURN(tc.identifier)
		if tc.hasError {
			assert.Error(t, err, "expected error for %s", tc.identifier)
		} else {
			assert.NoError(t, err, "expected error for %s", tc.identifier)
			assert.Equal(t, tc.expected, urn, "created URN mismatch for %s", tc.identifier)
		}
	}
}

func TestWebChatURNs(t *testing.T) {
	testCases := []struct {
		identifier string
		expected   URN
		hasError   bool
	}{
		{"123456@foo", URN("webchat:123456@foo"), false},
		{"matricula:123456@foo", URN("webchat:matricula:123456@foo"), false},
		{"123456", URN("webchat:123456@foo"), true},
	}

	for _, tc := range testCases {
		urn, err := NewWebChatURN(tc.identifier)
		if tc.hasError {
			assert.Error(t, err, "expected error for %s", tc.identifier)
		} else {
			assert.NoError(t, err, "expected error for %s", tc.identifier)
			assert.Equal(t, tc.expected, urn, "created URN mismatch for %s", tc.identifier)
		}
	}
}

func BenchmarkValidTel(b *testing.B) {
	for n := 0; n < b.N; n++ {
		NewTelURNForCountry("2065551212", "US")
	}
}

func BenchmarkInvalidTel(b *testing.B) {
	for n := 0; n < b.N; n++ {
		NewTelURNForCountry("notnumber", "US")
	}
}
