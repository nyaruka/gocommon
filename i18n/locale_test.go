package i18n_test

import (
	"testing"

	"github.com/nyaruka/gocommon/i18n"
	"github.com/stretchr/testify/assert"
)

func TestLocale(t *testing.T) {
	assert.Equal(t, i18n.Locale(""), i18n.NewLocale("", ""))
	assert.Equal(t, i18n.Locale(""), i18n.NewLocale("", "US"))     // invalid without language
	assert.Equal(t, i18n.Locale("eng"), i18n.NewLocale("eng", "")) // valid without country
	assert.Equal(t, i18n.Locale("eng-US"), i18n.NewLocale("eng", "US"))

	l, c := i18n.Locale("eng-US").Split()
	assert.Equal(t, i18n.Language("eng"), l)
	assert.Equal(t, i18n.Country("US"), c)

	l, c = i18n.NilLocale.Split()
	assert.Equal(t, i18n.NilLanguage, l)
	assert.Equal(t, i18n.NilCountry, c)

	v, err := i18n.NewLocale("eng", "US").Value()
	assert.NoError(t, err)
	assert.Equal(t, "eng-US", v)

	v, err = i18n.NilLanguage.Value()
	assert.NoError(t, err)
	assert.Nil(t, v)

	var lc i18n.Locale
	assert.NoError(t, lc.Scan("eng-US"))
	assert.Equal(t, i18n.Locale("eng-US"), lc)

	assert.NoError(t, lc.Scan(nil))
	assert.Equal(t, i18n.NilLocale, lc)
}

func TesBCP47Matcher(t *testing.T) {
	tests := []struct {
		preferred []i18n.Locale
		available []string
		best      string
	}{
		{preferred: []i18n.Locale{"eng-US"}, available: []string{"es_EC", "en-US"}, best: "en-US"},
		{preferred: []i18n.Locale{"eng-US"}, available: []string{"es", "en"}, best: "en"},
		{preferred: []i18n.Locale{"eng"}, available: []string{"es-US", "en-UK"}, best: "en-UK"},
		{preferred: []i18n.Locale{"eng", "fra"}, available: []string{"fr-CA", "en-RW"}, best: "en-RW"},
		{preferred: []i18n.Locale{"eng", "fra"}, available: []string{"fra-CA", "eng-RW"}, best: "eng-RW"},
		{preferred: []i18n.Locale{"fra", "eng"}, available: []string{"fra-CA", "eng-RW"}, best: "fra-CA"},
		{preferred: []i18n.Locale{"spa"}, available: []string{"es-EC", "es-MX", "es-ES"}, best: "es-ES"},
		{preferred: []i18n.Locale{}, available: []string{"es_EC", "en-US"}, best: "es_EC"},
	}

	for _, tc := range tests {
		m := i18n.NewBCP47Matcher(tc.available...)
		best := m.ForLocales(tc.preferred...)

		assert.Equal(t, tc.best, best, "locale mismatch for preferred=%v available=%s", tc.preferred, tc.available)
	}
}
