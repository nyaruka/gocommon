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

	// zero publications is a no-op
	require.NoError(t, mock.Publish(ctx))
	assert.Empty(t, mock.Publications())

	require.NoError(t, mock.Publish(ctx,
		&centrifugo.Publication{Channel: "chat:general", Data: json.RawMessage(`{"text":"hi"}`)},
		&centrifugo.Publication{Channel: "chat:random", Data: json.RawMessage(`{"text":"yo"}`)},
	))
	require.NoError(t, mock.Publish(ctx, &centrifugo.Publication{Channel: "chat:general", Data: json.RawMessage(`{"text":"bye"}`)}))

	// unmarshaled data is marshaled when recorded, like the real client marshals when sending
	require.NoError(t, mock.Publish(ctx, &centrifugo.Publication{Channel: "chat:general", Data: map[string]any{"text": "hola"}}))
	assert.Equal(t, json.RawMessage(`{"text":"hola"}`), mock.Publications()[3].Data)

	// the entire recording can be asserted at once, e.g. as JSON against a fixture
	assert.JSONEq(t, `[
		{"channel": "chat:general", "data": {"text": "hi"}},
		{"channel": "chat:random", "data": {"text": "yo"}},
		{"channel": "chat:general", "data": {"text": "bye"}},
		{"channel": "chat:general", "data": {"text": "hola"}}
	]`, string(jsonx.MustMarshal(mock.Publications())))

	// unmarshalable data is an error identifying the channel, and nothing is recorded
	assert.ErrorContains(t, mock.Publish(ctx, &centrifugo.Publication{Channel: "chat:bad", Data: func() {}}), "error marshaling data for channel chat:bad")
	assert.Len(t, mock.Publications(), 4)

	// a configured error is returned by Publish and Info, and nothing is recorded
	mock.SetError(errors.New("boom"))
	assert.EqualError(t, mock.Publish(ctx, &centrifugo.Publication{Channel: "chat:general", Data: json.RawMessage(`{}`)}), "boom")
	assert.EqualError(t, mock.Info(ctx), "boom")
	assert.Len(t, mock.Publications(), 4)
	mock.SetError(nil)

	// clearing removes recorded publications
	mock.Clear()
	assert.Empty(t, mock.Publications())
}
