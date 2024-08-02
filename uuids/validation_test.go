package uuids_test

import (
	"testing"

	"github.com/nyaruka/gocommon/uuids"
	"github.com/stretchr/testify/assert"
)

func TestIs(t *testing.T) {
	assert.True(t, uuids.Is("182faeb1-eb29-41e5-b288-c1af671ee671")) // v4
	assert.True(t, uuids.Is("182faeb1-eb29-71e5-b288-c1af671ee671")) // v7

	assert.False(t, uuids.Is(""))
	assert.False(t, uuids.Is("182faeb1"))
	assert.False(t, uuids.Is("182faeb1-eb29-41e5-b288-c1af671ee67x"))
	assert.False(t, uuids.Is("182faeb1-eb29-41e5-b288-c1af671ee67"))
	assert.False(t, uuids.Is("182faeb1-eb29-41e5-b288-c1af671ee6712"))
}

func TestVersion(t *testing.T) {
	assert.Equal(t, 4, uuids.Version("182faeb1-eb29-41e5-b288-c1af671ee671")) // v4
	assert.Equal(t, 7, uuids.Version("182faeb1-eb29-71e5-b288-c1af671ee671")) // v7

	assert.Equal(t, 0, uuids.Version(""))
	assert.Equal(t, 0, uuids.Version("182faeb1"))
}
