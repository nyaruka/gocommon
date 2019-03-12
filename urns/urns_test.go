package urns

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsFacebookRef(t *testing.T) {
	testCases := []struct {
		urn           URN
		IsFacebookRef bool
		FacebookRef   string
	}{
		{"facebook:ref:12345", true, "12345"},
		{"facebook:12345", false, ""},
		{"tel:25078838383", false, ""},
	}
	for _, tc := range testCases {
		if tc.urn.IsFacebookRef() != tc.IsFacebookRef {
			t.Errorf("Mismatch facebook ref for %s, expected %v", tc.urn, tc.IsFacebookRef)
		}

		if tc.urn.FacebookRef() != tc.FacebookRef {
			t.Errorf("Mismatch facebook ref for %s, expected %v", tc.urn, tc.IsFacebookRef)
		}
	}
}

func TestQuery(t *testing.T) {
	testCases := []struct {
		urn      URN
		rawQuery string
		query    url.Values
	}{
		{"facebook:ref:12345?foo=bar&foo=zap", "foo=bar&foo=zap", map[string][]string{"foo": {"bar", "zap"}}},
		{"tel:+250788383383", "", map[string][]string{}},
		{"twitter:85114?foo=bar#foobar", "foo=bar", map[string][]string{"foo": {"bar"}}},
	}
	for _, tc := range testCases {
		if tc.urn.RawQuery() != tc.rawQuery {
			t.Errorf("Mismatch raw query for '%s', expected '%s', got '%s'", tc.urn, tc.rawQuery, tc.urn.RawQuery())
		}
		query, _ := tc.urn.Query()
		assert.Equal(t, query, tc.query, "Mismatch raw query for '%s', expected '%s', got '%s'", tc.urn, tc.rawQuery, tc.urn.RawQuery())
	}
}

func TestDisplay(t *testing.T) {
	testCases := []struct {
		urn     URN
		display string
	}{
		{"facebook:ref:12345", ""},
		{"tel:+250788383383", ""},
		{"twitter:85114#foobar", "foobar"},
	}
	for _, tc := range testCases {
		if tc.urn.Display() != tc.display {
			t.Errorf("Mismatch display for %s, expected %s, got %s", tc.urn, tc.display, tc.urn.Display())
		}
	}
}

func TestFormat(t *testing.T) {
	testCases := []struct {
		urn    URN
		format string
	}{
		{"tel:+250788383383", "0788 383 383"},
		{"twitter:85114#billy_bob", "billy_bob"},
		{"twitter:billy_bob", "billy_bob"},
		{"tel:not-a-number", "not-a-number"},
	}
	for _, tc := range testCases {
		if tc.urn.Format() != tc.format {
			t.Errorf("Mismatch format for %s, expected %s, got %s", tc.urn, tc.format, tc.urn.Format())
		}
	}
}

