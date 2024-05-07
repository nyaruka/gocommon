package urns

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/nyaruka/gocommon/i18n"
	"github.com/nyaruka/phonenumbers"
	"github.com/pkg/errors"
)

var nonTelCharsRegex = regexp.MustCompile(`[^0-9A-Za-z]`)

// ParsePhone returns a validated phone URN. If it can parse a possible number then that is used.. otherwise any value
// that validates as a phone URN is used.
func ParsePhone(raw string, country i18n.Country) (URN, error) {
	// strip all non-tel characters.. only preserving an optional leading +
	raw = strings.TrimSpace(raw)
	hasPlus := strings.HasPrefix(raw, "+")
	raw = nonTelCharsRegex.ReplaceAllString(raw, "")
	if hasPlus {
		raw = "+" + raw
	}

	// if we're sufficienly long and don't start with a 0 then add a +
	if len(raw) >= 11 && !strings.HasPrefix(raw, "0") {
		raw = "+" + raw
	}

	number, err := parsePhoneOrShortcode(raw, country)
	if err != nil {
		if err == phonenumbers.ErrInvalidCountryCode {
			return NilURN, errors.New("invalid country code")
		}

		return NewFromParts(Phone.Prefix, raw, "", "")
	}

	return NewFromParts(Phone.Prefix, number, "", "")
}

// tries to extract a valid phone number or shortcode from the given string
func parsePhoneOrShortcode(raw string, country i18n.Country) (string, error) {
	parsed, err := phonenumbers.Parse(raw, string(country))
	if err != nil {
		return "", err
	}

	if phonenumbers.IsPossibleNumberWithReason(parsed) == phonenumbers.IS_POSSIBLE {
		return phonenumbers.Format(parsed, phonenumbers.E164), nil
	}

	if phonenumbers.IsPossibleShortNumberForRegion(parsed, string(country)) {
		return phonenumbers.Format(parsed, phonenumbers.NATIONAL), nil
	}

	return "", errors.New("unable to parse phone number or shortcode")
}

// ToLocalPhone converts a phone URN to a local phone number.. without any leading zeros. Kinda weird but used by
// Courier where channels want the number in that format.
func ToLocalPhone(u URN, country i18n.Country) string {
	_, path, _, _ := u.ToParts()

	parsed, err := phonenumbers.Parse(path, string(country))
	if err != nil {
		return path
	}

	return strconv.FormatUint(parsed.GetNationalNumber(), 10)
}
