package cache_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/cache"
	"github.com/stretchr/testify/assert"
)

func TestLocal(t *testing.T) {
	ctx := context.Background()

	fetchCounts := make(map[string]int)
	fetchCountsMutex := &sync.Mutex{}

	fetch := func(ctx context.Context, k string) (string, error) {
		fetchCountsMutex.Lock()
		fc := fetchCounts[k]
		fc++
		fetchCounts[k] = fc
		fetchCountsMutex.Unlock()

		if k == "error" {
			return "", errors.New("boom")
		} else if k == "slow" {
			time.Sleep(250 * time.Millisecond)
		}
		return fmt.Sprintf("%s/%d", strings.ToUpper(k), fc), nil
	}
	cache := cache.NewLocal[string, string](fetch, time.Second)
	cache.Start()

	v, err := cache.Get(ctx, "x")
	assert.NoError(t, err)
	assert.Equal(t, "X/1", v)
	assert.Equal(t, map[string]int{"x": 1}, fetchCounts)
	assert.Equal(t, 1, cache.Len())

	v, err = cache.Get(ctx, "x")
	assert.NoError(t, err)
	assert.Equal(t, "X/1", v)
	assert.Equal(t, map[string]int{"x": 1}, fetchCounts)
	assert.Equal(t, 1, cache.Len())

	v, err = cache.Get(ctx, "y")
	assert.NoError(t, err)
	assert.Equal(t, "Y/1", v)
	assert.Equal(t, map[string]int{"x": 1, "y": 1}, fetchCounts)
	assert.Equal(t, 2, cache.Len())

	// test 100 threads trying to get the same value simultaneously
	wg := sync.WaitGroup{}
	getZ := func() {
		v, err = cache.Get(ctx, "z")
		assert.NoError(t, err)
		assert.Equal(t, "Z/1", v)
		wg.Done()
	}

	wg.Add(100)
	for i := 0; i < 100; i++ {
		go getZ()
	}

	wg.Wait()
	assert.Equal(t, map[string]int{"x": 1, "y": 1, "z": 1}, fetchCounts) // should only have made one fetch for z
	assert.Equal(t, 3, cache.Len())

	// test that fetching one key isn't affected by a delay fetching a different key
	wg.Add(2)
	t0 := time.Now()
	var tFast, tSlow time.Duration

	go func() {
		v, err = cache.Get(ctx, "slow")
		tSlow = time.Since(t0)
		assert.NoError(t, err)
		assert.Equal(t, "SLOW/1", v)
		wg.Done()
	}()
	go func() {
		v, err = cache.Get(ctx, "fast")
		tFast = time.Since(t0)
		assert.NoError(t, err)
		assert.Equal(t, "FAST/1", v)
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

	// wait twice as long as the TTL so cache should be empty
	time.Sleep(2 * time.Second)

	assert.Equal(t, 0, cache.Len())

	v, err = cache.Get(ctx, "x")
	assert.NoError(t, err)
	assert.Equal(t, "X/2", v)
	assert.Equal(t, map[string]int{"x": 2, "y": 1, "z": 1, "fast": 1, "slow": 1, "error": 1}, fetchCounts)
	assert.Equal(t, 1, cache.Len())

	cache.Stop()
}
