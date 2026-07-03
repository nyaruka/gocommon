package centrifugo_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/nyaruka/gocommon/centrifugo"
	"github.com/nyaruka/gocommon/jsonx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockClient(t *testing.T) {
	ctx := t.Context()

	mock := centrifugo.NewMockClient()

	require.NoError(t, mock.Info(ctx))

	// zero publishes is a no-op that isn't counted as a request
	require.NoError(t, mock.Publish(ctx))
	assert.Equal(t, 0, mock.Requests())

	// each publish call is recorded as a single request, publishes readable per channel in order
	require.NoError(t, mock.Publish(ctx,
		&centrifugo.Publication{Channel: "chat:general", Data: []byte(`{"text":"hi"}`)},
		&centrifugo.Publication{Channel: "chat:random", Data: []byte(`{"text":"yo"}`)},
	))
	require.NoError(t, mock.Publish(ctx, &centrifugo.Publication{Channel: "chat:general", Data: []byte(`{"text":"bye"}`)}))

	assert.Equal(t, 2, mock.Requests())
	assert.Equal(t, []json.RawMessage{[]byte(`{"text":"hi"}`), []byte(`{"text":"bye"}`)}, mock.Published("chat:general"))
	assert.Equal(t, []json.RawMessage{[]byte(`{"text":"yo"}`)}, mock.Published("chat:random"))
	assert.Empty(t, mock.Published("chat:other"))

	// the entire recording can be asserted at once, e.g. as JSON against a fixture
	assert.JSONEq(t, `[
		{"channel": "chat:general", "data": {"text": "hi"}},
		{"channel": "chat:random", "data": {"text": "yo"}},
		{"channel": "chat:general", "data": {"text": "bye"}}
	]`, string(jsonx.MustMarshal(mock.Publications())))

	// a configured error is returned by Publish and Info, and nothing is recorded
	mock.SetError(errors.New("boom"))
	assert.EqualError(t, mock.Publish(ctx, &centrifugo.Publication{Channel: "chat:general", Data: []byte(`{}`)}), "boom")
	assert.EqualError(t, mock.Info(ctx), "boom")
	assert.Equal(t, 2, mock.Requests())
	assert.Len(t, mock.Published("chat:general"), 2)
	mock.SetError(nil)

	// clearing removes recorded publishes and resets the request count
	mock.Clear()
	assert.Equal(t, 0, mock.Requests())
	assert.Empty(t, mock.Published("chat:general"))
	assert.Empty(t, mock.Publications())
}
