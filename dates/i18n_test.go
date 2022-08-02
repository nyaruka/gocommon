package dates_test

import (
	"testing"

	"github.com/nyaruka/gocommon/dates"

	"github.com/stretchr/testify/assert"
)

func TestGetTranslation(t *testing.T) {
	tests := []struct {
		locale  string
		sun     string
		sunday  string
		jan     string
		january string
		am      string
	}{
		{"", "Sun", "Sunday", "Jan", "January", "AM"},
		{"en-US", "Sun", "Sunday", "Jan", "January", "AM"},
		{"en-GB", "Sun", "Sunday", "Jan", "January", "am"},
		{"en", "Sun", "Sunday", "Jan", "January", "am"},
		{"es-EC", "dom", "domingo", "ene", "enero", "AM"},
		{"es", "dom", "domingo", "ene", "enero", "AM"},
		{"pt-BR", "dom", "domingo", "jan", "janeiro", "AM"},
		{"pt-PT", "dom", "domingo", "jan", "janeiro", "AM"},
		{"pt", "dom", "domingo", "jan", "janeiro", "AM"},
		{"rw-RW", "Mwe", "Ku cyumweru", "Mut", "Mutarama", "AM"},
		{"rw", "Mwe", "Ku cyumweru", "Mut", "Mutarama", "AM"},
		{"zh-CN", "日", "星期日", "1月", "一月", "上午"},
		{"zh-HK", "日", "星期日", "1月", "一月", "上午"},
		{"zh-SG", "日", "星期日", "一月", "一月", "上午"},
		{"zh-TW", "日", "週日", " 1月", "一月", "上午"},
		{"zh", "日", "星期日", "1月", "一月", "上午"}, // backs down to first zh translation
	}

	for _, tc := range tests {
		trans := dates.GetTranslation(tc.locale)
		assert.Equal(t, tc.sun, trans.ShortDays[0], "short day mismatch for locale %s", tc.locale)
		assert.Equal(t, tc.sunday, trans.Days[0], "full day mismatch for locale %s", tc.locale)
		assert.Equal(t, tc.jan, trans.ShortMonths[0], "short month mismatch for locale %s", tc.locale)
		assert.Equal(t, tc.january, trans.Months[0], "full month mismatch for locale %s", tc.locale)
		assert.Equal(t, tc.am, trans.AmPm[0], "AM mismatch for locale %s", tc.locale)
	}
}
