package random_test

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/nyaruka/gocommon/random"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	// unlike SecureString, String isn't limited to alphabets that fit in a byte
	assert.Len(t, random.String(10, []rune(strings.Repeat("x", 300))), 10)

	assert.Panics(t, func() { random.String(10, []rune("")) })
}

func TestSecureString(t *testing.T) {
	defer random.SetSecureSource(random.DefaultSecureSource)

	base64Chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/")

	// with a seeded source, output is deterministic so it can be used in snapshots
	random.SetSecureSource(random.NewSeededSource(1234))

	assert.Equal(t, "ViueeFXROP", random.SecureString(10, base64Chars))
	assert.Equal(t, "4ELiwUJ1aY", random.SecureString(10, base64Chars))

	// and is repeatable from the same seed
	random.SetSecureSource(random.NewSeededSource(1234))

	assert.Equal(t, "ViueeFXROP", random.SecureString(10, base64Chars))

	// with the default source we can only assert shape
	random.SetSecureSource(random.DefaultSecureSource)

	assert.Equal(t, "", random.SecureString(0, base64Chars))

	seen := make(map[string]bool, 1000)
	for range 1000 {
		s := random.SecureString(20, base64Chars)

		require.Len(t, s, 20)
		require.False(t, seen[s], "returned duplicate string %s", s)
		seen[s] = true

		for _, c := range s {
			require.True(t, strings.ContainsRune(string(base64Chars), c), "unexpected character %c", c)
		}
	}

	// alphabets whose size doesn't divide 256 are still unbiased because we discard biased values
	counts := make(map[rune]int, 3)
	for range 30000 {
		for _, c := range random.SecureString(1, []rune("abc")) {
			counts[c]++
		}
	}
	for _, c := range []rune("abc") {
		assert.InEpsilon(t, 10000, counts[c], 0.1, "unbalanced distribution for %c", c)
	}

	assert.Panics(t, func() { random.SecureString(10, []rune("")) })

	// SecureString draws bytes, so it can't index into an alphabet larger than 256
	assert.Panics(t, func() { random.SecureString(10, []rune(strings.Repeat("x", 257))) })

	// a source that can't be read from is a broken test setup, so we panic rather than return a weak secret
	random.SetSecureSource(failingSource{})

	assert.Panics(t, func() { random.SecureString(10, base64Chars) })
}

// a source that always fails, to test that SecureString panics rather than returning a partial string
type failingSource struct{}

func (failingSource) Read(p []byte) (int, error) { return 0, errors.New("source unavailable") }

func TestSecureStringConcurrency(t *testing.T) {
	defer random.SetSecureSource(random.DefaultSecureSource)
	random.SetSecureSource(random.NewSeededSource(1234))

	runConcurrently(10000, func(int) { random.SecureString(10, []rune("abcdef")) })
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
