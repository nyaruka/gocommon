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

	wg      *sync.WaitGroup
	buffer  chan T
	stop    chan bool
	batch   []T
	timeout <-chan time.Time
}

// NewBatcher creates a new batcher. Queued items are passed to the `process` callback in batches of `maxItems` maximum
// size. Processing of a batch is triggered by reaching `maxItems` or `maxAge` since the oldest unprocessed item was queued.
func NewBatcher[T any](process func(batch []T), maxItems int, maxAge time.Duration, bufferSize int, wg *sync.WaitGroup) *Batcher[T] {
	return &Batcher[T]{
		process:  process,
		maxItems: maxItems,
		maxAge:   maxAge,
		wg:       wg,
		buffer:   make(chan T, bufferSize),
		stop:     make(chan bool),
		batch:    make([]T, 0, maxItems),
		timeout:  nil,
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

				// if this is the first item in the batch we need to restart the age timeout
				if b.timeout == nil {
					b.timeout = time.After(b.maxAge)
				}

				// if we have a full batch, flush it
				if len(b.batch) == b.maxItems {
					b.flush()
				}

			case <-b.timeout:
				// flush whatever we have
				b.flush()

			case <-b.stop:
				b.drain()
				close(b.buffer)
				return
			}
		}
	}()
}

// Queue queues the given value, potentially blocking. Returns the new free capacity (batch + buffer).
func (b *Batcher[T]) Queue(value T) int {
	b.buffer <- value

	return (cap(b.batch) + cap(b.buffer)) - (len(b.batch) + len(b.buffer))
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
		b.timeout = nil
	}
}

// processes everything in the batch and buffer until they're both empty
func (b *Batcher[T]) drain() {
	for len(b.buffer) > 0 || len(b.batch) > 0 {
		buffSize := len(b.buffer)
		canRead := min(b.maxItems-len(b.batch), buffSize)

		for i := 0; i < canRead; i++ {
			v := <-b.buffer
			b.batch = append(b.batch, v)
		}

		b.flush()
	}
}

// TODO delete when on go 1.21 and this is builtin
func min[T constraints.Ordered](x T, y T) T {
	if x < y {
		return x
	}
	return y
}
