package centrifugo

import (
	"context"
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

// Publish records the given publications, or returns the configured error.
func (c *MockClient) Publish(ctx context.Context, pubs ...*Publication) error {
	if len(pubs) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.err != nil {
		return c.err
	}
	c.publications = append(c.publications, pubs...)
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
