package dates

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/nyaruka/gocommon/i18n"
)

// Custom date/time formatting using layout strings like YYYY-MM-DD

const (
	dateSeq     int = 1
	timeSeq     int = 2
	dateTimeSeq int = 4
)

// LayoutType describes what layout sequences are permitted in a formatting operation
type LayoutType uint

// formatting mode constants
const (
	DateOnlyLayouts = LayoutType(dateSeq)
	TimeOnlyLayouts = LayoutType(timeSeq)
	DateTimeLayouts = LayoutType(dateSeq | timeSeq | dateTimeSeq)
)

// Includes returns whether the given sequence type is included in this layout type
func (t LayoutType) Includes(seqType int) bool {
	return LayoutType(seqType)&t != 0
}

// String converts a layout type to a string - used for error messages
func (t LayoutType) String() string {
	switch t {
	case DateOnlyLayouts:
		return "date"
	case TimeOnlyLayouts:
		return "time"
	default:
		return "datetime"
	}
}

// LayoutMode describes what a layout is being used for
type LayoutMode int

// formatting mode constants
const (
	FormattingMode LayoutMode = 1
	ParsingMode    LayoutMode = 2
)

// String converts a layout mode to a string - used for error messages
func (m LayoutMode) String() string {
	if m == FormattingMode {
		return "formatting"
	}
	return "parsing"
}

// valid sequences that can occur in a layout string
var layoutSequences = map[string]struct {
	mapped    string
	seqType   int
	parseable bool
}{
	"YY":        {"06", dateSeq, true},
	"YYYY":      {"2006", dateSeq, true},
	"M":         {"1", dateSeq, true},
	"MM":        {"01", dateSeq, true},
	"MMM":       {"Jan", dateSeq, false},
	"MMMM":      {"January", dateSeq, false},
	"D":         {"2", dateSeq, true},
	"DD":        {"02", dateSeq, true},
	"EEE":       {"Mon", dateSeq, false},
	"EEEE":      {"Monday", dateSeq, false},
	"fffffffff": {"000000000", timeSeq, true},
	"ffffff":    {"000000", timeSeq, true},
	"fff":       {"000", timeSeq, true},
	"h":         {"3", timeSeq, true},
	"hh":        {"03", timeSeq, true},
	"t":         {"15", timeSeq, true}, // handled as special case in formatting code
	"tt":        {"15", timeSeq, true},
	"m":         {"4", timeSeq, true},
	"mm":        {"04", timeSeq, true},
	"s":         {"5", timeSeq, true},
	"ss":        {"05", timeSeq, true},
	"aa":        {"pm", timeSeq, true},
	"AA":        {"PM", timeSeq, true},
	"Z":         {"Z07:00", dateTimeSeq, true},
	"ZZZ":       {"-07:00", dateTimeSeq, true},
}

// non-sequence runes that are permitted in layout strings
var ignoredFormattingRunes = map[rune]bool{' ': true, ':': true, '/': true, '.': true, ',': true, 'T': true, '-': true, '_': true}

// ValidateFormat parses a formatting layout string to validate it
func ValidateFormat(layout string, type_ LayoutType, mode LayoutMode) error {
	return visitLayout(layout, type_, mode, nil)
}

