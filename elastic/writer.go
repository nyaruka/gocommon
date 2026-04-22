package elastic

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/nyaruka/gocommon/syncx"
)

// Writer provides buffered writes to Elasticsearch using a batcher. If writes fail, they are added to the given
// spool for later processing.
type Writer struct {
	client  *elasticsearch.TypedClient
	batcher *syncx.Batcher[*Document]
	spool   *Spool

	wg sync.WaitGroup

	numWritten atomic.Int64 // number of documents that have been written
	numSpooled atomic.Int64 // number of documents that have been spooled
}

// NewWriter creates a new writer.
func NewWriter(client *elasticsearch.TypedClient, maxItems int, maxAge time.Duration, bufferSize int, spool *Spool) *Writer {
	w := &Writer{
		client: client,
		spool:  spool,
	}
	w.batcher = syncx.NewBatcher(w.flush, maxItems, maxAge, bufferSize)

	return w
}

// Start starts the writer's batch processing.
func (w *Writer) Start() {
	w.batcher.Start(&w.wg)
}

// Queue queues a document for writing and will block if the buffer is full.
// Returns the remaining free capacity (batch + buffer).
func (w *Writer) Queue(doc *Document) int {
	return w.batcher.Queue(doc)
}

// Stop stops the writer and flushes any remaining items.
func (w *Writer) Stop() {
	w.batcher.Stop()
	w.wg.Wait()
}

// Flush forces a flush of current queue. Should only be used in tests.
func (w *Writer) Flush() {
	w.batcher.Flush()
}

// Client returns the Elasticsearch client this writer is using.
func (w *Writer) Client() *elasticsearch.TypedClient {
	return w.client
}

// Stats returns the number of documents written and spooled.
func (w *Writer) Stats() (int64, int64) {
	return w.numWritten.Load(), w.numSpooled.Load()
}

func (w *Writer) flush(batch []*Document) {
	ctx := context.TODO()

	numWritten, unprocessed, err := Bulk(ctx, w.client, batch)
	if err != nil {
		slog.Error("error writing batch to elasticsearch", "count", len(batch), "error", err)
		if unprocessed == nil {
			unprocessed = batch
		}
	}

	w.numWritten.Add(int64(numWritten))

	if len(unprocessed) > 0 {
		if err := w.spool.Add(unprocessed); err != nil {
			slog.Error("error writing unprocessed items to spool", "count", len(unprocessed), "error", err)
		}

		w.numSpooled.Add(int64(len(unprocessed)))
	}
}
