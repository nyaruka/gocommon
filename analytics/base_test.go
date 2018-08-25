package analytics_test

import (
	"testing"
	"time"

	"github.com/nyaruka/gocommon/analytics"
)

func TestAnalytics(t *testing.T) {
	// all methods are NOOPs if analytics has been configured
	analytics.Start()
	analytics.Gauge("foo.bar", 123.45)
	analytics.Stop()

	analytics.Librato("bob", "1234567", "foo.com", time.Second*30, nil)
}
