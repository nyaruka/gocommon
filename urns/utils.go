package urns

import (
	"github.com/nyaruka/phonenumbers"

	"github.com/pkg/errors"
)

// ParseNumber tries to parse the given string as a phone number and if successful returns it as E164
func ParseNumber(s, country string) (string, error) {
	parsed, err := phonenumbers.Parse(s, country)
	if err != nil {
		return "", errors.Wrap(err, "unable to parse number")
	}

	// if it looks like a possible number, return it formatted as E164
	if phonenumbers.IsPossibleNumber(parsed) {
		return phonenumbers.Format(parsed, phonenumbers.E164), nil
	}

	return "", errors.New("not a possible number")
}
