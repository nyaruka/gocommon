package httpx_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/nyaruka/gocommon/dates"
	"github.com/nyaruka/gocommon/httpx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTracesTransport(t *testing.T) {
	ctx := context.Background()

	defer dates.SetNowFunc(time.Now)
	dates.SetNowFunc(dates.NewSequentialNow(time.Date(2019, 10, 7, 15, 21, 30, 123456789, time.UTC), time.Second))

	server := newTestHTTPServer(52026)
	defer server.Close()

	tt := httpx.WithTraces(http.DefaultTransport)

	request, err := httpx.NewRequest(ctx, "GET", server.URL+"?cmd=success", nil, nil)
	require.NoError(t, err)
	resp, err := tt.RoundTrip(request)
	require.NoError(t, err)

	// the caller can still read the full response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, `{ "ok": "true" }`, string(body))

	// and a complete trace was captured
	require.Len(t, tt.Traces(), 1)
	trace := tt.Traces()[0]
	assert.Equal(t, "GET /?cmd=success HTTP/1.1\r\nHost: 127.0.0.1:52026\r\nUser-Agent: Go-http-client/1.1\r\nAccept-Encoding: gzip\r\n\r\n", string(trace.RequestTrace))
	assert.Equal(t, "HTTP/1.1 200 OK\r\nContent-Length: 16\r\nContent-Type: text/plain; charset=utf-8\r\nDate: Wed, 11 Apr 2018 18:24:30 GMT\r\n\r\n", string(trace.ResponseTrace))
	assert.Equal(t, `{ "ok": "true" }`, string(trace.ResponseBody))
	assert.Equal(t, time.Date(2019, 10, 7, 15, 21, 30, 123456789, time.UTC), trace.StartTime)
	assert.Equal(t, time.Date(2019, 10, 7, 15, 21, 31, 123456789, time.UTC), trace.EndTime)
	assert.Equal(t, 0, trace.Retries)

	// a second request accumulates another trace
	request, err = httpx.NewRequest(ctx, "GET", server.URL+"?cmd=success", nil, nil)
	require.NoError(t, err)
	resp, err = tt.RoundTrip(request)
	require.NoError(t, err)
	io.ReadAll(resp.Body)
	assert.Len(t, tt.Traces(), 2)

	// the inner transport still sees the request body after DumpRequestOut has consumed and restored it
	capturer := &bodyCapturingTransport{}
	tt = httpx.WithTraces(capturer)
	request, err = httpx.NewRequest(ctx, "POST", "https://temba.io", bytes.NewReader([]byte("hello body")), nil)
	require.NoError(t, err)
	resp, err = tt.RoundTrip(request)
	require.NoError(t, err)
	assert.Equal(t, "hello body", string(capturer.body))

	// an error from the inner transport is captured in the trace and returned
	inner := httpx.WithMocks(http.DefaultTransport, map[string][]*httpx.MockResponse{
		"https://temba.io": {httpx.MockConnectionError},
	})
	tt = httpx.WithTraces(inner)
	request, err = httpx.NewRequest(ctx, "GET", "https://temba.io", nil, nil)
	require.NoError(t, err)
	resp, err = tt.RoundTrip(request)
	assert.EqualError(t, err, "unable to connect to server")
	assert.Nil(t, resp)
	require.Len(t, tt.Traces(), 1)
	assert.NotNil(t, tt.Traces()[0].RequestTrace)
	assert.Nil(t, tt.Traces()[0].Response)
	assert.Nil(t, tt.Traces()[0].ResponseBody)

	// a nil inner transport falls back to http.DefaultTransport
	tt = httpx.WithTraces(nil)
	assert.NotNil(t, tt)
	request, err = httpx.NewRequest(ctx, "GET", server.URL+"?cmd=success", nil, nil)
	require.NoError(t, err)
	resp, err = tt.RoundTrip(request)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

// bodyCapturingTransport is a test http.RoundTripper that records the request body it received
type bodyCapturingTransport struct{ body []byte }

func (c *bodyCapturingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.body, _ = io.ReadAll(r.Body)
	r.Body.Close()
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader([]byte("ok")))}, nil
}

func TestTracesTransportConcurrent(t *testing.T) {
	server := newTestHTTPServer(52027)
	defer server.Close()

	tt := httpx.WithTraces(http.DefaultTransport)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			request, _ := http.NewRequest("GET", server.URL+"?cmd=success", nil)
			resp, err := tt.RoundTrip(request)
			if assert.NoError(t, err) {
				io.ReadAll(resp.Body)
				resp.Body.Close()
			}
		}()
	}
	wg.Wait()

	assert.Len(t, tt.Traces(), 20)
}

