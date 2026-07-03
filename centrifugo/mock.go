package centrifugo

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
)

// MockClient is a mock implementation of Client that records publications in memory.
type MockClient struct {
	mu           sync.Mutex
	publications []*Publication
	err          error
}

// NewMockClient creates a new mock client.
func NewMockClient() *MockClient {
	return &MockClient{}
}

// Publish records the given publications, or returns the configured error. Like the real client it marshals each
// publication's data at this point, so what's recorded is the JSON that would have been sent.
func (c *MockClient) Publish(ctx context.Context, pubs ...*Publication) error {
	if len(pubs) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.err != nil {
		return c.err
	}

	// marshal everything before recording anything - like the real client, a batch with a marshal error sends nothing
	recorded := make([]*Publication, len(pubs))
	for i, p := range pubs {
		data, err := json.Marshal(p.Data)
		if err != nil {
			return fmt.Errorf("error marshaling data for channel %s: %w", p.Channel, err)
		}
		recorded[i] = &Publication{Channel: p.Channel, Data: json.RawMessage(data)}
	}
	c.publications = append(c.publications, recorded...)
	return nil
}

// Info returns the configured error, if any.
func (c *MockClient) Info(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.err
}

// Publications returns all recorded publications across all channels, oldest first. Publications marshal to JSON so
// the entire recording can be asserted in one go, e.g. against a test fixture.
func (c *MockClient) Publications() []*Publication {
	c.mu.Lock()
	defer c.mu.Unlock()

	return slices.Clone(c.publications)
}

// SetError sets the error returned by subsequent calls.
func (c *MockClient) SetError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.err = err
}

// Clear removes all recorded publications.
func (c *MockClient) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.publications = nil
}

var _ Client = (*MockClient)(nil)
