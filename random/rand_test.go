package random_test

import (
	"sync"
	"testing"

	"github.com/nyaruka/gocommon/random"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestRand(t *testing.T) {
	defer random.SetGenerator(random.DefaultGenerator)
	random.SetGenerator(random.NewSeededGenerator(1234))

	assert.Equal(t, 0, random.IntN(10))
	assert.Equal(t, 8, random.IntN(10))
	assert.Equal(t, 4, random.IntN(10))
	assert.Equal(t, decimal.RequireFromString("0.7189806938374759"), random.Decimal())
	assert.Equal(t, decimal.RequireFromString("0.824272697040096"), random.Decimal())
	assert.Equal(t, decimal.RequireFromString("0.10545532824596571"), random.Decimal())

	assert.Equal(t, "lJ4ZfHEr25", random.String(10, []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/")))
	assert.Equal(t, "zzzaaz!aaz", random.String(10, []rune("a!z")))
}

func TestRandConcurrency(t *testing.T) {
	runConcurrently(100000, func(int) { random.IntN(10); random.Decimal(); random.Float64() })
}

func runConcurrently(times int, fn func(int)) {
	wg := &sync.WaitGroup{}
	for i := 0; i < times; i++ {
		wg.Add(1)
		go func(t int) { defer wg.Done(); fn(t) }(i)
	}
	wg.Wait()
}
