package gsm7_test

import (
	"strings"
	"testing"

	"github.com/nyaruka/gocommon/gsm7"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecode(t *testing.T) {
	tcs := []struct {
		encoded string
		decoded string
	}{
		{"basic", "basic"},
		{"\x00\x0Fspecial", "@åspecial"},
		{"\x1B\x28extended\x1B\x29", "{extended}"},
		{"\x20space", " space"},
	}
	for _, tc := range tcs {
		assert.Equal(t, tc.decoded, gsm7.Decode([]byte(tc.encoded)))
		assert.Equal(t, []byte(tc.encoded), gsm7.Encode(tc.decoded))
	}

	assert.Equal(t, "?invalid?", gsm7.Decode([]byte("\x1B\x50invalid\x1B\x50")))
	assert.Equal(t, "?toobig", gsm7.Decode([]byte("\x81toobig")))

	assert.Equal(t, []byte("hi!\x20\x3F"), gsm7.Encode("hi! ☺"))
}

func TestValid(t *testing.T) {
	tcs := []struct {
		str   string
		valid bool
	}{
		{" basic", true},
		{"@åspecial", true},
		{"{extended}", true},
		{"hi! ☺", false},
	}
	for _, tc := range tcs {
		assert.Equal(t, tc.valid, gsm7.IsValid(tc.str), tc.str)
	}
}

func TestSubstitutions(t *testing.T) {
	tcs := []struct {
		str string
		exp string
	}{
		{" basic", " basic"},
		{"êxtended", "extended"},
		{"“quoted”", `"quoted"`},
		{"\x09tab", " tab"},
	}
	for _, tc := range tcs {
		assert.Equal(t, tc.exp, gsm7.ReplaceSubstitutions(tc.str), tc.str)
	}
}

func TestSegments(t *testing.T) {
	// utility pads
	tenChars := "0123456789"
	unicodeTenChars := "☺123456789"
	extendedTenChars := "[123456789"
	fiftyChars := tenChars + tenChars + tenChars + tenChars + tenChars
	hundredChars := fiftyChars + fiftyChars
	unicode := "☺"

	tcs := []struct {
		Text     string
		Segments int
	}{
		{"", 1},
		{strings.Repeat(" ", 160), 1},
		{strings.Repeat(" ", 161), 2},
		{strings.Repeat("\f", 80), 1},
		{strings.Repeat("\f", 81), 2},
		{"hello", 1},
		{"“word”", 1},
		{hundredChars + fiftyChars + tenChars, 1},
		{hundredChars + fiftyChars + tenChars + "Z", 2},
		{hundredChars + fiftyChars + extendedTenChars, 2},
		{hundredChars + hundredChars + hundredChars + "123456", 2},
		{hundredChars + hundredChars + hundredChars + "1234567", 3},
		{fiftyChars + "zZ" + unicode, 1},
		{fiftyChars + tenChars + unicodeTenChars, 1},
		{fiftyChars + tenChars + unicodeTenChars + "z", 2},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.Segments, gsm7.Segments(tc.Text), "unexpected num of segments for: %s", tc.Text)
	}
}
