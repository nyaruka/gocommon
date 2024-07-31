package uuids_test

import (
	"testing"

	"github.com/nyaruka/gocommon/uuids"
	"github.com/stretchr/testify/assert"
)

func TestIsV4(t *testing.T) {
	assert.False(t, uuids.IsV4(""))
	assert.True(t, uuids.IsV4("182faeb1-eb29-41e5-b288-c1af671ee671"))
	assert.False(t, uuids.IsV4("182faeb1-eb29-71e5-b288-c1af671ee671"))
	assert.False(t, uuids.IsV4("182faeb1-eb29-41e5-b288-c1af671ee67x"))
	assert.False(t, uuids.IsV4("182faeb1-eb29-41e5-b288-c1af671ee67"))
	assert.False(t, uuids.IsV4("182faeb1-eb29-41e5-b288-c1af671ee6712"))
}

func TestIsV7(t *testing.T) {
	assert.False(t, uuids.IsV7(""))
	assert.True(t, uuids.IsV7("182faeb1-eb29-71e5-b288-c1af671ee671"))
	assert.False(t, uuids.IsV7("182faeb1-eb29-41e5-b288-c1af671ee671"))
	assert.False(t, uuids.IsV7("182faeb1-eb29-71e5-b288-c1af671ee67x"))
	assert.False(t, uuids.IsV7("182faeb1-eb29-71e5-b288-c1af671ee67"))
	assert.False(t, uuids.IsV7("182faeb1-eb29-71e5-b288-c1af671ee6712"))
}
