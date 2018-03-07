package urns

import (
	"testing"
)

func TestParseURNAndBack(t *testing.T) {
	testCases := []struct {
		input    string
		scheme   string
		path     string
		query    string
		fragment string
		hasError bool
	}{
		{input: "scheme:path", scheme: "scheme", path: "path"},
		{input: "scheme:path#frag", scheme: "scheme", path: "path", fragment: "frag"},
		{input: "scheme:path?query", scheme: "scheme", path: "path", query: "query"},
		{input: "scheme:path?query#frag", scheme: "scheme", path: "path", query: "query", fragment: "frag"},
		{input: "scheme:pa%25th?qu%23ery#fra%3Fg", scheme: "scheme", path: "pa%th", query: "qu#ery", fragment: "fra?g"},

		{input: "scheme:path:morepath", scheme: "scheme", path: "path:morepath"},
		{input: "scheme:path:morepath?foo=bar", scheme: "scheme", path: "path:morepath", query: "foo=bar"},

		// can't be empty
		{input: "", hasError: true},

		// can't single part
		{input: "xyz", hasError: true},

		// can't omit scheme or path
		{input: "scheme:", hasError: true},
		{input: ":path", hasError: true},

		// can't have multiple queries or fragments
		{input: "scheme:path?query?query", hasError: true},
		{input: "scheme:path#frag#frag", hasError: true},

		// can't have query after fragment
		{input: "scheme:path#frag?query", hasError: true},
	}
	for _, tc := range testCases {
		p, err := parseURN(tc.input)

		if err != nil {
			if !tc.hasError {
				t.Errorf("Failed parsing URN, got unxpected error: %s", err.Error())
			}
		} else {
			if p.scheme != tc.scheme || p.path != tc.path || p.query != tc.query || p.fragment != tc.fragment {
				t.Errorf("Failed parsing URN, got %s|%s|%s|%s, expected %s|%s|%s|%s for '%s'", p.scheme, p.path, p.query, p.fragment, tc.scheme, tc.path, tc.query, tc.fragment, tc.input)
			} else {
				backToStr := p.String()

				if backToStr != tc.input {
					t.Errorf("Failed stringifying URN, got '%s', expected '%s' for %s|%s|%s|%s", backToStr, tc.input, tc.scheme, tc.path, tc.query, tc.fragment)
				}
			}
		}

	}
}
