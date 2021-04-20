package dates_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/dates"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDateTime(t *testing.T) {
	kigali, err := time.LoadLocation("Africa/Kigali")
	require.NoError(t, err)

	tests := []struct {
		layout   string
		value    string
		tz       *time.Location
		expected time.Time
		err      string
	}{
		{"YYYY-MM-DD t:mm", "2018-12-30 14:45", time.UTC, time.Date(2018, 12, 30, 14, 45, 0, 0, time.UTC), ""},
		{"YYYY-MM-DD t:mm", "2018-12-30 14:45", kigali, time.Date(2018, 12, 30, 14, 45, 0, 0, kigali), ""},
		{"YY/M/D t:mm:ss.fff", "21/1/2 9:15:30.123", kigali, time.Date(2021, 1, 2, 9, 15, 30, 123000000, kigali), ""},

		{"xxx", "Mon", time.UTC, dates.ZeroDateTime, "'xxx' is not valid in a datetime parsing layout"},
	}

	for _, tc := range tests {
		desc := fmt.Sprintf("'%s' as '%s'", tc.value, tc.layout)

		actual, err := dates.ParseDateTime(tc.layout, tc.value, tc.tz)
		if tc.err == "" {
			assert.NoError(t, err, "unexpected error for %s", desc)
			assert.Equal(t, tc.expected, actual, "parse mismatch for %s", desc)
		} else {
			assert.EqualError(t, err, tc.err, "error mismatch for %s", desc)
			assert.Equal(t, dates.ZeroDateTime, actual)
		}
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		layout   string
		value    string
		expected dates.Date
		err      string
	}{
		{"YYYY-MM-DD", "2018-12-30", dates.NewDate(2018, 12, 30), ""},
		{"DD.MM.YYYY", "20.04.2021", dates.NewDate(2021, 4, 20), ""},

		{"D/MM/YY", "1/3/21", dates.ZeroDate, "cannot parse '3/21' as 'MM'"}, // error because DD/02 requires zero-padding
		{"tt", "11", dates.ZeroDate, "'tt' is not valid in a date parsing layout"},
		{"EEE", "Mon", dates.ZeroDate, "'EEE' is not valid in a date parsing layout"},
	}

	for _, tc := range tests {
		desc := fmt.Sprintf("'%s' as '%s'", tc.value, tc.layout)

		actual, err := dates.ParseDate(tc.layout, tc.value)
		if tc.err == "" {
			assert.NoError(t, err, "unexpected error for %s", desc)
			assert.Equal(t, tc.expected, actual, "parse mismatch for %s", desc)
		} else {
			assert.EqualError(t, err, tc.err, "error mismatch for %s", desc)
			assert.Equal(t, dates.ZeroDate, actual)
		}
	}
}

func TestParseTimeOfDay(t *testing.T) {
	tests := []struct {
		layout   string
		value    string
		expected dates.TimeOfDay
		err      string
	}{
		{"tt:mm:ss.ffffff", "11:02:30.123456", dates.NewTimeOfDay(11, 2, 30, 123456000), ""},
		{"h:mm aa", "3:45 pm", dates.NewTimeOfDay(15, 45, 0, 0), ""},

		{"hh:mm aa", "3:45 pm", dates.ZeroTimeOfDay, "cannot parse '3:45 pm' as 'hh'"}, // error because hh/03 requires zero-padding
		{"MM:YY", "11:02:30.123456", dates.ZeroTimeOfDay, "'MM' is not valid in a time parsing layout"},
	}

	for _, tc := range tests {
		desc := fmt.Sprintf("'%s' as '%s'", tc.value, tc.layout)

		actual, err := dates.ParseTimeOfDay(tc.layout, tc.value)
		if tc.err == "" {
			assert.NoError(t, err, "unexpected error for %s", desc)
			assert.Equal(t, tc.expected, actual, "parse mismatch for %s", desc)
		} else {
			assert.EqualError(t, err, tc.err, "error mismatch for %s", desc)
			assert.Equal(t, dates.ZeroTimeOfDay, actual)
		}
	}
}
