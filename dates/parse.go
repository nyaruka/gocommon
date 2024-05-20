package dates

import (
	"fmt"
	"time"
)

// ZeroDateTime is our uninitialized datetime value
var ZeroDateTime = time.Time{}

// ParseDate parses the given string into a datetime
func ParseDateTime(layout string, value string, tz *time.Location) (time.Time, error) {
	goFormat, err := convertLayout(layout, DateTimeLayouts, ParsingMode)
	if err != nil {
		return ZeroDateTime, err
	}

	dt, err := time.ParseInLocation(goFormat, value, tz)
	if err != nil {
		return ZeroDateTime, parseError(err)
	}

	return dt, nil
}

// ParseDate parses the given string into a date
func ParseDate(layout string, value string) (Date, error) {
	goFormat, err := convertLayout(layout, DateOnlyLayouts, ParsingMode)
	if err != nil {
		return ZeroDate, err
	}

	dt, err := time.Parse(goFormat, value)
	if err != nil {
		return ZeroDate, parseError(err)
	}

	return ExtractDate(dt), nil
}

// ParseTimeOfDay parses the given string into a time of day
func ParseTimeOfDay(layout string, value string) (TimeOfDay, error) {
	goFormat, err := convertLayout(layout, TimeOnlyLayouts, ParsingMode)
	if err != nil {
		return ZeroTimeOfDay, err
	}

	dt, err := time.Parse(goFormat, value)
	if err != nil {
		return ZeroTimeOfDay, parseError(err)
	}

	return ExtractTimeOfDay(dt), nil
}

// converts a time.ParseError into a more helpful message
func parseError(err error) error {
	switch typed := err.(type) {
	case *time.ParseError:
		// reverse map the go layout element to the original layout sequence
		origLayoutSeq := typed.LayoutElem
		for seq, layoutSeq := range layoutSequences {
			if layoutSeq.mapped == typed.LayoutElem {
				origLayoutSeq = seq
				break
			}
		}

		return fmt.Errorf("cannot parse '%s' as '%s'", typed.ValueElem, origLayoutSeq)
	default:
		return err
	}
}
