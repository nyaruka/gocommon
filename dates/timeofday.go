package dates

import (
	"time"

	"github.com/nyaruka/gocommon/i18n"
)

const (
	iso8601Time = "tt:mm:ss.ffffff"
)

// TimeOfDay represents a local time of day value
type TimeOfDay struct {
	Hour   int
	Minute int
	Second int
	Nanos  int
}

// NewTimeOfDay creates a new time of day
func NewTimeOfDay(hour, minute, second, nanos int) TimeOfDay {
	return TimeOfDay{Hour: hour, Minute: minute, Second: second, Nanos: nanos}
}

// ExtractTimeOfDay extracts the time of day from the give datetime
func ExtractTimeOfDay(dt time.Time) TimeOfDay {
	return NewTimeOfDay(dt.Hour(), dt.Minute(), dt.Second(), dt.Nanosecond())
}

// Equal determines equality for this type
func (t TimeOfDay) Equal(other TimeOfDay) bool {
	return t.Hour == other.Hour && t.Minute == other.Minute && t.Second == other.Second && t.Nanos == other.Nanos
}

// Compare compares this time of day to another
func (t TimeOfDay) Compare(other TimeOfDay) int {
	if t.Hour != other.Hour {
		return t.Hour - other.Hour
	}
	if t.Minute != other.Minute {
		return t.Minute - other.Minute
	}
	if t.Second != other.Second {
		return t.Second - other.Second
	}
	return t.Nanos - other.Nanos
}

// Combine combines this time and a date to make a datetime
func (t TimeOfDay) Combine(date Date, tz *time.Location) time.Time {
	return time.Date(date.Year, time.Month(date.Month), date.Day, t.Hour, t.Minute, t.Second, t.Nanos, tz)
}

// Format formats this time of day as a string
func (t TimeOfDay) Format(layout string, loc i18n.Locale) (string, error) {
	// upgrade us to a date time so we can use standard time.Time formatting
	return Format(t.Combine(ZeroDate, time.UTC), layout, loc, TimeOnlyLayouts)
}

// String returns the ISO8601 representation
func (t TimeOfDay) String() string {
	s, _ := t.Format(iso8601Time, "")
	return s
}

// ZeroTimeOfDay is our uninitialized time of day value
var ZeroTimeOfDay = TimeOfDay{}
