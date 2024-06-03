package dates_test

import (
	"testing"

	"github.com/nyaruka/gocommon/dates"
	"github.com/nyaruka/gocommon/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTranslation(t *testing.T) {
	tests := []struct {
		locale  i18n.Locale
		sun     string
		sunday  string
		jan     string
		january string
		am      string
	}{
		{"", "Sun", "Sunday", "Jan", "January", "AM"},
		{"eng-US", "Sun", "Sunday", "Jan", "January", "AM"},
		{"eng-GB", "Sun", "Sunday", "Jan", "January", "am"},
		{"eng", "Sun", "Sunday", "Jan", "January", "AM"},
		{"spa-EC", "dom", "domingo", "ene", "enero", "AM"},
		{"spa", "dom", "domingo", `ene`, "enero", `a. m.`},
		{"por-BR", "dom", "domingo", "jan", "janeiro", "AM"},
		{"por-PT", "dom", "domingo", "jan", "janeiro", "AM"},
		{"por", "dom", "domingo", "jan", "janeiro", "AM"},
		{"kin-RW", "Mwe", "Ku cyumweru", "Mut", "Mutarama", "AM"},
		{"kin", "Mwe", "Ku cyumweru", "Mut", "Mutarama", "AM"},
		{"zho-CN", "日", "星期日", "1月", "一月", "上午"},
		{"zho-HK", "日", "星期日", "1月", "一月", "上午"},
		{"zho-SG", "日", "星期日", "一月", "一月", "上午"},
		{"zho-TW", "日", "週日", " 1月", "一月", "上午"},
		{"zho", "日", "星期日", "1月", "一月", "上午"}, // backs down to first zh translation
	}

	for _, tc := range tests {
		trans := dates.GetTranslation(tc.locale)
		require.NotNil(t, trans, "trans unexpectedly nil for local '%s'", tc.locale)
		assert.Equal(t, tc.sun, trans.ShortDays[0], "short day mismatch for locale %s", tc.locale)
		assert.Equal(t, tc.sunday, trans.Days[0], "full day mismatch for locale %s", tc.locale)
		assert.Equal(t, tc.jan, trans.ShortMonths[0], "short month mismatch for locale %s", tc.locale)
		assert.Equal(t, tc.january, trans.Months[0], "full month mismatch for locale %s", tc.locale)
		assert.Equal(t, tc.am, trans.AmPm[0], "AM mismatch for locale %s", tc.locale)
	}
}
