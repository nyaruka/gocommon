package stringsx_test

import (
	"testing"

	"github.com/nyaruka/gocommon/stringsx"
	"github.com/stretchr/testify/assert"
)

func TestSkeleton(t *testing.T) {
	assert.Equal(t, "", stringsx.Skeleton(""))
	assert.Equal(t, "foo", stringsx.Skeleton("foo"))
	assert.Equal(t, "nyaruka", stringsx.Skeleton("𝕟𝔂𝛼𝐫ᴜ𝞳𝕒"))
}

func TestConfusable(t *testing.T) {
	assert.True(t, stringsx.Confusable("", ""))
	assert.True(t, stringsx.Confusable("foo", "foo"))
	assert.True(t, stringsx.Confusable("١", "۱"))     // 0x661 vs 0x6f1
	assert.True(t, stringsx.Confusable("بلی", "بلى")) // 0x6cc vs 0x649
	assert.True(t, stringsx.Confusable("nyaruka", "𝕟𝔂𝛼𝐫ᴜ𝞳𝕒"))

	assert.False(t, stringsx.Confusable("foo", "bar"))
	assert.False(t, stringsx.Confusable("foo", "Foo"))
}