// Format formats a date/time value using a layout string.
//
// If type is DateOnlyLayouts or DateTimeLayouts, the following sequences are accepted:
//
//	`YY`        - last two digits of year 0-99
//	`YYYY`      - four digits of your 0000-9999
//	`M`         - month 1-12
//	`MM`        - month 01-12
//	`MMM`       - month Jan-Dec (localized using given locale)
//	`MMMM`      - month January-December (localized using given locale)
//	`D`         - day of month 1-31
//	`DD`        - day of month, zero padded 0-31
//	`EEE`       - day of week Mon-Sun (localized using given locale)
//	`EEEE`      - day of week Monday-Sunday (localized using given locale)
//
// If type is TimeOnlyLayouts or DateTimeLayouts, the following sequences are accepted:
//
//	`h`         - hour of the day 1-12
//	`hh`        - hour of the day 01-12
//	`t`         - twenty four hour of the day 0-23
//	`tt`        - twenty four hour of the day 00-23
//	`m`         - minute 0-59
//	`mm`        - minute 00-59
//	`s`         - second 0-59
//	`ss`        - second 00-59
//	`fff`       - milliseconds
//	`ffffff`    - microseconds
//	`fffffffff` - nanoseconds
//	`aa`        - am or pm (localized using given locale)
//	`AA`        - AM or PM (localized using given locale)
//
// If type is DateTimeLayouts, the following sequences are accepted:
//
//	`Z`         - hour and minute offset from UTC, or Z for UTC
//	`ZZZ`       - hour and minute offset from UTC
//
// The following chars are allowed and ignored: ' ', ':', ',', 'T', '-', '_', '/'
func Format(t time.Time, layout string, locale i18n.Locale, type_ LayoutType) (string, error) {
	output := bytes.Buffer{}

	translation := GetTranslation(locale)

	handleSeq := func(seq, mapped string) {
		out := ""

		switch mapped {
		case "January":
			out = translation.Months[t.Month()-1]
		case "Jan":
			out = translation.ShortMonths[t.Month()-1]
		case "Monday":
			out = translation.Days[t.Weekday()]
		case "Mon":
			out = translation.ShortDays[t.Weekday()]
		case "PM", "pm":
			i := 0
			if t.Hour() >= 12 {
				i = 1
			}
			out = translation.AmPm[i]
			if mapped == "PM" {
				out = strings.ToUpper(out)
			} else {
				out = strings.ToLower(out)
			}
		case "15":
			// go formatting has no way of specifying 24 hour without zero padding
			// so if user specified a single char, trim off the zero-padding
			out = t.Format("15")
			if seq == "t" {
				out = strings.TrimLeft(out, "0")
			}
		case "000000000", "000000", "000":
			// go only formats these after a period
			out = t.Format("." + mapped)[1:]
		case "":
			out = seq // a sequence of ignored chars
		default:
			out = t.Format(mapped)
		}
		output.WriteString(out)
	}

	if err := visitLayout(layout, type_, FormattingMode, handleSeq); err != nil {
		return "", err
	}

	return output.String(), nil
}

// converts a format layout to the go/time syntax, e.g. "YYYY-MM" -> "2006-01"
func convertLayout(layout string, type_ LayoutType, mode LayoutMode) (string, error) {
	output := bytes.Buffer{}

	handleSeq := func(seq, mapped string) {
		if mapped != "" {
			output.WriteString(mapped)
		} else {
			output.WriteString(seq)
		}
	}

	if err := visitLayout(layout, type_, mode, handleSeq); err != nil {
		return "", err
	}

	return output.String(), nil
}

// parses a layout string, invoking the given callback for every mappable sequence or sequence of ignored chars
func visitLayout(layout string, type_ LayoutType, mode LayoutMode, callback func(string, string)) error {
	runes := []rune(layout)
	var seqLen int

	for i := 0; i < len(runes); i += seqLen {
		r := runes[i]
		ignored := ignoredFormattingRunes[r]

		// peek to see how many repeated occurences of r there are
		for seqLen = 1; (i + seqLen) < len(runes); seqLen++ {
			rx := runes[i+seqLen]
			if (ignored && !ignoredFormattingRunes[rx]) || rx != r {
				break
			}
		}

		seq := string(runes[i : i+seqLen]) // e.g. "YYYY", "tt"
		mapped := ""                       // e.g. "2006", "15"

		if !ignored {
			layoutSeq, exists := layoutSequences[seq]
			if exists && type_.Includes(layoutSeq.seqType) && (mode != ParsingMode || layoutSeq.parseable) {
				mapped = layoutSeq.mapped
			} else {
				return fmt.Errorf("'%s' is not valid in a %s %s layout", seq, type_, mode)
			}
		}

		if callback != nil {
			callback(seq, mapped)
		}
	}
	return nil
}
