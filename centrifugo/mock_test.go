package centrifugo_test

import (
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
		&centrifugo.Publication{Channel: "chat:general", Data: []byte(`{"text":"hi"}`)},
		&centrifugo.Publication{Channel: "chat:random", Data: []byte(`{"text":"yo"}`)},
	))
	require.NoError(t, mock.Publish(ctx, &centrifugo.Publication{Channel: "chat:general", Data: []byte(`{"text":"bye"}`)}))

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
	assert.Len(t, mock.Publications(), 3)
	mock.SetError(nil)

	// clearing removes recorded publications
	mock.Clear()
	assert.Empty(t, mock.Publications())
}
