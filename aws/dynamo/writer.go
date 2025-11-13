package dynamo

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/nyaruka/gocommon/syncx"
)

// Writer provides buffered writes to a DynamoDB table using a batcher. If writes fail, they are added to the given
// spool for later processing.
type Writer struct {
	client  *dynamodb.Client
	table   string
	batcher *syncx.Batcher[map[string]types.AttributeValue]
	spool   *Spool
	keyFn   func(map[string]types.AttributeValue) string

	wg sync.WaitGroup

	numWritten atomic.Int64 // number of items that have been written
	numSpooled atomic.Int64 // number of items that have been spooled
}

// NewWriter creates a new writer.
func NewWriter(client *dynamodb.Client, table string, maxAge time.Duration, bufferSize int, spool *Spool, keyFn func(map[string]types.AttributeValue) string) *Writer {
	w := &Writer{
		client: client,
		table:  table,
		spool:  spool,
		keyFn:  keyFn,
	}
	w.batcher = syncx.NewBatcher(w.flush, 25, maxAge, bufferSize)

	return w
}

// Start starts the writer's batch processing.
func (w *Writer) Start() {
	w.batcher.Start(&w.wg)
}

// Queue queues an item for writing and will block if the buffer is full.
// Returns the remaining free capacity (batch + buffer).
func (w *Writer) Queue(item any) (int, error) {
	marshaled, err := Marshal(item)
	if err != nil {
		return 0, fmt.Errorf("error marshaling item: %w", err)
	}

	return w.batcher.Queue(marshaled), nil
}

// Stop stops the writer and flushes any remaining items
func (w *Writer) Stop() {
	w.batcher.Stop()
	w.wg.Wait()
}

// Flush forces a flush of current queue. Should only be used in tests.
func (w *Writer) Flush() {
	w.batcher.Flush()
}

// Table returns the table name this writer is writing to.
func (w *Writer) Table() string {
	return w.table
}

// Stats returns the number of items written and spooled.
func (w *Writer) Stats() (int64, int64) {
	return w.numWritten.Load(), w.numSpooled.Load()
}

func (w *Writer) flush(batch []map[string]types.AttributeValue) {
	ctx := context.TODO()

	if w.keyFn != nil {
		batch = dedupe(batch, w.keyFn)
	}

	unprocessed, err := batchPutItem(ctx, w.client, w.table, batch)
	if err != nil {
		slog.Error("error writing batch to dynamo", "count", len(batch), "error", err)
		if unprocessed == nil {
			unprocessed = batch
		}
	}

	w.numWritten.Add(int64(len(batch) - len(unprocessed)))

	if len(unprocessed) > 0 {
		if err := w.spool.Add(w.table, unprocessed); err != nil {
			slog.Error("error writing unprocessed items to spool", "count", len(unprocessed), "error", err)
		}

		w.numSpooled.Add(int64(len(unprocessed)))
	}
}

func dedupe(batch []map[string]types.AttributeValue, keyFn func(map[string]types.AttributeValue) string) []map[string]types.AttributeValue {
	if len(batch) <= 1 { // duplicates not possible
		return batch
	}

	seen := make(map[string]bool, len(batch))
	out := make([]map[string]types.AttributeValue, 0, len(batch))

	// iterate from end to start so we prefer the latest item for a given key
	for i := len(batch) - 1; i >= 0; i-- {
		item := batch[i]
		key := keyFn(item)

		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = true
		out = append(out, item)
	}

	// out is in reverse order (latest first); restore original ordering of the kept items
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}

	return out
}
