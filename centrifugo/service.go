package centrifugo

import (
	"context"
	"fmt"

	valkey "github.com/gomodule/redigo/redis"
)

// Service composes an API client with tracking of which channels currently have subscribers, so that publishes are
// only actually sent to the server for channels that someone is watching.
//
// Subscriber tracking is presence only - whether a channel has any subscribers, not who or how many. The service
// that authorizes channel subscriptions marks a channel as subscribed by setting a valkey key (see SubscriptionKey)
// with a TTL, re-arming it on every subscribe and refresh - there is no unsubscribe callback so expiry of that key
// is the only garbage collection. This service only ever reads those keys: a channel is subscribed if its key exists.
type Service struct {
	Client Client
	vk     *valkey.Pool
}

// NewService creates a new service from the given API client and valkey pool.
func NewService(client Client, vk *valkey.Pool) *Service {
	return &Service{Client: client, vk: vk}
}

// SubscriptionKey returns the valkey key marking that the given channel has at least one active subscriber, e.g.
// "socket-subs:chat:1234". The key name and its presence-via-existence semantics are a contract with the service
// that authorizes subscriptions and writes these keys. Exported so tests can simulate subscribers.
func SubscriptionKey(channel string) string {
	return fmt.Sprintf("socket-subs:%s", channel)
}

// Subscribed returns the subset of the given channels which currently have at least one active subscriber. All
// channels are resolved in a single round-trip by MGETting their presence keys, so checking many channels at once
// costs one lookup rather than one per channel. The returned map only contains the subscribed channels.
func (s *Service) Subscribed(ctx context.Context, channels ...string) (map[string]bool, error) {
	if len(channels) == 0 {
		return nil, nil
	}

	keys := make([]any, len(channels))
	for i, ch := range channels {
		keys[i] = SubscriptionKey(ch)
	}

	vc := s.vk.Get()
	defer vc.Close()

	values, err := valkey.Values(valkey.DoContext(vc, ctx, "MGET", keys...))
	if err != nil {
		return nil, fmt.Errorf("error checking channel subscriptions: %w", err)
	}

	subscribed := make(map[string]bool, len(channels))
	for i, v := range values {
		if v != nil {
			subscribed[channels[i]] = true
		}
	}
	return subscribed, nil
}

// Publish sends the given publishes to the server, skipping any whose channel has no current subscribers - such
// publishes would be delivered to nobody so dropping them just saves the server the work. Subscriber presence for
// all the channels is resolved in a single lookup and the surviving publishes are sent as a single pipelined
// request, so a batch of any size costs at most two round-trips and lands or fails together.
func (s *Service) Publish(ctx context.Context, pubs ...*Publication) error {
	if len(pubs) == 0 {
		return nil
	}

	channels := make([]string, 0, len(pubs))
	seen := make(map[string]bool, len(pubs))
	for _, p := range pubs {
		if !seen[p.Channel] {
			seen[p.Channel] = true
			channels = append(channels, p.Channel)
		}
	}

	subscribed, err := s.Subscribed(ctx, channels...)
	if err != nil {
		return err
	}

	send := make([]*Publication, 0, len(pubs))
	for _, p := range pubs {
		if subscribed[p.Channel] {
			send = append(send, p)
		}
	}

	return s.Client.Publish(ctx, send...)
}
