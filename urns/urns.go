package urns

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	maxPathLength    = 255
	maxDisplayLength = 255
)

// IsValidScheme checks whether the provided scheme is valid
func IsValidScheme(scheme string) bool {
	_, valid := schemes[scheme]
	return valid
}

// Schemes returns the valid URN schemes
func Schemes() []string {
	return schemePrefixes
}

// URN represents a Universal Resource Name, we use this for contact identifiers like phone numbers etc..
type URN string

// NilURN is our constant for nil URNs
const NilURN = URN("")

// returns a new URN for the given scheme, path, query and display
func newFromParts(scheme, path, query, display string) URN {
	u := &parsedURN{
		scheme:   scheme,
		path:     path,
		query:    query,
		fragment: display,
	}
	return URN(u.String())
}

// NewFromParts returns a validated URN for the given scheme, path, query and display
func NewFromParts(scheme *Scheme, path, query, display string) (URN, error) {
	urn := newFromParts(scheme.Prefix, path, query, display)

	if err := urn.Validate(); err != nil {
		return NilURN, err
	}
	return urn, nil
}

// New returns a validated URN for the given scheme and path
func New(scheme *Scheme, path string) (URN, error) {
	return NewFromParts(scheme, path, "", "")
}

// Parse parses a URN from the given string. The returned URN is only guaranteed to be structurally valid.
func Parse(s string) (URN, error) {
	parsed, err := parseURN(s)
	if err != nil {
		return NilURN, err
	}

	return URN(parsed.String()), nil
}

// ToParts splits the URN into scheme, path and display parts
func (u URN) ToParts() (string, string, string, string) {
	parsed, err := parseURN(string(u))
	if err != nil {
		return "", string(u), "", ""
	}

	return parsed.scheme, parsed.path, parsed.query, parsed.fragment
}

// Normalize normalizes the URN into it's canonical form and should be performed before URN comparisons
func (u URN) Normalize() URN {
	scheme, path, query, display := u.ToParts()
	s := schemes[scheme]

	path = strings.TrimSpace(path)

	if s != nil && s.Normalize != nil {
		path = s.Normalize(path)
	}

	return newFromParts(scheme, path, query, display)
}

// Validate returns whether this URN is considered valid
func (u URN) Validate() error {
	scheme, path, _, display := u.ToParts()

	if scheme == "" || path == "" {
		return fmt.Errorf("scheme or path cannot be empty")
	}

	if !IsValidScheme(scheme) {
		return fmt.Errorf("unknown URN scheme")
	}

	if len(path) > maxPathLength {
		return fmt.Errorf("path component too long")
	}

	s := schemes[scheme]
	if s.Validate != nil && !s.Validate(path) {
		return fmt.Errorf("invalid path component")
	}

	if len(display) > maxDisplayLength {
		return fmt.Errorf("display component too long")
	}

	return nil
}

// Scheme returns the scheme portion for the URN
func (u URN) Scheme() string {
	scheme, _, _, _ := u.ToParts()
	return scheme
}

// Path returns the path portion for the URN
func (u URN) Path() string {
	_, path, _, _ := u.ToParts()
	return path
}

// Display returns the display portion for the URN (if any)
func (u URN) Display() string {
	_, _, _, display := u.ToParts()
	return display
}

// RawQuery returns the unparsed query portion for the URN (if any)
func (u URN) RawQuery() string {
	_, _, query, _ := u.ToParts()
	return query
}

// Query returns the parsed query portion for the URN (if any)
func (u URN) Query() (url.Values, error) {
	_, _, query, _ := u.ToParts()
	return url.ParseQuery(query)
}

// Identity returns the URN with any query or display attributes stripped
func (u URN) Identity() URN {
	scheme, path, _, _ := u.ToParts()
	return newFromParts(scheme, path, "", "")
}

// String returns the string representation of this URN
func (u URN) String() string { return string(u) }

// Format formats this URN as a human friendly string
func (u URN) Format() string {
	scheme, path, _, display := u.ToParts()

	// display always takes precedence
	if display != "" {
		return display
	}

	s := schemes[scheme]
	if s != nil && s.Format != nil {
		return s.Format(path)
	}

	return path
}
