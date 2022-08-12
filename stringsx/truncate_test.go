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
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "", stringsx.Truncate("", 100))
	assert.Equal(t, "1234567890", stringsx.Truncate("1234567890", 100))
	assert.Equal(t, "1234567890", stringsx.Truncate("1234567890", 10))
	assert.Equal(t, "1234567", stringsx.Truncate("1234567890", 7))
	assert.Equal(t, "你喜欢我当然喜欢的电", stringsx.Truncate("你喜欢我当然喜欢的电", 100))
	assert.Equal(t, "你喜欢我当然喜欢的电", stringsx.Truncate("你喜欢我当然喜欢的电", 10))
	assert.Equal(t, "你喜欢我当然喜", stringsx.Truncate("你喜欢我当然喜欢的电", 7))
}
