package dates_test

import (
	"testing"
	"time"

	"github.com/nyaruka/gocommon/dates"

	"github.com/stretchr/testify/assert"
)

func TestTimeSources(t *testing.T) {
	defer dates.SetNowSource(dates.DefaultNowSource)

	d1 := time.Date(2018, 7, 5, 16, 29, 30, 123456, time.UTC)
	dates.SetNowSource(dates.NewFixedNowSource(d1))

	assert.Equal(t, time.Date(2018, 7, 5, 16, 29, 30, 123456, time.UTC), dates.Now())
	assert.Equal(t, time.Date(2018, 7, 5, 16, 29, 30, 123456, time.UTC), dates.Now())

	dates.SetNowSource(dates.NewSequentialNowSource(d1))

	assert.Equal(t, time.Date(2018, 7, 5, 16, 29, 30, 123456, time.UTC), dates.Now())
	assert.Equal(t, time.Date(2018, 7, 5, 16, 29, 31, 123456, time.UTC), dates.Now())
	assert.Equal(t, time.Date(2018, 7, 5, 16, 29, 32, 123456, time.UTC), dates.Now())

	assert.Equal(t, time.Second*3, dates.Since(time.Date(2018, 7, 5, 16, 29, 30, 123456, time.UTC)))
}
