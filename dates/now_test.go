package dates_test

import (
	"testing"
	"time"

	"github.com/nyaruka/gocommon/dates"

	"github.com/stretchr/testify/assert"
)

func TestNowFuncs(t *testing.T) {
	defer dates.SetNowFunc(time.Now)

	d1 := time.Date(2018, 7, 5, 16, 29, 30, 123456, time.UTC)
	dates.SetNowFunc(dates.NewFixedNow(d1))

	assert.Equal(t, time.Date(2018, 7, 5, 16, 29, 30, 123456, time.UTC), dates.Now())
	assert.Equal(t, time.Date(2018, 7, 5, 16, 29, 30, 123456, time.UTC), dates.Now())

	dates.SetNowFunc(dates.NewSequentialNow(d1, time.Second))

	assert.Equal(t, time.Date(2018, 7, 5, 16, 29, 30, 123456, time.UTC), dates.Now())
	assert.Equal(t, time.Date(2018, 7, 5, 16, 29, 31, 123456, time.UTC), dates.Now())
	assert.Equal(t, time.Date(2018, 7, 5, 16, 29, 32, 123456, time.UTC), dates.Now())

	assert.Equal(t, time.Second*3, dates.Since(time.Date(2018, 7, 5, 16, 29, 30, 123456, time.UTC)))
}
