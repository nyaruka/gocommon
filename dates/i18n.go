package dates

import (
	_ "embed"
	"encoding/json"
	"sort"
)

// file containing day and month translations, generated using https://github.com/nyaruka/go-locales
//
// ./localesdump --bcp47 --merge days=LC_TIME.day short_days=LC_TIME.abday months=LC_TIME.mon short_months=LC_TIME.abmon am_pm=LC_TIME.am_pm > dates.json
//
//go:embed i18n/i18n.json
var i18nJSON []byte

type Translation struct {
	Days        []string `json:"days"`
	ShortDays   []string `json:"short_days"`
	Months      []string `json:"months"`
	ShortMonths []string `json:"short_months"`
	AmPm        []string `json:"am_pm"`
}

var translations map[string]*Translation
var backdowns = map[string]*Translation{} // language only backdowns for locales that have countries
var defaultLocale = "en-US"

func init() {
	err := json.Unmarshal(i18nJSON, &translations)
	if err != nil {
		panic(err)
	}

	// not all locales have AM/PM values.. but it's simpler if we just given them a default
	for _, trans := range translations {
		if trans.AmPm[0] == "" {
			trans.AmPm = []string{"AM", "PM"}
		}
	}

	// so that we can iterate translations deterministically (code a-z)
	codes := make([]string, len(translations))
	for c := range translations {
		codes = append(codes, c)
	}
	sort.Strings(codes)

	for _, code := range codes {
		if len(code) == 5 {
			lang := code[:2]
			if backdowns[lang] == nil {
				backdowns[lang] = translations[code] // using first is arbitary but best we can do
			}
		}
	}
}

// GetTranslation gets the best match translation for the given locale
func GetTranslation(locale string) *Translation {
	if locale == "" {
		return translations[defaultLocale]
	}

	// try extract xx_YY match
	t := translations[locale]
	if t != nil {
		return t
	}

	// try match by language xx only
	lang := locale[:2]
	t = translations[lang]
	if t != nil {
		return t
	}

	// use backdown for this language
	t = backdowns[lang]
	if t != nil {
		return t
	}

	// use default
	return translations[defaultLocale]
}
