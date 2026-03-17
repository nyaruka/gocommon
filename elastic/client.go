package elastic

import (
	"context"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
)

// NewClient creates a new Elasticsearch client.
func NewClient(url, username, password string) (*elasticsearch.TypedClient, error) {
	return elasticsearch.NewTypedClient(elasticsearch.Config{
		Addresses: []string{url},
		Username:  username,
		Password:  password,
	})
}

// Test checks that the given indexes exist.
func Test(ctx context.Context, c *elasticsearch.TypedClient, indexes ...string) error {
	for _, index := range indexes {
		exists, err := c.Indices.Exists(index).IsSuccess(ctx)
		if err != nil {
			return fmt.Errorf("error checking elasticsearch index %s: %w", index, err)
		}
		if !exists {
			return fmt.Errorf("elasticsearch index %s does not exist", index)
		}
	}

	return nil
}
