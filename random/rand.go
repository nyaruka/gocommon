package random

import (
	crand "crypto/rand"
	"fmt"
	"io"
	"math/rand/v2"
	"sync"

	"github.com/shopspring/decimal"
)

// DefaultGenerator is the default generator for calls to Rand()
var DefaultGenerator = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
var currentGenerator = DefaultGenerator
var lock sync.Mutex

// NewSeededGenerator creates a new seeded generator
func NewSeededGenerator(seed int64) *rand.Rand {
	return rand.New(rand.NewPCG(uint64(seed), uint64(seed)))
}

// SetGenerator sets the rand used by Rand()
func SetGenerator(rnd *rand.Rand) {
	currentGenerator = rnd
}

// IntN returns a random integer in the range [0, n)
func IntN(n int) int {
	lock.Lock()
	defer lock.Unlock()
	return currentGenerator.IntN(n)
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

// String returns a string of length n composed of random characters from chars.
func String(n int, chars []rune) string {
	r := make([]rune, n)
	for i := range r {
		r[i] = chars[IntN(len(chars))]
	}
	return string(r)
}

// DefaultSecureSource is the default source of randomness for calls to SecureString()
var DefaultSecureSource io.Reader = crand.Reader
var currentSecureSource = DefaultSecureSource

// SetSecureSource sets the source of randomness used by SecureString(). It exists so that tests which compare output
// against snapshots can make secrets deterministic - see NewSeededSource. It must never be used outside of tests as
// it downgrades SecureString() to a non-cryptographic source.
func SetSecureSource(src io.Reader) {
	currentSecureSource = src
}

// NewSeededSource creates a new seeded source of randomness for use with SetSecureSource. It draws from its own stream
// so that adding or removing a SecureString() call doesn't shift the values returned by IntN() etc.
func NewSeededSource(seed int64) io.Reader {
	return &seededSource{rnd: NewSeededGenerator(seed)}
}

// adapts a math/rand/v2 generator to io.Reader
type seededSource struct {
	rnd  *rand.Rand
	lock sync.Mutex
}

func (s *seededSource) Read(p []byte) (int, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for i := 0; i < len(p); i += 8 {
		v := s.rnd.Uint64()
		for j := 0; j < 8 && i+j < len(p); j++ {
			p[i+j] = byte(v >> (8 * j))
		}
	}
	return len(p), nil
}

// SecureString returns a string of length n composed of random characters from chars, drawn from a cryptographically
// secure source of randomness. Use this rather than String() for tokens, passwords and other secrets. Panics if chars
// doesn't contain between 1 and 256 characters.
func SecureString(n int, chars []rune) string {
	k := len(chars)
	if k < 1 || k > 256 {
		panic("chars must contain between 1 and 256 characters")
	}

	// a byte only maps evenly onto k characters below the largest multiple of k that fits in a byte, so to avoid
	// modulo bias we discard values at or above that and read more
	limit := 256 - (256 % k)

	r := make([]rune, n)
	buf := make([]byte, n)

	for i := 0; i < n; {
		if _, err := io.ReadFull(currentSecureSource, buf); err != nil {
			panic(fmt.Sprintf("error reading from secure source: %s", err))
		}

		for _, b := range buf {
			if int(b) < limit {
				r[i] = chars[int(b)%k]
				i++
				if i == n {
					break
				}
			}
		}
	}

	return string(r)
}
