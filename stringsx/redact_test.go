package stringsx_test

import (
	"testing"

	"github.com/nyaruka/gocommon/stringsx"
	"github.com/stretchr/testify/assert"
)

func TestRedactor(t *testing.T) {
	assert.Equal(t, "hello world", stringsx.NewRedactor("****")("hello world"))                         // nothing to redact
	assert.Equal(t, "", stringsx.NewRedactor("****", "abc")(""))                                        // empty input
	assert.Equal(t, "**** def **** def", stringsx.NewRedactor("****", "abc")("abc def abc def"))        // all instances redacted
	assert.Equal(t, "**** def **** jkl", stringsx.NewRedactor("****", "abc", "ghi")("abc def ghi jkl")) // all values redacted
}
