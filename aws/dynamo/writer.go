package dynamo

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/nyaruka/gocommon/syncx"
)

// Writer provides buffered writes to a DynamoDB table using a batcher
type Writer struct {
	client  *dynamodb.Client
	table   string
	batcher *syncx.Batcher[map[string]types.AttributeValue]
	spool   *Spool
	wg      *sync.WaitGroup

	ctx context.Context
}

// NewWriter creates a new writer that buffers writes to the given table.
// maxAge is the maximum time to wait before flushing a partial batch.
// bufferSize is the size of the internal buffer for queued items.
func NewWriter(ctx context.Context, client *dynamodb.Client, table string, maxAge time.Duration, bufferSize int, spool *Spool, wg *sync.WaitGroup) *Writer {
	w := &Writer{
		client: client,
		table:  table,
		ctx:    ctx,
		spool:  spool,
		wg:     wg,
	}
	w.batcher = syncx.NewBatcher(w.flush, 25, maxAge, bufferSize, wg)

	return w
}

// Start starts the writer's background processing
func (w *Writer) Start() {
	w.batcher.Start()
}

// Write queues an item for writing and will block if the buffer is full.
// Returns the remaining free capacity (batch + buffer).
func (w *Writer) Write(item any) (int, error) {
	marshaled, err := Marshal(item)
	if err != nil {
		return 0, fmt.Errorf("error marshaling item: %w", err)
	}

	return w.batcher.Queue(marshaled), nil
}

// Stop stops the writer and flushes any remaining items
func (w *Writer) Stop() {
	w.batcher.Stop()
}

func (w *Writer) flush(batch []map[string]types.AttributeValue) {
	unprocessed, err := batchPutItem(w.ctx, w.client, w.table, batch)
	if err != nil {
		slog.Error("error writing batch to dynamo", "count", len(batch), "error", err)
		if unprocessed == nil {
			unprocessed = batch
		}
	}

	if len(unprocessed) > 0 {
		if err := w.spool.Add(w.table, unprocessed); err != nil {
			slog.Error("error writing unprocessed items to spool", "count", len(unprocessed), "error", err)
		}
	}
}
