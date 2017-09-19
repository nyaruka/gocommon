package urns

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/nyaruka/phonenumbers"
)

const (
	// TelScheme is the scheme used for telephone numbers
	TelScheme string = "tel"

	// FacebookScheme is the scheme used for Facebook identifiers
	FacebookScheme string = "facebook"

	// TelegramScheme is the scheme used for Telegram identifiers
	TelegramScheme string = "telegram"

	// TwitterScheme is the scheme used for Twitter handles
	TwitterScheme string = "twitter"

	// TwitterIDScheme is the scheme used for Twitter user ids
	TwitterIDScheme string = "twitterid"

	// ViberScheme is the scheme used for Viber identifiers
	ViberScheme string = "viber"

	// LineScheme is the scheme used for LINE identifiers
	LineScheme string = "line"

	// JiochatScheme is the scheme used for Jiochat identifiers
	JiochatScheme string = "jiochat"

	// EmailScheme is the scheme used for email addresses
	EmailScheme string = "mailto"

	// FCMScheme is the scheme used for Firebase Cloud Messaging identifiers
	FCMScheme string = "fcm"

	// ExternalScheme is the scheme used for externally defined identifiers
	ExternalScheme string = "ext"
)

var validSchemes = map[string]bool{
	TelScheme:       true,
	FacebookScheme:  true,
	TelegramScheme:  true,
	TwitterScheme:   true,
	TwitterIDScheme: true,
	ViberScheme:     true,
	LineScheme:      true,
	JiochatScheme:   true,
	EmailScheme:     true,
	FCMScheme:       true,
	ExternalScheme:  true,
}

// IsValidScheme checks whether the provided scheme is valid
func IsValidScheme(scheme string) bool {
	_, valid := validSchemes[scheme]
	return valid
}

var nonTelCharsRegex = regexp.MustCompile(`[^0-9a-z]`)
var twitterHandleRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,15}$`)
var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+$`)
var viberRegex = regexp.MustCompile(`^[a-zA-Z0-9_=]{1,16}$`) // (this is a guess)
var allDigitsRegex = regexp.MustCompile(`^[0-9]+$`)

// URN represents a Universal Resource Name, we use this for contact identifiers like phone numbers etc..
type URN string

// NewTelURNForCountry returns a URN for the passed in telephone number and country code ("US")
func NewTelURNForCountry(number string, country string) URN {
	return newURN(TelScheme, normalizeNumber(number, country), "")
}

// NewTelegramURN returns a URN for the passed in telegram identifier
func NewTelegramURN(identifier int64, display string) URN {
	return newURN(TelegramScheme, strconv.FormatInt(identifier, 10), display)
}

// NewURNFromParts returns a new URN for the given scheme, path and display
func NewURNFromParts(scheme string, path string, display string) (URN, error) {
	scheme = strings.ToLower(scheme)
	if !validSchemes[scheme] {
		return NilURN, fmt.Errorf("invalid scheme '%s'", scheme)
	}

	return newURN(scheme, path, display), nil
}

// private utility method to create a URN from a scheme and path
func newURN(scheme string, path string, display string) URN {
	urn := fmt.Sprintf("%s:%s", scheme, path)
	if display != "" {
		urn = fmt.Sprintf("%s#%s", urn, strings.ToLower(display))
	}
	return URN(urn)
}

// ToParts splits the URN into scheme, path and display parts
func (u URN) ToParts() (string, string, string) {
	parts := strings.SplitN(string(u), ":", 2)
	if len(parts) != 2 {
		return "", string(u), ""
	}

	scheme := parts[0]
	path := parts[1]
	display := ""

	pathParts := strings.SplitN(path, "#", 2)
	if len(pathParts) == 2 {
		path = pathParts[0]
		display = pathParts[1]
	}

	return scheme, path, display
}

