package dates

import (
	"bytes"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Custom date/time formatting using layout strings like YYYY-MM-DD

const (
	dateSeq     int = 1
	timeSeq     int = 2
	dateTimeSeq int = 4
)

// FormattingMode describes what layout sequences are permitted in a formatting operation
type FormattingMode uint

// formatting mode constants
const (
	DateOnlyFormatting = FormattingMode(dateSeq)
	TimeOnlyFormatting = FormattingMode(timeSeq)
	DateTimeFormatting = FormattingMode(dateSeq | timeSeq | dateTimeSeq)
)

// Includes returns whether the given sequence type is included in this formatting mode
func (m FormattingMode) Includes(seqType int) bool {
	return FormattingMode(seqType)&m != 0
}

// String converts formatting mode to a string - used for error messages
func (m FormattingMode) String() string {
	switch m {
	case DateOnlyFormatting:
		return "date"
	case TimeOnlyFormatting:
		return "time"
	default:
		return "datetime"
	}
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
func ValidateFormat(layout string, parseable bool, mode FormattingMode) error {
	return visitFormatLayout(layout, mode, parseable, nil)
}

// Format formats a date/time value using a layout string.
//
// If mode is DateOnlyFormatting or DateTimeFormatting, the following sequences are accepted:
//
//  `YY`        - last two digits of year 0-99
//  `YYYY`      - four digits of your 0000-9999
//  `M`         - month 1-12
//  `MM`        - month 01-12
//  `MMM`       - month Jan-Dec (localized using given locale)
//  `MMMM`      - month January-December (localized using given locale)
//  `D`         - day of month 1-31
//  `DD`        - day of month, zero padded 0-31
//  `EEE`       - day of week Mon-Sun (localized using given locale)
//  `EEEE`      - day of week Monday-Sunday (localized using given locale)
//
// If mode is TimeOnlyFormatting or DateTimeFormatting, the following sequences are accepted:
//
//  `h`         - hour of the day 1-12
//  `hh`        - hour of the day 01-12
//  `t`         - twenty four hour of the day 0-23
//  `tt`        - twenty four hour of the day 00-23
//  `m`         - minute 0-59
//  `mm`        - minute 00-59
//  `s`         - second 0-59
//  `ss`        - second 00-59
//  `fff`       - milliseconds
//  `ffffff`    - microseconds
//  `fffffffff` - nanoseconds
//  `aa`        - am or pm (localized using given locale)
//  `AA`        - AM or PM (localized using given locale)
//
// If mode is DateTimeFormatting, the following sequences are accepted:
//
//  `Z`         - hour and minute offset from UTC, or Z for UTC
//  `ZZZ`       - hour and minute offset from UTC
//
// The following chars are allowed and ignored: ' ', ':', ',', 'T', '-', '_', '/'
//
func Format(t time.Time, layout string, locale string, mode FormattingMode) (string, error) {
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

	if err := visitFormatLayout(layout, mode, false, handleSeq); err != nil {
		return "", err
	}

	return output.String(), nil
}

// converts a format layout to the go/time syntax, e.g. "YYYY-MM" -> "2006-01"
func convertFormat(layout string, parseable bool, mode FormattingMode) (string, error) {
	output := bytes.Buffer{}

	handleSeq := func(seq, mapped string) {
		if mapped != "" {
			output.WriteString(mapped)
		} else {
			output.WriteString(seq)
		}
	}

	if err := visitFormatLayout(layout, mode, parseable, handleSeq); err != nil {
		return "", err
	}

	return output.String(), nil
}

// parses a layout string, invoking the given callback for every mappable sequence or sequence of ignored chars
func visitFormatLayout(layout string, mode FormattingMode, parseable bool, callback func(string, string)) error {
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
			if exists && mode.Includes(layoutSeq.seqType) && (!parseable || layoutSeq.parseable) {
				mapped = layoutSeq.mapped
			} else {
				return errors.Errorf("'%s' is not valid in a %s format", seq, mode)
			}
		}

		if callback != nil {
			callback(seq, mapped)
		}
	}
	return nil
}
