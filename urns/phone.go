package urns

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/nyaruka/gocommon/i18n"
	"github.com/nyaruka/phonenumbers/v2"
)

var nonTelCharsRegex = regexp.MustCompile(`[^0-9A-Za-z]`)
var altShortCodeRegex = regexp.MustCompile(`^[1-9][0-9]{2,5}$`)
var senderIDRegex = regexp.MustCompile(`^[0-9A-Za-z]{3,64}$`)

var ErrNotNumber = errors.New("not a possible number")

// ParsePhone returns a validated phone URN or an error.
func ParsePhone(raw string, country i18n.Country, allowShort, allowSenderID bool) (URN, error) {
	number, err := ParseNumber(raw, country, allowShort, allowSenderID)
	if err != nil {
		return "", err
	}

	return NewFromParts(Phone.Prefix, number, nil, "")
}

// ParseNumber tries to extact a possible number or shortcode from the given string, returning an error if it can't.
func ParseNumber(raw string, country i18n.Country, allowShort, allowSenderID bool) (string, error) {
	// strip all non-alphanumeric characters.. only preserving an optional leading +
	raw = strings.TrimSpace(raw)
	hasPlus := strings.HasPrefix(raw, "+")
	raw = nonTelCharsRegex.ReplaceAllString(raw, "")
	if hasPlus {
		raw = "+" + raw
	}

	number, err := parsePossibleNumber(raw, country, allowShort, allowSenderID)
	if err != nil {
		return "", err
	}

	return number, nil
}

// tries to extract a valid phone number or shortcode from the given string
func parsePossibleNumber(input string, country i18n.Country, allowShort, allowSenderID bool) (string, error) {
	// try parsing as is, only bailing if we have a junk country code
	parsed, err := phonenumbers.Parse(input, string(country))
	if country != "" && err == phonenumbers.ErrInvalidCountryCode {
		return "", err
	}

	// check to see if we have a possible number
	if err == nil {
		if phonenumbers.IsPossibleNumberWithReason(parsed) == phonenumbers.IS_POSSIBLE {
			if e164, ok := formatE164(parsed); ok {
				return e164, nil
			}
		}
	}

	// if we're sufficiently long and don't start with a 0, try adding a + prefix and re-parsing
	if len(input) >= 11 && !strings.HasPrefix(input, "0") {
		parsedWithPlus, err := phonenumbers.Parse("+"+input, string(country))
		if err == nil {
			if phonenumbers.IsPossibleNumberWithReason(parsedWithPlus) == phonenumbers.IS_POSSIBLE {
				if e164, ok := formatE164(parsedWithPlus); ok {
					return e164, nil
				}
			}
		}
	}

	// if we allow short codes and we have a country.. check for one
	if parsed != nil && country != i18n.NilCountry && allowShort {
		if phonenumbers.IsPossibleShortNumberForRegion(parsed, string(country)) {
			return phonenumbers.Format(parsed, phonenumbers.NATIONAL), nil
		}

		// it seems libphonenumber's metadata regarding shortcodes is lacking so we also accept any sequence of 3-6 digits
		// that doesn't start with a zero as a shortcode
		if altShortCodeRegex.MatchString(input) {
			return input, nil
		}
	}

	// carriers send all sorts of junk, so if we're being very lenient...
	if allowSenderID && senderIDRegex.MatchString(input) {
		return strings.ToLower(input), nil
	}

	return "", ErrNotNumber
}

// formats a number as E164, re-parsing the result to check it's stable (i.e. re-parses to itself), because numbers
// with leading zeros in their national significant number can format to an E164 form that re-parses differently,
// e.g. +82011290 re-parses with the 0 stripped as a national prefix, giving +8211290. Numbers that don't stabilize
// within a few attempts are rejected.
func formatE164(parsed *phonenumbers.PhoneNumber) (string, bool) {
	e164 := phonenumbers.Format(parsed, phonenumbers.E164)

	for range 3 {
		reparsed, err := phonenumbers.Parse(e164, "")
		if err != nil || phonenumbers.IsPossibleNumberWithReason(reparsed) != phonenumbers.IS_POSSIBLE {
			return "", false
		}
		reformatted := phonenumbers.Format(reparsed, phonenumbers.E164)
		if reformatted == e164 {
			return e164, true
		}
		e164 = reformatted
	}
	return "", false
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
