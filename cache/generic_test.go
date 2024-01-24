package cache_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/cache"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	ctx := context.Background()

	var fetches atomic.Int32
	fetch := func(ctx context.Context, k string) (string, error) {
		fetches.Add(1)
		if k == "error" {
			return "", errors.New("boom")
		} else if k == "slow" {
			time.Sleep(250 * time.Millisecond)
		}
		return strings.ToUpper(k), nil
	}
	cache := cache.NewCache[string, string](fetch, time.Second)
	cache.Start()

	v, err := cache.Get(ctx, "x")
	assert.NoError(t, err)
	assert.Equal(t, "X", v)
	assert.Equal(t, int32(1), fetches.Load())
	assert.Equal(t, 1, cache.Len())

	v, err = cache.Get(ctx, "x")
	assert.NoError(t, err)
	assert.Equal(t, "X", v)
	assert.Equal(t, int32(1), fetches.Load())
	assert.Equal(t, 1, cache.Len())

	v, err = cache.Get(ctx, "y")
	assert.NoError(t, err)
	assert.Equal(t, "Y", v)
	assert.Equal(t, int32(2), fetches.Load())
	assert.Equal(t, 2, cache.Len())

	// test 100 threads trying to get the same value simultaneously
	wg := sync.WaitGroup{}
	getZ := func() {
		v, err = cache.Get(ctx, "z")
		assert.NoError(t, err)
		assert.Equal(t, "Z", v)
		wg.Done()
	}

	wg.Add(100)
	for i := 0; i < 100; i++ {
		go getZ()
	}

	wg.Wait()
	assert.Equal(t, int32(3), fetches.Load()) // should still only have made one fetch
	assert.Equal(t, 3, cache.Len())

	// test that fetching one key isn't affected by a delay fetching a different key
	wg.Add(2)
	t0 := time.Now()
	var tFast, tSlow time.Duration

	go func() {
		v, err = cache.Get(ctx, "slow")
		tSlow = time.Since(t0)
		assert.NoError(t, err)
		assert.Equal(t, "SLOW", v)
		wg.Done()
	}()
	go func() {
		v, err = cache.Get(ctx, "fast")
		tFast = time.Since(t0)
		assert.NoError(t, err)
		assert.Equal(t, "FAST", v)
		wg.Done()
	}()

	wg.Wait()

	assert.Less(t, tFast, 100*time.Millisecond)
	assert.GreaterOrEqual(t, tSlow, 250*time.Millisecond)
	assert.Equal(t, 5, cache.Len())

	v, err = cache.Get(ctx, "error")
	assert.EqualError(t, err, "boom")
	assert.Equal(t, "", v)

	assert.Equal(t, 5, cache.Len())

	time.Sleep(2 * time.Second)

	assert.Equal(t, 0, cache.Len())

	cache.Stop()
}
