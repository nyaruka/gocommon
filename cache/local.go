package cache

import (
	"context"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"golang.org/x/sync/singleflight"
)

// Local is a generic in-memory cache with builtin in fetching of missing items.
type Local[K ~string, V any] struct {
	cache     *ttlcache.Cache[K, V]
	fetch     Fetcher[K, V]
	fetchSync singleflight.Group
}

// Fetcher is a function which can fetch an item which doesn't yet exist in the cache.
type Fetcher[K ~string, V any] func(context.Context, K) (V, error)

// NewLocal creates a new in-memory cache.
func NewLocal[K ~string, V any](fetch Fetcher[K, V], ttl time.Duration) *Local[K, V] {
	return &Local[K, V]{
		cache: ttlcache.New[K, V](
			ttlcache.WithTTL[K, V](ttl),
			ttlcache.WithDisableTouchOnHit[K, V](),
		),
		fetch: fetch,
	}
}

// Start starts the routine to eliminate expired items from the cache.
func (c *Local[K, V]) Start() {
	go c.cache.Start()
}

// Stop stops that routine.
func (c *Local[K, V]) Stop() {
	c.cache.Stop()
}

// Len returns the number of items in the cache.
func (c *Local[K, V]) Len() int {
	return c.cache.Len()
}

func (c *Local[K, V]) Get(ctx context.Context, key K) (V, error) {
	item := c.cache.Get(key)

	if item == nil {
		var err error

		item, err = c.fetchAndSetSynced(ctx, key)
		if err != nil {
			var zero V
			return zero, err
		}
	}

	return item.Value(), nil
}

// Clear removes all items from the cache.
func (c *Local[K, V]) Clear() {
	c.cache.DeleteAll()
}

func (c *Local[K, V]) fetchAndSetSynced(ctx context.Context, key K) (*ttlcache.Item[K, V], error) {
	ii, err, _ := c.fetchSync.Do(string(key), func() (any, error) {
		// there's always a chance a different thread completed a fetch before we got here
		// so check again now that we have a lock for the key
		item := c.cache.Get(key)
		if item != nil {
			return item, nil
		}

		return c.fetchAndSet(ctx, key)
	})

	if err != nil {
		return nil, err
	}
	return ii.(*ttlcache.Item[K, V]), nil
}

func (c *Local[K, V]) fetchAndSet(ctx context.Context, key K) (*ttlcache.Item[K, V], error) {
	val, err := c.fetch(ctx, key)
	if err != nil {
		return nil, err
	}

	return c.cache.Set(key, val, ttlcache.DefaultTTL), nil
}
