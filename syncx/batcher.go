package syncx

import (
	"sync"
	"time"

	"golang.org/x/exp/constraints"
)

// Batcher allows values to be queued and processed in a background thread.
type Batcher[T any] struct {
	process  func(batch []T)
	maxItems int
	maxAge   time.Duration
	wg       *sync.WaitGroup
	buffer   chan T
	stop     chan bool
	batch    []T
}

// NewBatcher creates a new batcher.
func NewBatcher[T any](process func(batch []T), maxItems int, maxAge time.Duration, capacity int, wg *sync.WaitGroup) *Batcher[T] {
	return &Batcher[T]{
		process:  process,
		maxItems: maxItems,
		maxAge:   maxAge,
		wg:       wg,
		buffer:   make(chan T, capacity),
		stop:     make(chan bool),
	}
}

// Start starts this batcher's background processing, returning immediately.
func (b *Batcher[T]) Start() {
	b.wg.Add(1)

	go func() {
		defer b.wg.Done()

		for {
			select {
			case v := <-b.buffer:
				b.batch = append(b.batch, v)
				if len(b.batch) == b.maxItems {
					b.flush()
				}

			case <-time.After(b.maxAge):
				b.flush()

			case <-b.stop:
				for len(b.buffer) > 0 || len(b.batch) > 0 {
					buffSize := len(b.buffer)
					canRead := min(b.maxItems-len(b.batch), buffSize)

					for i := 0; i < canRead; i++ {
						v := <-b.buffer
						b.batch = append(b.batch, v)
					}

					b.flush()
				}
				return
			}
		}
	}()
}

// Queue queues the given value, potentially blocking. Returns the new free capacity.
func (b *Batcher[T]) Queue(value T) int {
	b.buffer <- value

	return cap(b.buffer) - len(b.buffer)
}

// Stop stops this batcher.
func (b *Batcher[T]) Stop() {
	close(b.stop)
}

// flushes whatever has been batched
func (b *Batcher[T]) flush() {
	if len(b.batch) > 0 {
		b.process(b.batch)
		b.batch = make([]T, 0, b.maxItems)
	}
}

// TODO delete when on go 1.21 and this is builtin
func min[T constraints.Ordered](x T, y T) T {
	if x < y {
		return x
	}
	return y
}
