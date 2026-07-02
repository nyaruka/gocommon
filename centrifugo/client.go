// Package centrifugo provides a client for the Centrifugo server API, and a mock implementation for testing.
package centrifugo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/centrifugal/gocent/v3"
)

// Publish is a single publish of data to a channel.
type Publish struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}

// Client is the interface for Centrifugo API clients, real and mock.
type Client interface {
	// Publish sends the given publishes to the server as a single pipelined request. If an individual publish is
	// rejected, the returned error identifies its channel.
	Publish(ctx context.Context, pubs ...*Publish) error

	// Info checks that the server is reachable and accepts our API key.
	Info(ctx context.Context) error
}

type client struct {
	gc *gocent.Client
}

// NewClient creates a new client for the Centrifugo API at the given endpoint.
func NewClient(endpoint, key string) Client {
	return &client{gc: gocent.New(gocent.Config{Addr: endpoint, Key: key})}
}

func (c *client) Publish(ctx context.Context, pubs ...*Publish) error {
	if len(pubs) == 0 {
		return nil
	}

	pipe := c.gc.Pipe()
	for _, p := range pubs {
		if err := pipe.AddPublish(p.Channel, p.Data); err != nil {
			return fmt.Errorf("error adding publish for channel %s: %w", p.Channel, err)
		}
	}

	// replies are index-parallel to pubs because we add exactly one command per publish
	replies, err := c.gc.SendPipe(ctx, pipe)
	if err != nil {
		return fmt.Errorf("error sending publishes: %w", err)
	}
	for i, reply := range replies {
		if reply.Error != nil {
			return fmt.Errorf("error publishing to channel %s: %w", pubs[i].Channel, reply.Error)
		}
	}
	return nil
}

func (c *client) Info(ctx context.Context) error {
	_, err := c.gc.Info(ctx)
	return err
}
