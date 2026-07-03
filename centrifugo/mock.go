package centrifugo

import (
	"context"
	"encoding/json"
	"slices"
	"sync"
)

// MockClient is a mock implementation of Client that records publications in memory.
type MockClient struct {
	mu           sync.Mutex
	publications []*Publication
	requests     int
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
	c.requests++
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

// Published returns the data payloads published to the given channel, oldest first.
func (c *MockClient) Published(channel string) []json.RawMessage {
	c.mu.Lock()
	defer c.mu.Unlock()

	var data []json.RawMessage
	for _, p := range c.publications {
		if p.Channel == channel {
			data = append(data, p.Data)
		}
	}
	return data
}

// Requests returns the number of publish requests made, i.e. server round-trips.
func (c *MockClient) Requests() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.requests
}

// SetError sets the error returned by subsequent calls.
func (c *MockClient) SetError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.err = err
}

// Clear removes all recorded publications and resets the request count.
func (c *MockClient) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.publications = nil
	c.requests = 0
}

var _ Client = (*MockClient)(nil)
