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

	// check if this is possible number, excluding local-only options
	if phonenumbers.IsPossibleNumberWithReason(parsed) != phonenumbers.IS_POSSIBLE {
		return "", errors.New("not a possible number")
	}

	return phonenumbers.Format(parsed, phonenumbers.E164), nil
}
