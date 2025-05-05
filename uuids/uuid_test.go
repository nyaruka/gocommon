package uuids_test

import (
	"testing"
	"time"

	"github.com/nyaruka/gocommon/dates"
	"github.com/nyaruka/gocommon/uuids"
	"github.com/stretchr/testify/assert"
)

func TestNewV4(t *testing.T) {
	uuid1 := uuids.NewV4()
	uuid2 := uuids.NewV4()

	assert.Equal(t, 4, uuids.Version(string(uuid1)))
	assert.Equal(t, 4, uuids.Version(string(uuid2)))
	assert.NotEqual(t, uuid1, uuid2)
}

func TestNewV7(t *testing.T) {
	uuid1 := uuids.NewV7()
	uuid2 := uuids.NewV7()

	assert.Equal(t, 7, uuids.Version(string(uuid1)))
	assert.Equal(t, 7, uuids.Version(string(uuid2)))
	assert.NotEqual(t, uuid1, uuid2)

	u1 := uuids.NewV7()
	for range 1000000 {
		u2 := uuids.NewV7()
		assert.Greater(t, string(u2), string(u1))
		u1 = u2
	}
}

func TestSeededGenerator(t *testing.T) {
	defer uuids.SetGenerator(uuids.DefaultGenerator)

	uuids.SetGenerator(uuids.NewSeededGenerator(123456, dates.NewSequentialNow(time.Date(2024, 7, 32, 17, 29, 30, 123456, time.UTC), time.Second)))

	uuid1 := uuids.NewV4()
	uuid2 := uuids.NewV7()
	uuid3 := uuids.NewV4()

	assert.Equal(t, 4, uuids.Version(string(uuid1)))
	assert.Equal(t, 7, uuids.Version(string(uuid2)))
	assert.Equal(t, 4, uuids.Version(string(uuid3)))

	assert.Equal(t, uuids.UUID(`d2f852ec-7b4e-457f-ae7f-f8b243c49ff5`), uuid1)
	assert.Equal(t, uuids.UUID(`01910efd-5890-71e2-bd38-d266ec8d3716`), uuid2)
	assert.Equal(t, uuids.UUID(`8720f157-ca1c-432f-9c0b-2014ddc77094`), uuid3)

	uuids.SetGenerator(uuids.NewSeededGenerator(123456, dates.NewSequentialNow(time.Date(2024, 7, 32, 17, 29, 30, 123456, time.UTC), time.Second)))

	// should get same sequence again for same seed
	assert.Equal(t, uuids.UUID(`d2f852ec-7b4e-457f-ae7f-f8b243c49ff5`), uuids.NewV4())
	assert.Equal(t, uuids.UUID(`01910efd-5890-71e2-bd38-d266ec8d3716`), uuids.NewV7())
	assert.Equal(t, uuids.UUID(`8720f157-ca1c-432f-9c0b-2014ddc77094`), uuids.NewV4())
}