func TestTraceSizes(t *testing.T) {
	ctx := context.Background()

	tt := httpx.WithTraces(httpx.WithMocks(http.DefaultTransport, map[string][]*httpx.MockResponse{
		"https://temba.io": {
			httpx.NewMockResponse(200, nil, []byte(`{"ok": true}`)),
		},
	}))

	request, err := httpx.NewRequest(ctx, "POST", "https://temba.io", bytes.NewReader([]byte(`{"foo": "bar"}`)), nil)
	require.NoError(t, err)

	resp, err := tt.RoundTrip(request)
	require.NoError(t, err)
	resp.Body.Close()

	trace := tt.Traces()[0]
	assert.Equal(t, len(trace.RequestTrace), trace.RequestSize())
	assert.Equal(t, len(trace.ResponseTrace)+12, trace.ResponseSize())

	// if the body was discarded, the server declared Content-Length is used in its place
	trace.ResponseBody = nil
	assert.Equal(t, len(trace.ResponseTrace)+12, trace.ResponseSize())

	// unless there isn't one, e.g. a chunked response
	trace.Response.ContentLength = -1
	assert.Equal(t, len(trace.ResponseTrace), trace.ResponseSize())

	// a connection error leaves no response at all
	trace.Response = nil
	trace.ResponseTrace = nil
	assert.Equal(t, 0, trace.ResponseSize())
}

func TestNonUTF8Request(t *testing.T) {
	ctx := context.Background()

	tt := httpx.WithTraces(httpx.WithMocks(http.DefaultTransport, map[string][]*httpx.MockResponse{
		"https://temba.io": {
			&httpx.MockResponse{Status: 200, Headers: nil, Body: nil},
		},
	}))

	request, err := httpx.NewRequest(ctx, "GET", "https://temba.io", bytes.NewReader([]byte{'\xc3', '\x28'}), map[string]string{"X-Badness": string([]byte{'\xc3', '\x28'})})
	require.NoError(t, err)

	resp, err := tt.RoundTrip(request)
	require.NoError(t, err)
	resp.Body.Close()

	trace := tt.Traces()[0]
	assert.Equal(t, "GET / HTTP/1.1\r\nHost: temba.io\r\nUser-Agent: Go-http-client/1.1\r\nContent-Length: 2\r\nX-Badness: \xc3(\r\nAccept-Encoding: gzip\r\n\r\n\xc3(", string(trace.RequestTrace))
	assert.Equal(t, "HTTP/1.0 200 OK\r\nContent-Length: 0\r\n\r\n", string(trace.ResponseTrace))
	assert.False(t, utf8.Valid(trace.RequestTrace))
	assert.True(t, utf8.Valid(trace.ResponseTrace))
	assert.True(t, utf8.Valid(trace.ResponseBody))

	sanitized := trace.SanitizedRequest("...")
	assert.Equal(t, "GET / HTTP/1.1\r\nHost: temba.io\r\nUser-Agent: Go-http-client/1.1\r\nContent-Length: 2\r\nX-Badness: �(\r\nAccept-Encoding: gzip\r\n\r\n...", sanitized)
	assert.True(t, utf8.Valid([]byte(sanitized)))
}

func TestNonUTF8Response(t *testing.T) {
	ctx := context.Background()

	tt := httpx.WithTraces(httpx.WithMocks(http.DefaultTransport, map[string][]*httpx.MockResponse{
		"https://temba.io": {
			&httpx.MockResponse{Status: 200, Headers: map[string]string{"X-Badness": string([]byte{'\xc3', '\x28'})}, Body: []byte{'\xc3', '\x28'}},
		},
	}))

	request, err := httpx.NewRequest(ctx, "GET", "https://temba.io", nil, nil)
	require.NoError(t, err)

	resp, err := tt.RoundTrip(request)
	require.NoError(t, err)
	resp.Body.Close()

	trace := tt.Traces()[0]
	assert.Equal(t, "GET / HTTP/1.1\r\nHost: temba.io\r\nUser-Agent: Go-http-client/1.1\r\nAccept-Encoding: gzip\r\n\r\n", string(trace.RequestTrace))
	assert.Equal(t, "HTTP/1.0 200 OK\r\nContent-Length: 2\r\nX-Badness: \xc3(\r\n\r\n", string(trace.ResponseTrace))
	assert.Equal(t, []byte{'\xc3', '\x28'}, trace.ResponseBody)
	assert.False(t, utf8.Valid(trace.ResponseTrace))
	assert.False(t, utf8.Valid(trace.ResponseBody))

	sanitized := trace.SanitizedResponse("...")
	assert.Equal(t, "HTTP/1.0 200 OK\r\nContent-Length: 2\r\nX-Badness: �(\r\n\r\n...", sanitized)
	assert.True(t, utf8.Valid([]byte(sanitized)))
}

