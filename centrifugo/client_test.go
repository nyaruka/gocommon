package centrifugo_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nyaruka/gocommon/centrifugo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type command struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// a fake Centrifugo API server: the client sends newline-delimited command JSON and expects one newline-delimited
// reply per command - success replies unless error replies have been queued for the next request
type testServer struct {
	*httptest.Server

	auths       []string    // authorization header of each request
	requests    [][]command // commands of each request
	nextReplies []string    // if set, the replies to the next request, instead of one {"result":{}} per command
}

func newTestServer() *testServer {
	s := &testServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.auths = append(s.auths, r.Header.Get("Authorization"))

		var cmds []command
		dec := json.NewDecoder(r.Body)
		for {
			var cmd command
			if err := dec.Decode(&cmd); err == io.EOF {
				break
			} else if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			cmds = append(cmds, cmd)
		}
		s.requests = append(s.requests, cmds)

		replies := s.nextReplies
		s.nextReplies = nil
		if replies == nil {
			for range cmds {
				replies = append(replies, `{"result":{}}`)
			}
		}
		for _, rep := range replies {
			io.WriteString(w, rep+"\n")
		}
	}))
	return s
}

func TestClientPublish(t *testing.T) {
	ctx := t.Context()

	srv := newTestServer()
	defer srv.Close()

	c := centrifugo.NewClient(srv.URL, "sesame")

	// zero publishes is a no-op that doesn't touch the server
	require.NoError(t, c.Publish(ctx))
	assert.Len(t, srv.requests, 0)

	// a batch of publishes is sent as a single request with one command per publish, in order
	err := c.Publish(ctx,
		&centrifugo.Publication{Channel: "chat:general", Data: []byte(`{"text":"hi"}`)},
		&centrifugo.Publication{Channel: "chat:random", Data: []byte(`{"text":"yo"}`)},
		&centrifugo.Publication{Channel: "chat:general", Data: []byte(`{"text":"bye"}`)},
	)
	require.NoError(t, err)

	require.Len(t, srv.requests, 1)
	assert.Equal(t, "apikey sesame", srv.auths[0])

	cmds := srv.requests[0]
	require.Len(t, cmds, 3)
	for _, cmd := range cmds {
		assert.Equal(t, "publish", cmd.Method)
	}
	assert.JSONEq(t, `{"channel":"chat:general","data":{"text":"hi"}}`, string(cmds[0].Params))
	assert.JSONEq(t, `{"channel":"chat:random","data":{"text":"yo"}}`, string(cmds[1].Params))
	assert.JSONEq(t, `{"channel":"chat:general","data":{"text":"bye"}}`, string(cmds[2].Params))

	// an error reply is attributed to the channel of the publish it corresponds to
	srv.nextReplies = []string{`{"result":{}}`, `{"error":{"code":102,"message":"unknown channel"}}`}
	err = c.Publish(ctx,
		&centrifugo.Publication{Channel: "chat:general", Data: []byte(`{"text":"hi"}`)},
		&centrifugo.Publication{Channel: "chat:nope", Data: []byte(`{"text":"yo"}`)},
	)
	assert.EqualError(t, err, "error publishing to channel chat:nope: unknown channel: 102")

	// a non-200 response is an error
	srv.Close()
	err = c.Publish(ctx, &centrifugo.Publication{Channel: "chat:general", Data: []byte(`{}`)})
	assert.ErrorContains(t, err, "error sending publishes")
}

func TestClientInfo(t *testing.T) {
	ctx := t.Context()

	srv := newTestServer()
	defer srv.Close()

	c := centrifugo.NewClient(srv.URL, "sesame")

	srv.nextReplies = []string{`{"result":{"nodes":[]}}`}
	require.NoError(t, c.Info(ctx))

	require.Len(t, srv.requests, 1)
	assert.Equal(t, "apikey sesame", srv.auths[0])
	require.Len(t, srv.requests[0], 1)
	assert.Equal(t, "info", srv.requests[0][0].Method)

	srv.nextReplies = []string{`{"error":{"code":401,"message":"unauthorized"}}`}
	assert.Error(t, c.Info(ctx))

	srv.Close()
	assert.Error(t, c.Info(ctx))
}