func TestFromParts(t *testing.T) {
	testCases := []struct {
		scheme   string
		path     string
		display  string
		expected string
		identity string
		hasError bool
	}{
		{"tel", "+250788383383", "", "tel:+250788383383", "tel:+250788383383", false},
		{"twitter", "hello", "", "twitter:hello", "twitter:hello", false},
		{"facebook", "12345", "", "facebook:12345", "facebook:12345", false},
		{"telegram", "12345", "Jane", "telegram:12345#Jane", "telegram:12345", false},
		{"whatsapp", "12345", "", "whatsapp:12345", "whatsapp:12345", false},
		{"viber", "", "", "", ":", true},
	}

	for _, tc := range testCases {
		urn, err := NewURNFromParts(tc.scheme, tc.path, "", tc.display)
		if urn != URN(tc.expected) {
			t.Errorf("Failed creating urn, got '%s', expected '%s' for '%s:%s'", urn, tc.expected, tc.scheme, tc.path)
		}

		identity := urn.Identity()
		if identity != URN(tc.identity) {
			t.Errorf("Failed creating urn, got identity '%s', expected '%s' for '%s:%s'", identity, tc.identity, tc.scheme, tc.path)
		}

		if err != nil != tc.hasError {
			t.Errorf("Failed creating urn, got error: %s when expecting: %s", err.Error(), tc.expected)
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

		// un-normalizable tel numbers
		{"tel:12345", "RW", "tel:12345"},
		{"tel:0788383383", "", "tel:0788383383"},
		{"tel:0788383383", "ZZ", "tel:0788383383"},
		{"tel:MTN", "RW", "tel:mtn"},

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
		if normalized != tc.expected {
			t.Errorf("Failed normalizing urn, got '%s', expected '%s' for '%s' in country %s", normalized, tc.expected, string(tc.rawURN), tc.country)
		}
	}
}

func TestLocalize(t *testing.T) {
	testCases := []struct {
		input    URN
		country  string
		expected URN
	}{
		// valid tel numbers
		{"tel:+250788383383", "RW", "tel:788383383"},
		{"tel:+447531669965", "GB", "tel:7531669965"},
		{"tel:+19179925253", "US", "tel:9179925253"},

		// un-localizable tel numbers
		{"tel:12345", "RW", "tel:12345"},
		{"tel:0788383383", "", "tel:0788383383"},
		{"tel:0788383383", "ZZ", "tel:0788383383"},
		{"tel:MTN", "RW", "tel:MTN"},

		// other schemes left as is
		{"twitter:jimmyjo", "RW", "twitter:jimmyjo"},
		{"twitterid:12345#jimmyjo", "RW", "twitterid:12345#jimmyjo"},
		{"mailto:bob@example.com", "", "mailto:bob@example.com"},
	}

	for _, tc := range testCases {
		localized := tc.input.Localize(tc.country)
		if localized != tc.expected {
			t.Errorf("Failed localizing urn, got '%s', expected '%s' for '%s' in country %s", localized, tc.expected, string(tc.input), tc.country)
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
		{"tel:PRIZES", ""},
		{"tel:12", ""},
		{"tel:1", ""},

		// invalid tel numbers
		{"tel:07883 83383", "invalid tel number"},       // can't have spaces
		{"tel:12345678901234567", "invalid tel number"}, // too long

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
	}

	for _, tc := range testCases {
		err := tc.urn.Validate()
		if tc.expectedError != "" {
			if err == nil {
				t.Errorf("Failed wrong validation, expected error with '%s' for '%s'", tc.expectedError, string(tc.urn))
			}

			if err != nil && !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("Failed wrong error, '%s' not found in '%s' for '%s'", tc.expectedError, err.Error(), string(tc.urn))
			}
		}

		if err != nil && tc.expectedError == "" {
			t.Errorf("Failed validating urn, got %s, expected no error for '%s'", err.Error(), string(tc.urn))
		}
	}
}

func TestTelURNs(t *testing.T) {
	testCases := []struct {
		number   string
		country  string
		expected string
		hasError bool
	}{
		{"0788383383", "RW", "tel:+250788383383", false},
		{" +250788383383 ", "KE", "tel:+250788383383", false},
		{"+250788383383", "", "tel:+250788383383", false},
		{"250788383383", "", "tel:+250788383383", false},
		{"(917)992-5253", "US", "tel:+19179925253", false},
		{"(917) 992 - 5253", "US", "tel:+19179925253", false},
		{"19179925253", "", "tel:+19179925253", false},
		{"+62877747666", "", "tel:+62877747666", false},
		{"62877747666", "ID", "tel:+62877747666", false},
		{"0877747666", "ID", "tel:+62877747666", false},
		{"07531669965", "GB", "tel:+447531669965", false},
		{"12345", "RW", "tel:12345", false},
		{"0788383383", "", "tel:0788383383", false},
		{"0788383383", "ZZ", "tel:0788383383", false},
		{"PRIZES", "RW", "tel:prizes", false},
		{"PRIZES!", "RW", "tel:prizes", false},
		{"1", "RW", "tel:1", false},
		{"", "RW", "", true},
		{"-", "RW", "", true},
		{"123456789012345678901234567890", "RW", "", true},
	}

	for _, tc := range testCases {
		urn, err := NewTelURNForCountry(tc.number, tc.country)
		if urn != URN(tc.expected) {
			t.Errorf("Failed tel parsing, got '%s', expected '%s' for '%s:%s'", urn, tc.expected, tc.number, tc.country)
		}
		if err != nil != tc.hasError {
			t.Errorf("Failed tel parsing, got error: %s when expecting: %s", err.Error(), tc.expected)
		}
	}
}

func TestTelegramURNs(t *testing.T) {
	testCases := []struct {
		identifier int64
		display    string
		expected   string
		hasError   bool
	}{
		{12345, "", "telegram:12345", false},
		{12345, "Sarah", "telegram:12345#Sarah", false},
	}

	for _, tc := range testCases {
		urn, err := NewTelegramURN(tc.identifier, tc.display)
		if urn != URN(tc.expected) {
			t.Errorf("Failed Telegram URN, got '%s', expected '%s' for '%d'", urn, tc.expected, tc.identifier)
		}
		if err != nil != tc.hasError {
			t.Errorf("Failed Telegram URN, got error: %s when expecting: %s", err.Error(), tc.expected)
		}
	}
}

func TestWhatsAppURNs(t *testing.T) {
	testCases := []struct {
		identifier string
		expected   string
		hasError   bool
	}{
		{"12345", "whatsapp:12345", false},
		{"+12345", "", true},
	}

	for _, tc := range testCases {
		urn, err := NewWhatsAppURN(tc.identifier)
		if urn != URN(tc.expected) {
			t.Errorf("Failed WhatsApp URN, got '%s', expected '%s' for '%s'", urn, tc.expected, tc.identifier)
		}
		if err != nil != tc.hasError {
			t.Errorf("Failed WhatsApp URN, got error: %s when expecting: %s", err.Error(), tc.expected)
		}
	}
}

func TestFacebookURNs(t *testing.T) {
	testCases := []struct {
		identifier string
		expected   string
		hasError   bool
	}{
		{"12345", "facebook:12345", false},
		{"invalid", "", true},
	}

	for _, tc := range testCases {
		urn, err := NewFacebookURN(tc.identifier)
		if urn != URN(tc.expected) {
			t.Errorf("Failed Facebook URN, got '%s', expected '%s' for '%s'", urn, tc.expected, tc.identifier)
		}
		if err != nil != tc.hasError {
			t.Errorf("Failed Facebook URN, got error: %s when expecting: %s", err.Error(), tc.expected)
		}
	}
}

func TestFirebaseURNs(t *testing.T) {
	testCases := []struct {
		identifier string
		expected   string
		hasError   bool
	}{
		{"12345", "fcm:12345", false},
		{"asdf", "fcm:asdf", false},
		{"", "", true},
	}

	for _, tc := range testCases {
		urn, err := NewFirebaseURN(tc.identifier)
		if urn != URN(tc.expected) {
			t.Errorf("Failed Firebase URN, got '%s', expected '%s' for '%s'", urn, tc.expected, tc.identifier)
		}
		if err != nil != tc.hasError {
			t.Errorf("Failed Firebase URN, got error: %s when expecting: %s", err.Error(), tc.expected)
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
