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
	assert.ErrorIs(t, err, httpx.ErrResponseSize)
}
