package urns

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/nyaruka/phonenumbers"
)

const (
	// EmailScheme is the scheme used for email addresses
	EmailScheme string = "mailto"

	// ExternalScheme is the scheme used for externally defined identifiers
	ExternalScheme string = "ext"

	// FacebookScheme is the scheme used for Facebook identifiers
	FacebookScheme string = "facebook"

	// FCMScheme is the scheme used for Firebase Cloud Messaging identifiers
	FCMScheme string = "fcm"

	// JiochatScheme is the scheme used for Jiochat identifiers
	JiochatScheme string = "jiochat"

	// LineScheme is the scheme used for LINE identifiers
	LineScheme string = "line"

	// RbmScheme is the scheme used for RBM messaging
	RbmScheme string = "rbm"

	// TelegramScheme is the scheme used for Telegram identifiers
	TelegramScheme string = "telegram"

	// TelScheme is the scheme used for telephone numbers
	TelScheme string = "tel"

	// TwitterIDScheme is the scheme used for Twitter user ids
	TwitterIDScheme string = "twitterid"

	// TwitterScheme is the scheme used for Twitter handles
	TwitterScheme string = "twitter"

	// ViberScheme is the scheme used for Viber identifiers
	ViberScheme string = "viber"

	// WhatsAppScheme is the scheme used for WhatsApp identifiers
	WhatsAppScheme string = "whatsapp"

	// WeChatScheme is the scheme used for WeChat identifiers
	WeChatScheme string = "wechat"

	// FacebookRefPrefix is the path prefix used for facebook referral URNs
	FacebookRefPrefix string = "ref:"
)

// ValidSchemes is the set of URN schemes understood by this library
var ValidSchemes = map[string]bool{
	EmailScheme:     true,
	ExternalScheme:  true,
	FacebookScheme:  true,
	FCMScheme:       true,
	JiochatScheme:   true,
	LineScheme:      true,
	RbmScheme:       true,
	TelegramScheme:  true,
	TelScheme:       true,
	TwitterIDScheme: true,
	TwitterScheme:   true,
	ViberScheme:     true,
	WhatsAppScheme:  true,
	WeChatScheme:    true,
}

// IsValidScheme checks whether the provided scheme is valid
func IsValidScheme(scheme string) bool {
	_, valid := ValidSchemes[scheme]
	return valid
}

