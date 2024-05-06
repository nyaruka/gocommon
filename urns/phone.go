package urns

import (
	"strconv"
	"strings"

	"github.com/nyaruka/phonenumbers"
	"github.com/pkg/errors"
)

// FromLocalPhone returns a validated tel URN
func FromLocalPhone(number string, country string) (URN, error) {
	path, err := ParsePhone(number, country)
	if err != nil {
		return NilURN, err
	}

	return NewURNFromParts(Phone, path, "", "")
}

// ToLocalPhone converts a phone URN to a local number in the given country
func ToLocalPhone(u URN, country string) string {
	_, path, _, _ := u.ToParts()

	parsed, err := phonenumbers.Parse(path, country)
	if err == nil {
		return strconv.FormatUint(parsed.GetNationalNumber(), 10)
	}
	return path
}

// ParsePhone tries to parse the given string as a phone number and if successful returns it as E164
func ParsePhone(s, country string) (string, error) {
	parsed, err := phonenumbers.Parse(s, country)
	if err != nil {
		return "", errors.Wrap(err, "unable to parse number")
	}

	if phonenumbers.IsPossibleNumberWithReason(parsed) != phonenumbers.IS_POSSIBLE {
		// if it's not a possible number, try adding a + and parsing again
		if !strings.HasPrefix(s, "+") {
			return ParsePhone("+"+s, country)
		}

		return "", errors.New("not a possible number")
	}

	return phonenumbers.Format(parsed, phonenumbers.E164), nil
}
