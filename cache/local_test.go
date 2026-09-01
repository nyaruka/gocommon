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
	cache := cache.NewLocal[string, string](fetch, time.Second, 0)
	cache.Start()

	assert.Equal(t, "", cache.Get("x"))
	assert.Equal(t, map[string]int{}, fetchCounts)

	v, err := cache.GetOrFetch(ctx, "x")
	assert.NoError(t, err)
	assert.Equal(t, "X/1", v)
	assert.Equal(t, map[string]int{"x": 1}, fetchCounts)
	assert.Equal(t, 1, cache.Len())

	v, err = cache.GetOrFetch(ctx, "x")
	assert.NoError(t, err)
	assert.Equal(t, "X/1", v)
	assert.Equal(t, map[string]int{"x": 1}, fetchCounts)
	assert.Equal(t, 1, cache.Len())

	assert.Equal(t, "X/1", cache.Get("x"))
	assert.Equal(t, map[string]int{"x": 1}, fetchCounts)

	v, err = cache.GetOrFetch(ctx, "y")
	assert.NoError(t, err)
	assert.Equal(t, "Y/1", v)
	assert.Equal(t, map[string]int{"x": 1, "y": 1}, fetchCounts)
	assert.Equal(t, 2, cache.Len())

	// test 100 threads trying to get the same value simultaneously
	wg := sync.WaitGroup{}
	getZ := func() {
		vZ, errZ := cache.GetOrFetch(ctx, "z")
		assert.NoError(t, errZ)
		assert.Equal(t, "Z/1", vZ)
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
		vSlow, errSlow := cache.GetOrFetch(ctx, "slow")
		tSlow = time.Since(t0)
		assert.NoError(t, errSlow)
		assert.Equal(t, "SLOW/1", vSlow)
		wg.Done()
	}()
	go func() {
		vFast, errFast := cache.GetOrFetch(ctx, "fast")
		tFast = time.Since(t0)
		assert.NoError(t, errFast)
		assert.Equal(t, "FAST/1", vFast)
		wg.Done()
	}()

	wg.Wait()

	assert.Less(t, tFast, 100*time.Millisecond)
	assert.GreaterOrEqual(t, tSlow, 250*time.Millisecond)
	assert.Equal(t, 5, cache.Len())

	v, err = cache.GetOrFetch(ctx, "error")
	assert.EqualError(t, err, "boom")
	assert.Equal(t, "", v)

	assert.Equal(t, 5, cache.Len())

	// wait twice as long as the TTL so cache should be empty
	time.Sleep(2 * time.Second)

	assert.Equal(t, 0, cache.Len())

	v, err = cache.GetOrFetch(ctx, "x")
	assert.NoError(t, err)
	assert.Equal(t, "X/2", v)
	assert.Equal(t, map[string]int{"x": 2, "y": 1, "z": 1, "fast": 1, "slow": 1, "error": 1}, fetchCounts)
	assert.Equal(t, 1, cache.Len())

	// can also explicity set items
	cache.Set("a", "123")
	cache.Set("x", "234")
	assert.Equal(t, 2, cache.Len())

	cache.Clear()

	assert.Equal(t, 0, cache.Len())

	cache.Stop()
}

func TestLocalWithCapacity(t *testing.T) {
	ctx := context.Background()

	c := cache.NewLocal(func(ctx context.Context, k string) (string, error) {
		return "v" + k, nil
	}, time.Minute, 3)

	for _, k := range []string{"a", "b", "c"} {
		c.GetOrFetch(ctx, k)
	}
	assert.Equal(t, 3, c.Len())

	// reading an item makes it the most recently used, even though reads don't extend its TTL
	assert.Equal(t, "va", c.Get("a"))

	// so making room for a new item evicts "b" rather than "a"
	c.GetOrFetch(ctx, "d")
	assert.Equal(t, 3, c.Len())
	assert.Equal(t, "va", c.Get("a"))
	assert.Equal(t, "", c.Get("b"))
	assert.Equal(t, "vc", c.Get("c"))
	assert.Equal(t, "vd", c.Get("d"))

	// which means keys we never see again can't grow the cache past its capacity
	for i := range 10000 {
		c.GetOrFetch(ctx, fmt.Sprintf("flood-%d", i))
	}
	assert.Equal(t, 3, c.Len())

	// whereas a zero capacity means no bound at all
	u := cache.NewLocal(func(ctx context.Context, k string) (string, error) {
		return "v" + k, nil
	}, time.Minute, 0)

	for i := range 10000 {
		u.GetOrFetch(ctx, fmt.Sprintf("flood-%d", i))
	}
	assert.Equal(t, 10000, u.Len())
}
