package analytics_test

import (
	"testing"
	"time"

	"github.com/nyaruka/gocommon/analytics"
	"github.com/stretchr/testify/assert"
)

func TestLibratoBackend(t *testing.T) {
	b := analytics.NewLibrato("nyaruka", "12345678", "box1", time.Second, nil)
	assert.Equal(t, "librato", b.Name())
}
