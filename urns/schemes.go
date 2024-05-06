package urns

import (
	"regexp"
	"strings"

	"github.com/nyaruka/phonenumbers"
)

var allDigitsRegex = regexp.MustCompile(`^[0-9]+$`)
var nonTelCharsRegex = regexp.MustCompile(`[^0-9A-Z]`)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+$`)
var freshchatRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}/[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}$`)
var viberRegex = regexp.MustCompile(`^[a-zA-Z0-9_=/+]{1,24}$`)
var lineRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,36}$`)
var telRegex = regexp.MustCompile(`^\+?[a-zA-Z0-9]{1,64}$`)
var twitterHandleRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,15}$`)
var webchatRegex = regexp.MustCompile(`^[a-zA-Z0-9]{24}(:[^\s@]+@[^\s@]+)?$`)

const (
	// FacebookRefPrefix is prefix used for facebook referral URNs
	FacebookRefPrefix string = "ref:"
)

func init() {
	register(Discord)
	register(Email)
	register(External)
	register(Facebook)
	register(Firebase)
	register(FreshChat)
	register(Instagram)
	register(JioChat)
	register(Line)
	register(Phone)
	register(RocketChat)
	register(Slack)
	register(Telegram)
	register(Twitter)
	register(TwitterID)
	register(Viber)
	register(VK)
	register(WebChat)
	register(WeChat)
	register(WhatsApp)
}

var schemes = map[string]*Scheme{}
var schemePrefixes = []string{}

func register(s *Scheme) {
	schemes[s.Prefix] = s
	schemePrefixes = append(schemePrefixes, s.Prefix)
}

type Scheme struct {
	Prefix    string
	Normalize func(string) string
	Validate  func(string) bool
	Format    func(string) string
}

var Discord = &Scheme{
	Prefix:   "discord",
	Validate: func(path string) bool { return allDigitsRegex.MatchString(path) },
}

var Email = &Scheme{
	Prefix:    "mailto",
	Normalize: func(path string) string { return strings.ToLower(path) },
	Validate:  func(path string) bool { return emailRegex.MatchString(path) },
}

var External = &Scheme{
	Prefix: "ext",
}

var Facebook = &Scheme{
	Prefix: "facebook",
	Validate: func(path string) bool {
		// we don't validate facebook refs since they come from the outside
		if strings.HasPrefix(path, FacebookRefPrefix) {
			return true
		}
		// otherwise, this should be an int
		return allDigitsRegex.MatchString(path)
	},
}

var Firebase = &Scheme{
	Prefix: "fcm",
}

var FreshChat = &Scheme{
	Prefix:   "freshchat",
	Validate: func(path string) bool { return freshchatRegex.MatchString(path) },
}

var Instagram = &Scheme{
	Prefix:   "instagram",
	Validate: func(path string) bool { return allDigitsRegex.MatchString(path) },
}

var JioChat = &Scheme{
	Prefix:   "jiochat",
	Validate: func(path string) bool { return allDigitsRegex.MatchString(path) },
}

var Line = &Scheme{
	Prefix:   "line",
	Validate: func(path string) bool { return lineRegex.MatchString(path) },
}

var Phone = &Scheme{
	Prefix: "tel",
	Normalize: func(path string) string {
		e164, err := ParsePhone(path, "")
		if err != nil {
			// could be a short code so uppercase and remove non alphanumeric characters
			return nonTelCharsRegex.ReplaceAllString(strings.ToUpper(path), "")
		}

		return e164
	},
	Validate: func(path string) bool { return telRegex.MatchString(path) },
	Format: func(path string) string {
		parsed, err := phonenumbers.Parse(path, "")
		if err != nil {
			return path
		}
		return phonenumbers.Format(parsed, phonenumbers.NATIONAL)
	},
}

var RocketChat = &Scheme{
	Prefix: "rocketchat",
}

var Slack = &Scheme{
	Prefix: "slack",
}

var Telegram = &Scheme{
	Prefix:   "telegram",
	Validate: func(path string) bool { return allDigitsRegex.MatchString(path) },
}

var Twitter = &Scheme{
	Prefix: "twitter",
	Normalize: func(path string) string {
		// handles are case-insensitive, so we always store as lowercase
		path = strings.ToLower(path)

		// strip @ prefix if provided
		return strings.TrimPrefix(path, "@")
	},
	Validate: func(path string) bool { return twitterHandleRegex.MatchString(path) },
}

var TwitterID = &Scheme{
	Prefix:   "twitterid",
	Validate: func(path string) bool { return allDigitsRegex.MatchString(path) },
}

var Viber = &Scheme{
	Prefix:   "viber",
	Validate: func(path string) bool { return viberRegex.MatchString(path) },
}

var VK = &Scheme{
	Prefix: "vk",
}

var WebChat = &Scheme{
	Prefix:   "webchat",
	Validate: func(path string) bool { return webchatRegex.MatchString(path) },
}

var WeChat = &Scheme{
	Prefix: "wechat",
}

var WhatsApp = &Scheme{
	Prefix:   "whatsapp",
	Validate: func(path string) bool { return allDigitsRegex.MatchString(path) },
}
