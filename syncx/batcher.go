package syncx

import (
	"context"
	"sync"
	"time"
)

// Batcher allows values to be queued and processed in a background thread.
type Batcher[T any] struct {
	process  func(batch []T)
	maxItems int
	maxAge   time.Duration

	buffer  chan T
	force   chan *sync.WaitGroup
	batch   []T
	timeout <-chan time.Time

	ctx    context.Context
	cancel context.CancelFunc
}

// NewBatcher creates a new batcher. Queued items are passed to the `process` callback in batches of `maxItems` maximum
// size. Processing of a batch is triggered by reaching `maxItems` or `maxAge` since the oldest unprocessed item was queued.
func NewBatcher[T any](process func(batch []T), maxItems int, maxAge time.Duration, bufferSize int) *Batcher[T] {
	ctx, cancel := context.WithCancel(context.Background())

	return &Batcher[T]{
		process:  process,
		maxItems: maxItems,
		maxAge:   maxAge,
		buffer:   make(chan T, bufferSize),
		batch:    make([]T, 0, maxItems),
		timeout:  nil,
		force:    make(chan *sync.WaitGroup, 1),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts this batcher's background processing, returning immediately.
func (b *Batcher[T]) Start(wg *sync.WaitGroup) {
	wg.Add(1)

	go func() {
		defer wg.Done()

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

			case fwg := <-b.force:
				// drain everything (batch + buffer) and signal completion
				b.drain()
				fwg.Done()

			case <-b.timeout:
				// flush whatever we have
				b.flush()

			case <-b.ctx.Done():
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
	b.cancel()
}

// Flush forces a flush of the current batch.
func (b *Batcher[T]) Flush() {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	b.force <- wg
	wg.Wait()
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

		for range canRead {
			v := <-b.buffer
			b.batch = append(b.batch, v)
		}

		b.flush()
	}
}
