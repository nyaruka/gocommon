package cache

import (
	"context"
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v3"
)

// Cache is a generic in-memory cache.
type Cache[K comparable, V any] struct {
	cache    *ttlcache.Cache[K, V]
	fetch    FetchFunc[K, V]
	fetchers sync.Map
}

// FetchFunc is a function which can fetch an item which doesn't yet exist in the cache.
type FetchFunc[K comparable, V any] func(context.Context, K) (V, error)

// NewCache creates a new cache.
func NewCache[K comparable, V any](fetch FetchFunc[K, V], ttl time.Duration) *Cache[K, V] {
	return &Cache[K, V]{
		cache: ttlcache.New[K, V](
			ttlcache.WithTTL[K, V](ttl),
			ttlcache.WithDisableTouchOnHit[K, V](),
		),
		fetch: fetch,
	}
}

// Start starts the routine to eliminate expired items from the cache.
func (c *Cache[K, V]) Start() {
	go c.cache.Start()
}

// Stop stops that routine.
func (c *Cache[K, V]) Stop() {
	c.cache.Stop()
}

// Len returns the number of items in the cache.
func (c *Cache[K, V]) Len() int {
	return c.cache.Len()
}

func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, error) {
	item := c.cache.Get(key)

	if item == nil {
		var err error

		item, err = c.fetchAndSet(ctx, key)
		if err != nil {
			var zero V
			return zero, err
		}
	}

	return item.Value(), nil
}

type fetcher[K comparable, V any] struct {
	item *ttlcache.Item[K, V]
	err  error
	done chan struct{}
}

func (c *Cache[K, V]) fetchAndSet(ctx context.Context, key K) (*ttlcache.Item[K, V], error) {
	// try to set the fetcher for this key
	actual, alreadyExists := c.fetchers.LoadOrStore(key, &fetcher[K, V]{done: make(chan struct{})})
	fetcher := actual.(*fetcher[K, V])

	if alreadyExists {
		// wait for other fetcher routine to do the fetch
		<-fetcher.done
	} else {
		defer func() {
			c.fetchers.Delete(key)
			close(fetcher.done)
		}()

		// there's always a chance a different thread completed a fetch before we got here
		// so check again now that we have a lock for the key
		if item := c.cache.Get(key); item != nil {
			fetcher.item, fetcher.err = item, nil
		} else {
			val, err := c.fetch(ctx, key)
			if err != nil {
				fetcher.err = err
			} else {
				fetcher.item = c.cache.Set(key, val, ttlcache.DefaultTTL)
			}
		}
	}
	return fetcher.item, fetcher.err
}
