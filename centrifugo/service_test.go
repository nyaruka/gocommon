package centrifugo_test

import (
	"testing"

	valkey "github.com/gomodule/redigo/redis"
	"github.com/nyaruka/gocommon/centrifugo"
	"github.com/nyaruka/vkutil/assertvk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// creates a pool to the standard valkey test database, flushed before each test
func testValkeyPool(t *testing.T) *valkey.Pool {
	vk := assertvk.TestDB()
	t.Cleanup(func() { vk.Close() })

	assertvk.FlushDB()

	return vk
}

func setSubscribed(t *testing.T, vk *valkey.Pool, channel string) {
	vc := vk.Get()
	defer vc.Close()
	_, err := vc.Do("SET", centrifugo.SubscriptionKey(channel), "1", "EX", 150)
	require.NoError(t, err)
}

func TestServiceSubscribed(t *testing.T) {
	ctx := t.Context()

	vk := testValkeyPool(t)
	svc := centrifugo.NewService(centrifugo.NewMockClient(), vk)

	// zero channels is a no-op
	subs, err := svc.Subscribed(ctx)
	assert.NoError(t, err)
	assert.Empty(t, subs)

	// no presence keys set means nothing is subscribed
	subs, err = svc.Subscribed(ctx, "chat:1", "chat:2")
	assert.NoError(t, err)
	assert.Empty(t, subs)

	setSubscribed(t, vk, "chat:1")
	setSubscribed(t, vk, "chat:3")

	// only channels with presence keys come back, and all are resolved in one lookup
	subs, err = svc.Subscribed(ctx, "chat:1", "chat:2", "chat:3")
	assert.NoError(t, err)
	assert.Equal(t, map[string]bool{"chat:1": true, "chat:3": true}, subs)
}

func TestServicePublish(t *testing.T) {
	ctx := t.Context()

	vk := testValkeyPool(t)
	mock := centrifugo.NewMockClient()
	svc := centrifugo.NewService(mock, vk)

	// zero publishes is a no-op that doesn't touch valkey or the server
	require.NoError(t, svc.Publish(ctx))
	assert.Equal(t, 0, mock.Requests())

	setSubscribed(t, vk, "chat:1")

	// only publishes to subscribed channels are sent, as a single request
	err := svc.Publish(ctx,
		&centrifugo.Publish{Channel: "chat:1", Data: []byte(`{"text":"hi"}`)},
		&centrifugo.Publish{Channel: "chat:2", Data: []byte(`{"text":"yo"}`)},
		&centrifugo.Publish{Channel: "chat:1", Data: []byte(`{"text":"bye"}`)},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, mock.Requests())
	assert.Len(t, mock.Published("chat:1"), 2)
	assert.Len(t, mock.Published("chat:2"), 0)

	// a batch with no subscribed channels doesn't touch the server at all
	mock.Clear()
	err = svc.Publish(ctx, &centrifugo.Publish{Channel: "chat:2", Data: []byte(`{}`)})
	require.NoError(t, err)
	assert.Equal(t, 0, mock.Requests())

	// client errors are returned
	mock.SetError(assert.AnError)
	err = svc.Publish(ctx, &centrifugo.Publish{Channel: "chat:1", Data: []byte(`{}`)})
	assert.ErrorIs(t, err, assert.AnError)
}
