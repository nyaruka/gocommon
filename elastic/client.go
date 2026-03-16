package elastic

import (
	"context"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
)

// NewClient creates a new Elasticsearch client.
func NewClient(url string) (*elasticsearch.Client, error) {
	return elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{url},
	})
}

// Test checks that the given indexes exist.
func Test(ctx context.Context, c *elasticsearch.Client, indexes ...string) error {
	for _, index := range indexes {
		resp, err := c.Indices.Exists([]string{index}, c.Indices.Exists.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("error checking elasticsearch index %s: %w", index, err)
		}
		resp.Body.Close()

		if resp.IsError() {
			return fmt.Errorf("elasticsearch index %s does not exist", index)
		}
	}

	return nil
}
