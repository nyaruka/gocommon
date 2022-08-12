package stringsx

import "strings"

// Redactor is a function which can redact the given string
type Redactor func(s string) string

// NewRedactor creates a new redaction function which replaces the given values
func NewRedactor(mask string, values ...string) Redactor {
	// convert list of redaction values to list of replacements with mask
	replacements := make([]string, len(values)*2)
	for i := range values {
		replacements[i*2] = values[i]
		replacements[i*2+1] = mask
	}
	return strings.NewReplacer(replacements...).Replace
}
