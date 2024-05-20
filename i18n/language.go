package i18n

import (
	"database/sql/driver"
	"fmt"

	"github.com/nyaruka/null/v2"
	"golang.org/x/text/language"
)

// Language is holds a an ISO-639-3 language code.
type Language string

// ISO639_1 returns the ISO-639-1 2-letter code for this language if it has one
func (l Language) ISO639_1() string {
	base, err := language.ParseBase(string(l))
	if err != nil {
		return ""
	}
	code := base.String()

	// not all languages have a 2-letter code
	if len(code) != 2 {
		return ""
	}
	return code
}

// NilLanguage represents our nil, or unknown language
var NilLanguage = Language("")

// ParseLanguage returns a new Language for the passed in language string, or an error if not found
func ParseLanguage(lang string) (Language, error) {
	if len(lang) != 3 {
		return NilLanguage, fmt.Errorf("iso-639-3 codes must be 3 characters, got: %s", lang)
	}

	base, err := language.ParseBase(lang)
	if err != nil {
		return NilLanguage, fmt.Errorf("unrecognized language code: %s", lang)
	}

	return Language(base.ISO3()), nil
}

// Place nicely with NULLs if persisting to a database or JSON
func (l *Language) Scan(value any) error         { return null.ScanString(value, l) }
func (l Language) Value() (driver.Value, error)  { return null.StringValue(l) }
func (l Language) MarshalJSON() ([]byte, error)  { return null.MarshalString(l) }
func (l *Language) UnmarshalJSON(b []byte) error { return null.UnmarshalString(b, l) }
