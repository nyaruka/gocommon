package dates_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/dates"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormat(t *testing.T) {
	mst, err := time.LoadLocation("MST")
	require.NoError(t, err)

	d1 := time.Date(2006, 1, 2, 15, 4, 5, 123456789, mst)
	d2 := time.Date(1998, 4, 18, 9, 45, 30, 123456789, time.UTC)
	d3 := time.Date(2030, 12, 31, 23, 59, 59, 987654321, time.UTC)

	tests := []struct {
		value    time.Time
		layout   string
		locale   string
		expected string
		err      string
	}{
		{d1, "MM-DD-YYYY", "en_US", "01-02-2006", ""},
		{d1, "M-D-YY", "en_US", "1-2-06", ""},
		{d1, "h:m", "en_US", "3:4", ""},
		{d1, "h:m:s aa", "en_US", "3:4:5 pm", ""},
		{d1, "h:m:s AA", "en_US", "3:4:5 PM", ""},
		{d1, "tt:mm:ss", "en_US", "15:04:05", ""},
		{d2, "tt:mm:ss", "en_US", "09:45:30", ""},
		{d2, "t:mm:ss", "en_US", "9:45:30", ""},
		{d1, "YYYY-MM-DDTtt:mm:ssZZZ", "en_US", "2006-01-02T15:04:05-07:00", ""},
		{d1, "YYYY-MM-DDTtt:mm:ssZZZ", "en_US", "2006-01-02T15:04:05-07:00", ""},
		{d1, "YYYY-MM-DDThh:mm:ss.fffZZZ", "en_US", "2006-01-02T03:04:05.123-07:00", ""},
		{d1, "YYYY-MM-DDThh:mm:ss.fffZ", "en_US", "2006-01-02T03:04:05.123-07:00", ""},
		{d1, "YY-M-D", "en_US", "06-1-2", ""},
		{d1, "YYYY-MM-DD", "en_US", "2006-01-02", ""},
		{d1, "YYYY-MMM-DD", "en_US", "2006-Jan-02", ""},
		{d1, "YYYY-MMMM-DD", "en_US", "2006-January-02", ""},
		{d1, "//YY--MM::DD..", "en_US", "//06--01::02..", ""},

		// localization
		{d1, "EEE EEEE MMM MMMM AA aa", "en_US", "Mon Monday Jan January PM pm", ""},
		{d1, "EEE EEEE MMM MMMM AA aa", "es_EC", "lun lunes ene enero PM pm", ""},
		{d1, "EEE EEEE MMM MMMM AA aa", "ar_QA", "ن الاثنين ينا يناير م م", ""},
		{d1, "EEE EEEE MMM MMMM AA aa", "ru", "Пн Понедельник янв января PM pm", ""},
		{d1, "EEE EEEE MMM MMMM AA aa", "ti", "ሰኑይ ሰኑይ ጥሪ  ጥሪ ድሕር ሰዓት ድሕር ሰዓት", ""},
		{d2, "EEE EEEE MMM MMMM AA aa", "en_US", "Sat Saturday Apr April AM am", ""},
		{d2, "EEE EEEE MMM MMMM AA aa", "es_EC", "sáb sábado abr abril AM am", ""},
		{d2, "EEE EEEE MMM MMMM AA aa", "ar_QA", "س السبت أبر أبريل ص ص", ""},
		{d2, "EEE EEEE MMM MMMM AA aa", "ru", "Сб Суббота апр апреля AM am", ""},
		{d2, "EEE EEEE MMM MMMM AA aa", "ti", "ቀዳም ቀዳም ሚያዝ ሚያዝያ ንጉሆ ሰዓተ ንጉሆ ሰዓተ", ""},
		{d3, "EEE EEEE MMM MMMM AA aa", "en_US", "Tue Tuesday Dec December PM pm", ""},
		{d3, "EEE EEEE MMM MMMM AA aa", "es_EC", "mar martes dic diciembre PM pm", ""},
		{d3, "EEE EEEE MMM MMMM AA aa", "ar_QA", "ث الثلاثاء ديس ديسمبر م م", ""},
		{d3, "EEE EEEE MMM MMMM AA aa", "ru", "Вт Вторник дек декабря PM pm", ""},
		{d3, "EEE EEEE MMM MMMM AA aa", "ti", "ሰሉስ ሰሉስ ታሕሳ ታሕሳስ ድሕር ሰዓት ድሕር ሰዓት", ""},

		// fractional seconds
		{d1, "tt:mm:ss.fff", "en_US", "15:04:05.123", ""},
		{d1, "tt:mm:ss.ffffff", "en_US", "15:04:05.123456", ""},
		{d1, "tt:mm:ss.fffffffff", "en_US", "15:04:05.123456789", ""},

		// errors
		{d1, "YYY-MM-DD", "en_US", "", "'YYY' is not valid in a datetime formatting layout"},
		{d1, "YYYY-MMMMM-DD", "en_US", "", "'MMMMM' is not valid in a datetime formatting layout"},
		{d1, "EE", "en_US", "", "'EE' is not valid in a datetime formatting layout"},
		{d1, "tt:mm:ss.ffff", "en_US", "", "'ffff' is not valid in a datetime formatting layout"},
		{d1, "tt:mmm:ss.ffff", "en_US", "", "'mmm' is not valid in a datetime formatting layout"},
		{d1, "tt:mm:sss", "en_US", "", "'sss' is not valid in a datetime formatting layout"},
		{d1, "tt:mm:ss a", "en_US", "", "'a' is not valid in a datetime formatting layout"},
		{d1, "tt:mm:ss A", "en_US", "", "'A' is not valid in a datetime formatting layout"},
		{d1, "tt:mm:ssZZZZ", "en_US", "", "'ZZZZ' is not valid in a datetime formatting layout"},
		{d1, "2006-01-02", "en_US", "", "'2' is not valid in a datetime formatting layout"},
	}

	for _, tc := range tests {
		desc := fmt.Sprintf("%s as '%s' in '%s'", tc.value.String(), tc.layout, tc.locale)

		actual, err := dates.Format(tc.value, tc.layout, tc.locale, dates.DateTimeLayouts)
		if tc.err == "" {
			assert.NoError(t, err, "unexpected error for %s", desc)
			assert.Equal(t, tc.expected, actual, "format mismatch for %s", desc)
		} else {
			assert.EqualError(t, err, tc.err, "error mismatch for %s", desc)
			assert.Equal(t, "", actual)
		}
	}
}
