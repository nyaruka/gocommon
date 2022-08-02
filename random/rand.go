package random

import (
	"math/rand"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// DefaultGenerator is the default generator for calls to Rand()
var DefaultGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))
var currentGenerator = DefaultGenerator
var lock sync.Mutex

// NewSeededGenerator creates a new seeded generator
func NewSeededGenerator(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}

// SetGenerator sets the rand used by Rand()
func SetGenerator(rnd *rand.Rand) {
	currentGenerator = rnd
}

// IntN returns a random integer in the range [0, n)
func IntN(n int) int {
	lock.Lock()
	defer lock.Unlock()
	return currentGenerator.Intn(n)
}

// Float64 returns, as a float64, a pseudo-random number in the half-open interval [0.0,1.0).
func Float64() float64 {
	lock.Lock()
	defer lock.Unlock()
	return currentGenerator.Float64()
}

// Decimal returns a random decimal in the range [0.0, 1.0)
func Decimal() decimal.Decimal {
	return decimal.NewFromFloat(Float64())
}
