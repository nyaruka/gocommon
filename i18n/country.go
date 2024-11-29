package i18n

import (
	"database/sql/driver"
	"regexp"

	"github.com/nyaruka/null/v2"
	"github.com/nyaruka/phonenumbers"
)

// Country is a ISO 3166-1 alpha-2 country code
type Country string

// NilCountry represents our nil, or unknown country
var NilCountry = Country("")

var countryPattern = regexp.MustCompile(`^[A-Z][A-Z]$`)

// DeriveCountryFromTel attempts to derive a country code (e.g. RW) from a phone number
func DeriveCountryFromTel(number string) Country {
	parsed, err := phonenumbers.Parse(number, "")
	if err != nil {
		return ""
	}

	region := phonenumbers.GetRegionCodeForNumber(parsed)

	// check this is an actual country code and not a special "region" like 001
	if countryPattern.MatchString(region) {
		return Country(region)
	}

	return NilCountry
}

// Place nicely with NULLs if persisting to a database or JSON
func (c *Country) Scan(value any) error         { return null.ScanString(value, c) }
func (c Country) Value() (driver.Value, error)  { return null.StringValue(c) }
func (c Country) MarshalJSON() ([]byte, error)  { return null.MarshalString(c) }
func (c *Country) UnmarshalJSON(b []byte) error { return null.UnmarshalString(b, c) }
