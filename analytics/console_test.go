package analytics_test

import (
	"strings"
	"testing"

	"github.com/nyaruka/gocommon/analytics"
	"github.com/stretchr/testify/assert"
)

func TestConsoleBackend(t *testing.T) {
	out := &strings.Builder{}

	b := analytics.NewConsole(out)
	assert.Equal(t, "console", b.Name())

	analytics.RegisterBackend(b)
	assert.NoError(t, analytics.Start())

	analytics.Gauge("foo", 123456)
	analytics.Gauge("bar", 0.1234)

	assert.NoError(t, analytics.Stop())

	assert.Equal(t, "[analytics] gauge=foo value=123456.00\n[analytics] gauge=bar value=0.12\n", out.String())
}
