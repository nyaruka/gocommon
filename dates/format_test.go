package dates_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/dates"
	"github.com/nyaruka/gocommon/i18n"
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
		locale   i18n.Locale
		expected string
		err      string
	}{
		{d1, "MM-DD-YYYY", "en-US", "01-02-2006", ""},
		{d1, "M-D-YY", "eng-US", "1-2-06", ""},
		{d1, "h:m", "eng-US", "3:4", ""},
		{d1, "h:m:s aa", "eng-US", "3:4:5 pm", ""},
		{d1, "h:m:s AA", "eng-US", "3:4:5 PM", ""},
		{d1, "tt:mm:ss", "eng-US", "15:04:05", ""},
		{d2, "tt:mm:ss", "eng-US", "09:45:30", ""},
		{d2, "t:mm:ss", "eng-US", "9:45:30", ""},
		{d1, "YYYY-MM-DDTtt:mm:ssZZZ", "eng-US", "2006-01-02T15:04:05-07:00", ""},
		{d1, "YYYY-MM-DDTtt:mm:ssZZZ", "eng-US", "2006-01-02T15:04:05-07:00", ""},
		{d1, "YYYY-MM-DDThh:mm:ss.fffZZZ", "eng-US", "2006-01-02T03:04:05.123-07:00", ""},
		{d1, "YYYY-MM-DDThh:mm:ss.fffZ", "eng-US", "2006-01-02T03:04:05.123-07:00", ""},
		{d1, "YY-M-D", "eng-US", "06-1-2", ""},
		{d1, "YYYY-MM-DD", "eng-US", "2006-01-02", ""},
		{d1, "YYYY-MMM-DD", "eng-US", "2006-Jan-02", ""},
		{d1, "YYYY-MMMM-DD", "eng-US", "2006-January-02", ""},
		{d1, "//YY--MM::DD..", "eng-US", "//06--01::02..", ""},

		// localization
		{d1, "EEE EEEE MMM MMMM AA aa", "eng-US", `Mon Monday Jan January PM pm`, ""},
		{d1, "EEE EEEE MMM MMMM AA aa", "spa-EC", `lun lunes ene enero PM pm`, ""},
		{d1, "EEE EEEE MMM MMMM AA aa", "spa", `lun lunes ene enero P. M. p. m.`, ""},
		{d1, "EEE EEEE MMM MMMM AA aa", "ara-QA", `ن الاثنين ينا يناير م م`, ""},
		{d1, "EEE EEEE MMM MMMM AA aa", "rus", `Пн Понедельник янв января PM pm`, ""},
		{d1, "EEE EEEE MMM MMMM AA aa", "tir", `ሰኑይ ሰኑይ ጃንዩ ጃንዩወሪ ድሕር ሰዓት ድሕር ሰዓት`, ""},
		{d2, "EEE EEEE MMM MMMM AA aa", "eng-US", `Sat Saturday Apr April AM am`, ""},
		{d2, "EEE EEEE MMM MMMM AA aa", "spa-EC", `sáb sábado abr abril AM am`, ""},
		{d2, "EEE EEEE MMM MMMM AA aa", "ara-QA", `س السبت أبر أبريل ص ص`, ""},
		{d2, "EEE EEEE MMM MMMM AA aa", "rus", `Сб Суббота апр апреля AM am`, ""},
		{d2, "EEE EEEE MMM MMMM AA aa", "tir", `ቀዳም ቀዳም ኤፕረ ኤፕረል ንጉሆ ሰዓተ ንጉሆ ሰዓተ`, ""},
		{d3, "EEE EEEE MMM MMMM AA aa", "eng-US", `Tue Tuesday Dec December PM pm`, ""},
		{d3, "EEE EEEE MMM MMMM AA aa", "spa-EC", `mar martes dic diciembre PM pm`, ""},
		{d3, "EEE EEEE MMM MMMM AA aa", "ara-QA", `ث الثلاثاء ديس ديسمبر م م`, ""},
		{d3, "EEE EEEE MMM MMMM AA aa", "rus", `Вт Вторник дек декабря PM pm`, ""},
		{d3, "EEE EEEE MMM MMMM AA aa", "tir", `ሰሉስ ሰሉስ ዲሴም ዲሴምበር ድሕር ሰዓት ድሕር ሰዓት`, ""},

		// fractional seconds
		{d1, "tt:mm:ss.fff", "eng-US", "15:04:05.123", ""},
		{d1, "tt:mm:ss.ffffff", "eng-US", "15:04:05.123456", ""},
		{d1, "tt:mm:ss.fffffffff", "eng-US", "15:04:05.123456789", ""},

		// errors
		{d1, "YYY-MM-DD", "eng-US", "", "'YYY' is not valid in a datetime formatting layout"},
		{d1, "YYYY-MMMMM-DD", "eng-US", "", "'MMMMM' is not valid in a datetime formatting layout"},
		{d1, "EE", "eng-US", "", "'EE' is not valid in a datetime formatting layout"},
		{d1, "tt:mm:ss.ffff", "eng-US", "", "'ffff' is not valid in a datetime formatting layout"},
		{d1, "tt:mmm:ss.ffff", "eng-US", "", "'mmm' is not valid in a datetime formatting layout"},
		{d1, "tt:mm:sss", "eng-US", "", "'sss' is not valid in a datetime formatting layout"},
		{d1, "tt:mm:ss a", "eng-US", "", "'a' is not valid in a datetime formatting layout"},
		{d1, "tt:mm:ss A", "eng-US", "", "'A' is not valid in a datetime formatting layout"},
		{d1, "tt:mm:ssZZZZ", "eng-US", "", "'ZZZZ' is not valid in a datetime formatting layout"},
		{d1, "2006-01-02", "eng-US", "", "'2' is not valid in a datetime formatting layout"},
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
