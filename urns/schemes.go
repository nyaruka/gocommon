package urns

import (
	"regexp"
	"strings"

	"github.com/nyaruka/phonenumbers"
)

var allDigitsRegex = regexp.MustCompile(`^[0-9]+$`)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+$`)
var freshchatRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}/[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}$`)
var viberRegex = regexp.MustCompile(`^[a-zA-Z0-9_=/+]{1,24}$`)
var lineRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,36}$`)
var phoneRegex = regexp.MustCompile(`^\+?\d{1,64}$`)
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

var schemeByPrefix = map[string]*Scheme{}
var Schemes = []*Scheme{}

func register(s *Scheme) {
	schemeByPrefix[s.Prefix] = s
	Schemes = append(Schemes, s)
}

// Scheme represents a URN scheme, e.g. tel, email, etc.
type Scheme struct {
	Prefix    string
	Name      string
	Normalize func(string) string
	Validate  func(string) bool
	Format    func(string) string
}

var Discord = &Scheme{
	Prefix:   "discord",
	Name:     "Discord",
	Validate: func(path string) bool { return allDigitsRegex.MatchString(path) },
}

var Email = &Scheme{
	Prefix:    "mailto",
	Name:      "Email",
	Normalize: func(path string) string { return strings.ToLower(path) },
	Validate:  func(path string) bool { return emailRegex.MatchString(path) },
}

var External = &Scheme{
	Prefix: "ext",
	Name:   "External",
}

var Facebook = &Scheme{
	Prefix: "facebook",
	Name:   "Facebook",
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
	Name:   "Firebase",
}

var FreshChat = &Scheme{
	Prefix:   "freshchat",
	Name:     "FreshChat",
	Validate: func(path string) bool { return freshchatRegex.MatchString(path) },
}

var Instagram = &Scheme{
	Prefix:   "instagram",
	Name:     "Instagram",
	Validate: func(path string) bool { return allDigitsRegex.MatchString(path) },
}

var JioChat = &Scheme{
	Prefix:   "jiochat",
	Name:     "JioChat",
	Validate: func(path string) bool { return allDigitsRegex.MatchString(path) },
}

var Line = &Scheme{
	Prefix:   "line",
	Name:     "LINE",
	Validate: func(path string) bool { return lineRegex.MatchString(path) },
}

var Phone = &Scheme{
	Prefix: "tel",
	Name:   "Phone",
	Normalize: func(path string) string {
		// might have alpha characters in it
		return strings.ToUpper(path)
	},
	Validate: func(path string) bool { return phoneRegex.MatchString(path) },
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
	Name:   "Rocket.Chat",
}

var Slack = &Scheme{
	Prefix: "slack",
	Name:   "Slack",
}

var Telegram = &Scheme{
	Prefix:   "telegram",
	Name:     "Telegram",
	Validate: func(path string) bool { return allDigitsRegex.MatchString(path) },
}

var Twitter = &Scheme{
	Prefix: "twitter",
	Name:   "Twitter Handle",
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
	Name:     "Twitter",
	Validate: func(path string) bool { return allDigitsRegex.MatchString(path) },
}

var Viber = &Scheme{
	Prefix:   "viber",
	Name:     "Viber",
	Validate: func(path string) bool { return viberRegex.MatchString(path) },
}

var VK = &Scheme{
	Prefix: "vk",
	Name:   "VK",
}

var WebChat = &Scheme{
	Prefix:   "webchat",
	Name:     "WebChat",
	Validate: func(path string) bool { return webchatRegex.MatchString(path) },
}

var WeChat = &Scheme{
	Prefix: "wechat",
	Name:   "WeChat",
}

var WhatsApp = &Scheme{
	Prefix:   "whatsapp",
	Name:     "WhatsApp",
	Validate: func(path string) bool { return allDigitsRegex.MatchString(path) },
}