var nonTelCharsRegex = regexp.MustCompile(`[^0-9a-z]`)
var telRegex = regexp.MustCompile(`^\+?[a-zA-Z0-9]{3,16}$`)
var twitterHandleRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,15}$`)
var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+$`)
var viberRegex = regexp.MustCompile(`^[a-zA-Z0-9_=/+]{1,24}$`)
var lineRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,36}$`)
var allDigitsRegex = regexp.MustCompile(`^[0-9]+$`)

// URN represents a Universal Resource Name, we use this for contact identifiers like phone numbers etc..
type URN string

// NewTelURNForCountry returns a URN for the passed in telephone number and country code ("US")
func NewTelURNForCountry(number string, country string) (URN, error) {
	return NewURNFromParts(TelScheme, normalizeNumber(number, country), "", "")
}

// NewTelegramURN returns a URN for the passed in telegram identifier
func NewTelegramURN(identifier int64, display string) (URN, error) {
	return NewURNFromParts(TelegramScheme, strconv.FormatInt(identifier, 10), "", display)
}

// NewWhatsAppURN returns a URN for the passed in whatsapp identifier
func NewWhatsAppURN(identifier string) (URN, error) {
	return NewURNFromParts(WhatsAppScheme, identifier, "", "")
}

// NewRbmURN returns a URN for the passed in whatsapp identifier
func NewRbmURN(identifier string) (URN, error) {
	return NewURNFromParts(RbmScheme, identifier, "", "")
}

// NewFirebaseURN returns a URN for the passed in firebase identifier
func NewFirebaseURN(identifier string) (URN, error) {
	return NewURNFromParts(FCMScheme, identifier, "", "")
}

// NewFacebookURN returns a URN for the passed in facebook identifier
func NewFacebookURN(identifier string) (URN, error) {
	return NewURNFromParts(FacebookScheme, identifier, "", "")
}

// returns a new URN for the given scheme, path, query and display
func newURNFromParts(scheme string, path string, query string, display string) URN {
	u := &parsedURN{
		scheme:   scheme,
		path:     path,
		query:    query,
		fragment: display,
	}
	return URN(u.String())
}

// NewURNFromParts returns a validated URN for the given scheme, path, query and display
func NewURNFromParts(scheme string, path string, query string, display string) (URN, error) {
	urn := newURNFromParts(scheme, path, query, display)
	err := urn.Validate()
	if err != nil {
		return NilURN, err
	}
	return urn, nil
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
func (u URN) Normalize(country string) URN {
	scheme, path, query, display := u.ToParts()
	normPath := strings.TrimSpace(path)

	switch scheme {
	case TelScheme:
		normPath = normalizeNumber(normPath, country)

	case TwitterScheme:
		// Twitter handles are case-insensitive, so we always store as lowercase
		normPath = strings.ToLower(normPath)

		// strip @ prefix if provided
		if strings.HasPrefix(normPath, "@") {
			normPath = normPath[1:]
		}

	case TwitterIDScheme:
		if display != "" {
			display = strings.ToLower(strings.TrimSpace(display))
			if display != "" && strings.HasPrefix(display, "@") {
				display = display[1:]
			}
		}

	case EmailScheme:
		normPath = strings.ToLower(normPath)
	}

	return newURNFromParts(scheme, normPath, query, display)
}

// Validate returns whether this URN is considered valid
func (u URN) Validate() error {
	scheme, path, _, display := u.ToParts()

	if scheme == "" || path == "" {
		return fmt.Errorf("scheme or path cannot be empty")
	}
	if !IsValidScheme(scheme) {
		return fmt.Errorf("invalid scheme: '%s'", scheme)
	}

	switch scheme {
	case TelScheme:
		// validate is possible phone number
		if !telRegex.MatchString(path) {
			return fmt.Errorf("invalid tel number: %s", path)
		}

	case TwitterScheme:
		// validate twitter URNs look like handles
		if !twitterHandleRegex.MatchString(path) {
			return fmt.Errorf("invalid twitter handle: %s", path)
		}

	case TwitterIDScheme:
		// validate path is a number and display is a handle if present
		if !allDigitsRegex.MatchString(path) {
			return fmt.Errorf("invalid twitter id: %s", path)
		}
		if display != "" && !twitterHandleRegex.MatchString(display) {
			return fmt.Errorf("invalid twitter handle: %s", display)
		}

	case EmailScheme:
		if !emailRegex.MatchString(path) {
			return fmt.Errorf("invalid email: %s", path)
		}

	case FacebookScheme:
		// we don't validate facebook refs since they come from the outside
		if u.IsFacebookRef() {
			return nil
		}

		// otherwise, this should be an int
		if !allDigitsRegex.MatchString(path) {
			return fmt.Errorf("invalid facebook id: %s", path)
		}
	case JiochatScheme:
		if !allDigitsRegex.MatchString(path) {
			return fmt.Errorf("invalid jiochat id: %s", path)
		}

	case LineScheme:
		if !lineRegex.MatchString(path) {
			return fmt.Errorf("invalid line id: %s", path)
		}
	case RbmScheme:
		if !telRegex.MatchString(path) {
			return fmt.Errorf("invalid rbm number: %s", path)
		}
	case TelegramScheme:
		if !allDigitsRegex.MatchString(path) {
			return fmt.Errorf("invalid telegram id: %s", path)
		}

	case ViberScheme:
		if !viberRegex.MatchString(path) {
			return fmt.Errorf("invalid viber id: %s", path)
		}

	case WhatsAppScheme:
		if !allDigitsRegex.MatchString(path) {
			return fmt.Errorf("invalid whatsapp id: %s", path)
		}
	}

	return nil // anything goes for external schemes
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
	return newURNFromParts(scheme, path, "", "")
}

// Localize returns a new URN which is local to the given country
func (u URN) Localize(country string) URN {
	scheme, path, query, display := u.ToParts()

	if scheme == TelScheme {
		parsed, err := phonenumbers.Parse(path, country)
		if err == nil {
			path = strconv.FormatUint(parsed.GetNationalNumber(), 10)
		}
	}
	return newURNFromParts(scheme, path, query, display)
}

// IsFacebookRef returns whether this URN is a facebook referral
func (u URN) IsFacebookRef() bool {
	return u.Scheme() == FacebookScheme && strings.HasPrefix(u.Path(), FacebookRefPrefix)
}

// FacebookRef returns the facebook referral portion of our path, this return empty string in the case where we aren't a Facebook scheme
func (u URN) FacebookRef() string {
	if u.IsFacebookRef() {
		return strings.TrimPrefix(u.Path(), FacebookRefPrefix)
	}
	return ""
}

// String returns the string representation of this URN
func (u URN) String() string { return string(u) }

// Format formats this URN as a human friendly string
func (u URN) Format() string {
	scheme, path, _, display := u.ToParts()

	if scheme == TelScheme {
		parsed, err := phonenumbers.Parse(path, "")
		if err != nil {
			return path
		}
		return phonenumbers.Format(parsed, phonenumbers.NATIONAL)
	}

	if display != "" {
		return display
	}
	return path
}

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