// Normalize normalizes the URN into it's canonical form and should be performed before URN comparisons
func (u URN) Normalize(country string) (URN, error) {
	scheme, path, display := u.ToParts()
	normPath := strings.TrimSpace(path)

	switch scheme {
	case TelScheme:
		normPath = normalizeNumber(normPath, country)

	case TwitterScheme:
		// Twitter handles are case-insensitive, so we always store as lowercase
		normPath = strings.ToLower(normPath)

		// strip @ prefix if provided
		if strings.HasPrefix(normPath, "@") {
			normPath = normPath[1:len(normPath)]
		}

	case TwitterIDScheme:
		if display != "" {
			display = strings.ToLower(strings.TrimSpace(display))
			if display != "" && strings.HasPrefix(display, "@") {
				display = display[1:len(display)]
			}
		}

	case EmailScheme:
		normPath = strings.ToLower(normPath)
	}

	return NewURNFromParts(scheme, normPath, display)
}

// Validate returns whether this URN is considered valid
func (u URN) Validate(country string) bool {
	scheme, path, display := u.ToParts()
	if scheme == "" || path == "" {
		return false
	}

	switch scheme {
	case TelScheme:
		// validate is possible phone number
		parsed, err := phonenumbers.Parse(path, country)
		if err != nil {
			return false
		}
		return phonenumbers.IsPossibleNumber(parsed)

	case TwitterScheme:
		// validate twitter URNs look like handles
		return twitterHandleRegex.MatchString(path)

	case TwitterIDScheme:
		// validate path is a number and display is a handle if present
		if !allDigitsRegex.MatchString(path) {
			return false
		}
		if display != "" && !twitterHandleRegex.MatchString(display) {
			return false
		}

	case EmailScheme:
		return emailRegex.MatchString(path)

	case FacebookScheme:
		// we don't validate facebook refs since they come from the outside
		if strings.HasPrefix(path, "ref:") {
			return true
		}

		// otherwise, this should be an int
		return allDigitsRegex.MatchString(path)

	case TelegramScheme:
		return allDigitsRegex.MatchString(path)

	case ViberScheme:
		return viberRegex.MatchString(path)
	}

	return true // anything goes for external schemes
}

// Scheme returns the scheme portion for the URN
func (u URN) Scheme() string {
	scheme, _, _ := u.ToParts()
	return scheme
}

// Path returns the path portion for the URN
func (u URN) Path() string {
	_, path, _ := u.ToParts()
	return path
}

// Display returns the display portion for the URN (if any)
func (u URN) Display() string {
	_, _, display := u.ToParts()
	return display
}

// Identity returns the URN with any display attributes stripped
func (u URN) Identity() string {
	parts := strings.SplitN(string(u), "#", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	return string(u)
}

func (u URN) Resolve(key string) interface{} {
	switch key {
	case "path":
		return u.Path()
	case "scheme":
		return u.Scheme()
	case "urn":
		return string(u)
	}
	return fmt.Errorf("no field '%s' on URN", key)
}

func (u URN) Default() interface{} { return u }
func (u URN) String() string       { return string(u.Path()) }

// NilURN is our constant for nil URNs
var NilURN = URN("")

func normalizeNumber(number string, country string) string {
	number = strings.ToLower(number)

	// if the number ends with e11, then that is Excel corrupting it, remove it
	if strings.HasSuffix(number, "e+11") || strings.HasSuffix(number, "e+12") {
		number = strings.Replace(number[0:len(number)-4], ".", "", -1)
	}

	// remove other characters
	number = nonTelCharsRegex.ReplaceAllString(strings.ToLower(strings.TrimSpace(number)), "")
	parseNumber := number

	// add on a plus if it looks like it could be a fully qualified number
	if len(number) >= 11 && !(strings.HasPrefix(number, "+") || strings.HasPrefix(number, "0")) {
		parseNumber = fmt.Sprintf("+%s", number)
	}

	normalized, err := phonenumbers.Parse(parseNumber, country)

	// couldn't parse it, use the original number
	if err != nil {
		return number
	}

	// if it looks valid, return it
	if phonenumbers.IsValidNumber(normalized) {
		return phonenumbers.Format(normalized, phonenumbers.E164)
	}

	// this doesn't look like anything we recognize, use the original number
	return number
}