package analytics_test

import (
	"testing"

	"github.com/nyaruka/gocommon/analytics"
	"github.com/stretchr/testify/assert"
)

func TestMockBackend(t *testing.T) {
	b := analytics.NewMock()
	assert.Equal(t, "mock", b.Name())

	analytics.RegisterBackend(b)
	assert.NoError(t, analytics.Start())

	analytics.Gauge("foo", 123456)
	analytics.Gauge("foo", 567)
	analytics.Gauge("bar", 0.1234)

	assert.NoError(t, analytics.Stop())

	assert.Equal(t, map[string][]float64{
		"foo": {123456, 567},
		"bar": {0.1234},
	}, b.Gauges)
}
