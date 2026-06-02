package httpx_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/nyaruka/gocommon/httpx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBodyLimitTransport(t *testing.T) {
	ctx := context.Background()

	server := newTestHTTPServer(52028)
	defer server.Close()

	// ?cmd=success returns a 16-byte body
	tcs := []struct {
		limit     int
		expectErr bool
	}{
		{limit: -1, expectErr: false}, // negative limit means no limit
		{limit: 0, expectErr: false},  // zero limit means no limit
		{limit: 15, expectErr: true},  // one byte short of the body size
		{limit: 16, expectErr: false}, // exactly the body size is allowed
		{limit: 20, expectErr: false}, // comfortably larger than the body
	}

	for _, tc := range tcs {
		tt := httpx.WithBodyLimit(http.DefaultTransport, tc.limit)
		request, err := httpx.NewRequest(ctx, "GET", server.URL+"?cmd=success", nil, nil)
		require.NoError(t, err)

		// the limit only surfaces when the body is read, not from RoundTrip itself
		resp, err := tt.RoundTrip(request)
		require.NoError(t, err, "limit=%d", tc.limit)

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if tc.expectErr {
			assert.ErrorIs(t, err, httpx.ErrResponseSize, "limit=%d", tc.limit)
			// the within-limit bytes are still returned alongside the error
			assert.Equal(t, `{ "ok": "true" }`[:tc.limit], string(body), "limit=%d", tc.limit)
		} else {
			assert.NoError(t, err, "limit=%d", tc.limit)
			assert.Equal(t, `{ "ok": "true" }`, string(body), "limit=%d", tc.limit)
		}
	}

	// a nil inner transport falls back to http.DefaultTransport
	tt := httpx.WithBodyLimit(nil, 1024)
	require.NotNil(t, tt)
	request, err := httpx.NewRequest(ctx, "GET", server.URL+"?cmd=success", nil, nil)
	require.NoError(t, err)
	resp, err := tt.RoundTrip(request)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, `{ "ok": "true" }`, string(body))

	// composed inside WithTracing, the limit bounds the body before it's buffered, so the caller reading the
	// handed-back body sees ErrResponseSize rather than WithTracing silently buffering the whole thing
	tracing := httpx.WithTracing(httpx.WithBodyLimit(http.DefaultTransport, 4), -1)
	request, err = httpx.NewRequest(ctx, "GET", server.URL+"?cmd=success", nil, nil)
	require.NoError(t, err)
	resp, err = tracing.RoundTrip(request)
	require.NoError(t, err)
	_, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.ErrorIs(t, err, httpx.ErrResponseSize)

	// an error from the inner transport is passed through unchanged, with no response to wrap
	inner := httpx.WithMocking(http.DefaultTransport, map[string][]*httpx.MockResponse{
		"https://temba.io": {httpx.MockConnectionError},
	})
	tt = httpx.WithBodyLimit(inner, 10)
	request, err = httpx.NewRequest(ctx, "GET", "https://temba.io", nil, nil)
	require.NoError(t, err)
	resp, err = tt.RoundTrip(request)
	assert.EqualError(t, err, "unable to connect to server")
	assert.Nil(t, resp)

	// closing the wrapped body closes the inner body
	spy := &scriptedBody{data: []byte("hello")}
	tt = httpx.WithBodyLimit(&fixedBodyTransport{body: spy}, 100)
	request, err = httpx.NewRequest(ctx, "GET", "https://temba.io", nil, nil)
	require.NoError(t, err)
	resp, err = tt.RoundTrip(request)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.True(t, spy.closed)

	// a body delivered in small chunks that straddles the limit yields exactly the within-limit bytes plus the size
	// error, exercising the multi-call accounting in limitedBody.Read
	chunked := &scriptedBody{data: []byte("0123456789abcdefghij"), chunk: 3}
	tt = httpx.WithBodyLimit(&fixedBodyTransport{body: chunked}, 8)
	request, err = httpx.NewRequest(ctx, "GET", "https://temba.io", nil, nil)
	require.NoError(t, err)
	resp, err = tt.RoundTrip(request)
	require.NoError(t, err)
	body, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.ErrorIs(t, err, httpx.ErrResponseSize)
	assert.Equal(t, "01234567", string(body))
}

// scriptedBody is a test io.ReadCloser that yields its data in fixed-size chunks (chunk <= 0 means all at once) and
// records whether it was closed.
type scriptedBody struct {
	data   []byte
	pos    int
	chunk  int
	closed bool
}

func (b *scriptedBody) Read(p []byte) (int, error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	end := len(b.data)
	if b.chunk > 0 && b.pos+b.chunk < end {
		end = b.pos + b.chunk
	}
	n := copy(p, b.data[b.pos:end])
	b.pos += n
	return n, nil
}

func (b *scriptedBody) Close() error { b.closed = true; return nil }

// fixedBodyTransport is a test http.RoundTripper that returns a 200 response with a preset body.
type fixedBodyTransport struct{ body io.ReadCloser }

func (t *fixedBodyTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: t.body}, nil
}
