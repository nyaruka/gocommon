// Package centrifugo provides a client for the Centrifugo server API, a mock implementation for testing, and a
// service that layers channel subscriber tracking on top of a client.
package centrifugo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/centrifugal/gocent/v3"
)

// Publication is a single publish of data to a channel. Data can be pre-marshaled JSON (json.RawMessage or []byte)
// or any JSON marshal-able value - the latter is only marshaled when the publication is actually sent, so callers
// can defer the marshaling cost of publications that end up dropped (see Service.Publish).
type Publication struct {
	Channel string `json:"channel"`
	Data    any    `json:"data"`
}

// marshaledData returns the data as JSON, marshaling it if it isn't already marshaled.
func (p *Publication) marshaledData() (json.RawMessage, error) {
	switch typed := p.Data.(type) {
	case json.RawMessage:
		return typed, nil
	case []byte:
		return typed, nil
	default:
		return json.Marshal(p.Data)
	}
}

// Client is the interface for Centrifugo API clients, real and mock.
type Client interface {
	// Publish sends the given publishes to the server as a single pipelined request. If an individual publish is
	// rejected, the returned error identifies its channel.
	Publish(ctx context.Context, pubs ...*Publication) error

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

func (c *client) Publish(ctx context.Context, pubs ...*Publication) error {
	if len(pubs) == 0 {
		return nil
	}

	pipe := c.gc.Pipe()
	for _, p := range pubs {
		data, err := p.marshaledData()
		if err != nil {
			return fmt.Errorf("error marshaling data for channel %s: %w", p.Channel, err)
		}
		if err := pipe.AddPublish(p.Channel, data); err != nil {
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
