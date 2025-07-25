package dynamo

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nyaruka/gocommon/syncx"
)

// Writer provides buffered writes to a DynamoDB table using a batcher
type Writer[K, I any] struct {
	ctx     context.Context
	table   *Table[K, I]
	batcher *syncx.Batcher[*I]
	spool   *Spool[I]
	wg      *sync.WaitGroup
}

// NewWriter creates a new writer that buffers writes to the given table.
// maxAge is the maximum time to wait before flushing a partial batch.
// bufferSize is the size of the internal buffer for queued items.
func NewWriter[K, I any](ctx context.Context, table *Table[K, I], maxAge time.Duration, bufferSize int, spool *Spool[I], wg *sync.WaitGroup) *Writer[K, I] {
	w := &Writer[K, I]{
		table: table,
		ctx:   ctx,
		spool: spool,
		wg:    wg,
	}
	w.batcher = syncx.NewBatcher(w.writeBatch, 25, maxAge, bufferSize, wg)

	return w
}

// Start starts the writer's background processing
func (w *Writer[K, I]) Start() error {
	if err := w.spool.Start(); err != nil {
		return fmt.Errorf("error starting spool: %w", err)
	}

	w.batcher.Start()

	return nil
}

// Write queues an item for writing and will block if the buffer is full.
// Returns the remaining free capacity (batch + buffer).
func (w *Writer[K, I]) Write(item *I) int {
	return w.batcher.Queue(item)
}

// Stop stops the writer and flushes any remaining items
func (w *Writer[K, I]) Stop() {
	w.batcher.Stop()
	w.spool.Stop()
}

func (w *Writer[K, I]) writeBatch(batch []*I) {
	unprocessed, err := w.table.BatchWriteItem(w.ctx, batch)
	if err != nil {
		slog.Error("error writing batch to dynamo", "count", len(batch), "error", err)
		if unprocessed == nil {
			unprocessed = batch
		}
	}

	if len(unprocessed) > 0 {
		if err := w.spool.Write(unprocessed); err != nil {
			slog.Error("error writing unprocessed items to spool", "count", len(unprocessed), "error", err)
		}
	}
}
