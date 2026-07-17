package stringsx_test

import (
	"testing"

	"github.com/nyaruka/gocommon/stringsx"
	"github.com/stretchr/testify/assert"
)

func TestTruncateEllipsis(t *testing.T) {
	assert.Equal(t, "", stringsx.TruncateEllipsis("", 100))
	assert.Equal(t, "1234567890", stringsx.TruncateEllipsis("1234567890", 100))
	assert.Equal(t, "1234567890", stringsx.TruncateEllipsis("1234567890", 10))
	assert.Equal(t, "1234...", stringsx.TruncateEllipsis("1234567890", 7))
	assert.Equal(t, "你喜欢我当然喜欢的电", stringsx.TruncateEllipsis("你喜欢我当然喜欢的电", 100))
	assert.Equal(t, "你喜欢我当然喜欢的电", stringsx.TruncateEllipsis("你喜欢我当然喜欢的电", 10))
	assert.Equal(t, "你喜欢我...", stringsx.TruncateEllipsis("你喜欢我当然喜欢的电", 7))

	// limits smaller than the ellipsis must not panic - just hard-truncate to the limit
	assert.Equal(t, "123", stringsx.TruncateEllipsis("1234567890", 3))
	assert.Equal(t, "12", stringsx.TruncateEllipsis("1234567890", 2))
	assert.Equal(t, "1", stringsx.TruncateEllipsis("1234567890", 1))
	assert.Equal(t, "", stringsx.TruncateEllipsis("1234567890", 0))
	assert.Equal(t, "", stringsx.TruncateEllipsis("1234567890", -1))
	assert.Equal(t, "你", stringsx.TruncateEllipsis("你喜欢我当然喜欢的电", 1))
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "", stringsx.Truncate("", 100))
	assert.Equal(t, "1234567890", stringsx.Truncate("1234567890", 100))
	assert.Equal(t, "1234567890", stringsx.Truncate("1234567890", 10))
	assert.Equal(t, "1234567", stringsx.Truncate("1234567890", 7))
	assert.Equal(t, "你喜欢我当然喜欢的电", stringsx.Truncate("你喜欢我当然喜欢的电", 100))
	assert.Equal(t, "你喜欢我当然喜欢的电", stringsx.Truncate("你喜欢我当然喜欢的电", 10))
	assert.Equal(t, "你喜欢我当然喜", stringsx.Truncate("你喜欢我当然喜欢的电", 7))

	// zero and negative limits must not panic
	assert.Equal(t, "", stringsx.Truncate("1234567890", 0))
	assert.Equal(t, "", stringsx.Truncate("1234567890", -1))
}
