package dates

import (
	_ "embed"
	"maps"
	"slices"

	"github.com/nyaruka/gocommon/i18n"
	"github.com/nyaruka/gocommon/jsonx"
)

// File containing day and month translations, generated using https://github.com/nyaruka/go-locales
//
// ./localesdump --merge days=LC_TIME.day short_days=LC_TIME.abday months=LC_TIME.mon short_months=LC_TIME.abmon am_pm=LC_TIME.am_pm > dates.json
//
//go:embed i18n/dates.json
var i18nJSON []byte

type Translation struct {
	Days        []string `json:"days"`
	ShortDays   []string `json:"short_days"`
	Months      []string `json:"months"`
	ShortMonths []string `json:"short_months"`
	AmPm        []string `json:"am_pm"`
}

var bcp47Matcher *i18n.BCP47Matcher
var translations map[string]*Translation
var defaultLocale = "en_US"

func init() {
	jsonx.MustUnmarshal(i18nJSON, &translations)

	bcp47Matcher = i18n.NewBCP47Matcher(slices.Collect(maps.Keys(translations))...)

	// not all locales have AM/PM values.. but it's simpler if we just given them a default
	for _, trans := range translations {
		if trans.AmPm[0] == "" {
			trans.AmPm = []string{"AM", "PM"}
		}
	}
}

// GetTranslation gets the best match translation for the given locale
func GetTranslation(loc i18n.Locale) *Translation {
	if loc == "" {
		return translations[defaultLocale]
	}

	code := bcp47Matcher.ForLocales(loc)

	return translations[code]
}
