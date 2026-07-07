package elastic

import (
	"context"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/nyaruka/gocommon/spool"
)

// Spool writes Elasticsearch documents to local files and periodically retries indexing them.
//
// Flushing is at-least-once so documents may be re-indexed after a crash - see [spool.Spool].
type Spool = spool.Spool[*Document]

// NewSpool creates a new spool using the given directory and flush interval.
func NewSpool(client *elasticsearch.TypedClient, directory string, flushInterval time.Duration) *Spool {
	return spool.New(directory, flushInterval, spool.MarshalJSON[*Document], spool.UnmarshalJSON[*Document],
		func(ctx context.Context, batch []*Document) ([]*Document, error) {
			_, unprocessed, err := Bulk(ctx, client, batch)
			return unprocessed, err
		},
	)
}
