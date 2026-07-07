package elastic

import (
	"context"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/nyaruka/gocommon/spools"
)

// NewSpool creates a new spool of documents which couldn't be written to Elasticsearch and are periodically retried
// via [Bulk]. Flushing is at-least-once so documents may be re-indexed after a crash - see [spools.Spool].
func NewSpool(client *elasticsearch.TypedClient, directory string, flushInterval time.Duration) *spools.Spool[*Document] {
	return spools.New(directory, flushInterval, spools.MarshalJSON[*Document], spools.UnmarshalJSON[*Document],
		func(ctx context.Context, batch []*Document) ([]*Document, error) {
			_, unprocessed, err := Bulk(ctx, client, batch)
			return unprocessed, err
		},
	)
}
