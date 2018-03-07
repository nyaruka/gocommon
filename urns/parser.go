package urns

import (
	"bytes"
	"fmt"
	"strings"
)

// Simple URN parser loosely based on RFC2141 (https://www.ietf.org/rfc/rfc2141.txt)

const (
	stateScheme = iota
	statePath
	stateQuery
	stateFragment
)

var escapes = map[rune]string{
	'#': `%23`,
	'%': `%25`,
	// '/': `%2F`,  can't enable this until we fix our URNs with slashes
	'?': `%3F`,
}

type parsedURN struct {
	scheme   string
	path     string
	query    string
	fragment string
}

func (u *parsedURN) String() string {
	s := escape(u.scheme) + ":" + escape(u.path)
	if u.query != "" {
		s += "?" + escape(u.query)
	}
	if u.fragment != "" {
		s += "#" + escape(u.fragment)
	}
	return s
}

func parseURN(urn string) (*parsedURN, error) {
	state := stateScheme

	buffers := map[int]*bytes.Buffer{
		stateScheme:   {},
		statePath:     {},
		stateQuery:    {},
		stateFragment: {},
	}

	for _, c := range urn {
		if c == ':' {
			if state == stateScheme {
				state = statePath
				continue
			}
		} else if c == '?' {
			if state == statePath {
				state = stateQuery
				continue
			} else {
				return nil, fmt.Errorf("query component can only come after path component")
			}
		} else if c == '#' {
			if state == statePath || state == stateQuery {
				state = stateFragment
				continue
			} else {
				return nil, fmt.Errorf("fragment component can only come after path or query components")
			}
		}

		buffers[state].WriteRune(c)
	}

	if buffers[stateScheme].Len() == 0 || buffers[statePath].Len() == 0 {
		return nil, fmt.Errorf("must contain at least scheme and path")
	}

	return &parsedURN{
		scheme:   unescape(buffers[stateScheme].String()),
		path:     unescape(buffers[statePath].String()),
		query:    unescape(buffers[stateQuery].String()),
		fragment: unescape(buffers[stateFragment].String()),
	}, nil
}

func escape(s string) string {
	b := bytes.Buffer{}
	for _, c := range s {
		esc, isEsc := escapes[c]
		if isEsc {
			b.WriteString(esc)
		} else {
			b.WriteRune(c)
		}
	}
	return b.String()
}

func unescape(s string) string {
	for ch, esc := range escapes {
		s = strings.Replace(s, esc, string(ch), -1)
	}
	return s
}
