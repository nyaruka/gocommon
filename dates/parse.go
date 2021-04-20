package dates

import "time"

// ZeroDateTime is our uninitialized datetime value
var ZeroDateTime = time.Time{}

// ParseDate parses the given string into a datetime
func ParseDateTime(layout string, value string, tz *time.Location) (time.Time, error) {
	goFormat, err := convertFormat(layout, true, DateTimeFormatting)
	if err != nil {
		return ZeroDateTime, err
	}

	dt, err := time.ParseInLocation(goFormat, value, tz)
	if err != nil {
		return ZeroDateTime, err
	}

	return dt, nil
}

// ParseDate parses the given string into a date
func ParseDate(layout string, value string) (Date, error) {
	goFormat, err := convertFormat(layout, true, DateOnlyFormatting)
	if err != nil {
		return ZeroDate, err
	}

	dt, err := time.Parse(goFormat, value)
	if err != nil {
		return ZeroDate, err
	}

	return ExtractDate(dt), nil
}

// ParseTimeOfDay parses the given string into a time of day
func ParseTimeOfDay(layout string, value string) (TimeOfDay, error) {
	goFormat, err := convertFormat(layout, true, TimeOnlyFormatting)
	if err != nil {
		return ZeroTimeOfDay, err
	}

	dt, err := time.Parse(goFormat, value)
	if err != nil {
		return ZeroTimeOfDay, err
	}

	return ExtractTimeOfDay(dt), nil
}