// roundTripFunc adapts a function to an http.RoundTripper for tests.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTraceCollector(t *testing.T) {
	const url = "https://example.com/thing"

	// the tracing transport is installed once, on a shared client
	client := &http.Client{Transport: httpx.WithTraces(httpx.WithMocks(nil, map[string][]*httpx.MockResponse{
		url: {httpx.NewMockResponse(200, nil, []byte("hello")), httpx.NewMockResponse(200, nil, []byte("again"))},
	}))}

	// a call made with a collector in its context can pick out its own trace
	ctx, traces := httpx.WithTraceCollector(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	require.Len(t, traces.Traces(), 1)
	require.NotNil(t, traces.Last())
	assert.Equal(t, []byte("hello"), traces.Last().ResponseBody)

	// a second call gets its own collector, unpolluted by the first
	ctx2, traces2 := httpx.WithTraceCollector(context.Background())
	req2, _ := http.NewRequestWithContext(ctx2, "GET", url, nil)
	_, err = client.Do(req2)
	require.NoError(t, err)

	assert.Len(t, traces.Traces(), 1, "first collector should not see the second call")
	require.Len(t, traces2.Traces(), 1)
	assert.Equal(t, []byte("again"), traces2.Last().ResponseBody)

	// a redirect records every hop, and Last is the final one
	redirectClient := &http.Client{Transport: httpx.WithTraces(httpx.WithMocks(nil, map[string][]*httpx.MockResponse{
		"https://example.com/redirect": {httpx.NewMockResponse(302, map[string]string{"Location": url}, nil)},
		url:                            {httpx.NewMockResponse(200, nil, []byte("final"))},
	}))}
	ctx3, traces3 := httpx.WithTraceCollector(context.Background())
	req3, _ := http.NewRequestWithContext(ctx3, "GET", "https://example.com/redirect", nil)
	_, err = redirectClient.Do(req3)
	require.NoError(t, err)

	assert.Len(t, traces3.Traces(), 2)
	assert.Equal(t, 200, traces3.Last().Response.StatusCode)
	assert.Equal(t, []byte("final"), traces3.Last().ResponseBody)

	// requests made without a collector still work, and are still recorded by the transport itself
	tracer := httpx.WithTraces(httpx.WithMocks(nil, map[string][]*httpx.MockResponse{
		url: {httpx.NewMockResponse(200, nil, []byte("solo"))},
	}))
	req4, _ := http.NewRequest("GET", url, nil)
	_, err = (&http.Client{Transport: tracer}).Do(req4)
	require.NoError(t, err)
	assert.Len(t, tracer.Traces(), 1)

	// an empty collector reports no traces rather than panicking
	_, empty := httpx.WithTraceCollector(context.Background())
	assert.Empty(t, empty.Traces())
	assert.Nil(t, empty.Last())
}

func TestTraceCollectorConcurrent(t *testing.T) {
	client := &http.Client{Transport: httpx.WithTraces(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{Status: "200 OK", StatusCode: 200, Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1,
			Header: make(http.Header), Body: io.NopCloser(bytes.NewReader([]byte(r.URL.Path)))}, nil
	}))}

	// each concurrent caller must see exactly its own trace, not those of the calls racing alongside it
	wg := sync.WaitGroup{}
	for i := range 50 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			ctx, traces := httpx.WithTraceCollector(context.Background())
			path := fmt.Sprintf("/req-%d", i)
			req, _ := http.NewRequestWithContext(ctx, "GET", "https://example.com"+path, nil)
			_, err := client.Do(req)

			assert.NoError(t, err)
			if assert.Len(t, traces.Traces(), 1) {
				assert.Equal(t, []byte(path), traces.Last().ResponseBody)
			}
		}(i)
	}
	wg.Wait()
}

func TestTraceCollectorWithRetries(t *testing.T) {
	// a retrier composed inside the tracer reports its retries on the collected trace
	client := &http.Client{Transport: httpx.WithTraces(httpx.WithRetries(httpx.WithMocks(nil, map[string][]*httpx.MockResponse{
		"https://example.com/thing": {httpx.NewMockResponse(502, nil, nil), httpx.NewMockResponse(200, nil, []byte("ok"))},
	}), httpx.NewFixedRetries(time.Millisecond)))}

	ctx, traces := httpx.WithTraceCollector(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://example.com/thing", nil)
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	require.Len(t, traces.Traces(), 1)
	assert.Equal(t, 1, traces.Last().Retries)
	assert.Equal(t, []byte("ok"), traces.Last().ResponseBody)
}

func TestTraceCollectorReadLimit(t *testing.T) {
	// the read limit stays a transport concern, composed inside the tracer on the client
	client := &http.Client{Transport: httpx.WithTraces(httpx.WithReadLimit(httpx.WithMocks(nil, map[string][]*httpx.MockResponse{
		"https://example.com/big": {httpx.NewMockResponse(200, nil, bytes.Repeat([]byte("x"), 100))},
	}), 10))}

	ctx, traces := httpx.WithTraceCollector(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://example.com/big", nil)
	resp, err := client.Do(req)
	require.NoError(t, err) // the limit is enforced when the body is read, not by Do

	_, err = io.ReadAll(resp.Body)
	assert.ErrorIs(t, err, httpx.ErrResponseSize)
	require.NotNil(t, traces.Last())
}
