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

	assert.Equal(t, 2, random.IntN(10))
	assert.Equal(t, 5, random.IntN(10))
	assert.Equal(t, 9, random.IntN(10))
	assert.Equal(t, decimal.RequireFromString("0.8989115230327291"), random.Decimal())
	assert.Equal(t, decimal.RequireFromString("0.6087185537746531"), random.Decimal())
	assert.Equal(t, decimal.RequireFromString("0.3023554328904116"), random.Decimal())
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
