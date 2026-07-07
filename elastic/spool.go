package elastic

import (
	"context"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/nyaruka/gocommon/spool"
)

// NewSpool creates a new spool of documents which couldn't be written to Elasticsearch and are periodically retried
// via [Bulk]. Flushing is at-least-once so documents may be re-indexed after a crash - see [spool.Spool].
func NewSpool(client *elasticsearch.TypedClient, directory string, flushInterval time.Duration) *spool.Spool[*Document] {
	return spool.New(directory, flushInterval, spool.MarshalJSON[*Document], spool.UnmarshalJSON[*Document],
		func(ctx context.Context, batch []*Document) ([]*Document, error) {
			_, unprocessed, err := Bulk(ctx, client, batch)
			return unprocessed, err
		},
	)
}
