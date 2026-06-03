package httpx_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/dates"
	"github.com/nyaruka/gocommon/httpx"
	"github.com/nyaruka/gocommon/random"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExponentialRetries(t *testing.T) {
	defer random.SetGenerator(random.DefaultGenerator)

	retries := httpx.NewExponentialRetries(5*time.Second, 2, 0.5)

	assert.Equal(t, []time.Duration{5 * time.Second, 10 * time.Second}, retries.Backoffs)
	assert.Equal(t, float64(0.5), retries.Jitter)

	retries = httpx.NewExponentialRetries(2*time.Second, 4, 0.0)

	assert.Equal(t, []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}, retries.Backoffs)
	assert.Equal(t, float64(0.0), retries.Jitter)
	assert.Equal(t, 4, retries.MaxRetries())

	random.SetGenerator(random.NewSeededGenerator(123456))

	// test backoffs with no jitter
	assert.Equal(t, 2*time.Second, retries.Backoff(0))
	assert.Equal(t, 4*time.Second, retries.Backoff(1))
	assert.Equal(t, 8*time.Second, retries.Backoff(2))
	assert.Equal(t, 16*time.Second, retries.Backoff(3))
	assert.Panics(t, func() { retries.Backoff(4) })

	// test backoffs with 5% jitter
	retries.Jitter = 0.05

	assert.Equal(t, time.Duration(1964211898), retries.Backoff(0))
	assert.Equal(t, time.Duration(3970345144), retries.Backoff(1))
	assert.Equal(t, time.Duration(8142741864), retries.Backoff(2))
	assert.Equal(t, time.Duration(15884061444), retries.Backoff(3))

	// test backoffs with 100% jitter
	retries.Jitter = 1.0

	assert.Equal(t, time.Duration(1280781995), retries.Backoff(0))
	assert.Equal(t, time.Duration(5877181643), retries.Backoff(1))
	assert.Equal(t, time.Duration(8587700930), retries.Backoff(2))
	assert.Equal(t, time.Duration(9120513163), retries.Backoff(3))
}

func TestDoWithRetries(t *testing.T) {
	ctx := context.Background()

	defer httpx.SetRequestor(httpx.DefaultRequestor)

	mocks := httpx.NewMockRequestor(map[string][]*httpx.MockResponse{
		"http://temba.io/1/": {
			httpx.NewMockResponse(502, nil, []byte("a")),
		},
		"http://temba.io/2/": {
			httpx.NewMockResponse(503, nil, []byte("a")),
			httpx.NewMockResponse(504, nil, []byte("b")),
			httpx.NewMockResponse(505, nil, []byte("c")),
		},
		"http://temba.io/3/": {
			httpx.NewMockResponse(200, nil, []byte("a")),
		},
		"http://temba.io/4/": {
			httpx.NewMockResponse(502, nil, []byte("a")),
		},
		"http://temba.io/5/": {
			httpx.NewMockResponse(502, nil, []byte("a")),
			httpx.NewMockResponse(200, nil, []byte("b")),
		},
		"http://temba.io/6/": {
			httpx.NewMockResponse(429, map[string]string{"Retry-After": "1"}, []byte("a")),
			httpx.NewMockResponse(201, nil, []byte("b")),
		},
		"http://temba.io/7/": {
			httpx.NewMockResponse(429, map[string]string{"Retry-After": "100"}, []byte("a")),
		},
	})
	httpx.SetRequestor(mocks)

	call := func(method, url string, headers map[string]string, retries *httpx.RetryConfig) *httpx.Trace {
		request, err := httpx.NewRequest(ctx, method, url, nil, headers)
		require.NoError(t, err)

		trace, err := httpx.DoTrace(http.DefaultClient, request, retries, nil, -1)
		require.NoError(t, err)

		return trace
	}

	// no retry config
	trace := call("GET", "http://temba.io/1/", nil, nil)
	assert.Equal(t, 502, trace.Response.StatusCode)

	// a retry config which can make 2 retries
	retries := httpx.NewFixedRetries(1*time.Millisecond, 2*time.Millisecond)

	// retrying thats ends with failure
	trace = call("GET", "http://temba.io/2/", nil, retries)
	assert.Equal(t, 505, trace.Response.StatusCode)
	assert.Equal(t, 2, trace.Retries)

	// retrying not needed
	trace = call("GET", "http://temba.io/3/", nil, retries)
	assert.Equal(t, 200, trace.Response.StatusCode)
	assert.Equal(t, 0, trace.Retries)

	// retrying not used for POSTs
	trace = call("POST", "http://temba.io/4/", nil, retries)
	assert.Equal(t, 502, trace.Response.StatusCode)
	assert.Equal(t, 0, trace.Retries)

	// unless idempotency declared via request header
	trace = call("POST", "http://temba.io/5/", map[string]string{"Idempotency-Key": "123"}, retries)
	assert.Equal(t, 200, trace.Response.StatusCode)
	assert.Equal(t, 1, trace.Retries)

	// a retry config which can make 1 retry (need a longer delay so that the Retry-After header value can be used)
	retries = httpx.NewFixedRetries(1 * time.Second)

	// retrying due to Retry-After header
	trace = call("POST", "http://temba.io/6/", nil, retries)
	assert.Equal(t, 201, trace.Response.StatusCode)
	assert.Equal(t, 1, trace.Retries)

	// ignoring Retry-After header when it's too long
	trace = call("GET", "http://temba.io/7/", nil, retries)
	assert.Equal(t, 429, trace.Response.StatusCode)
	assert.Equal(t, 0, trace.Retries)

	assert.False(t, mocks.HasUnused())
}

func TestParseRetryAfter(t *testing.T) {
	defer dates.SetNowFunc(time.Now)

	dates.SetNowFunc(dates.NewFixedNow(time.Date(2020, 1, 7, 15, 10, 30, 500000000, time.UTC)))

	assert.Equal(t, 0*time.Second, httpx.ParseRetryAfter("x"))
	assert.Equal(t, 0*time.Second, httpx.ParseRetryAfter("0"))
	assert.Equal(t, 10*time.Second, httpx.ParseRetryAfter("10"))
	assert.Equal(t, 10*time.Second, httpx.ParseRetryAfter("10"))
	assert.Equal(t, 4500*time.Millisecond, httpx.ParseRetryAfter("Wed, 07 Jan 2020 15:10:35 GMT")) // 4.5 seconds in future
	assert.Equal(t, 0*time.Second, httpx.ParseRetryAfter("Wed, 07 Jan 2020 15:10:25 GMT"))         // 5.5 seconds in the past
}

// recordingTransport is a test http.RoundTripper which returns a programmed sequence of responses (a zero status
// simulating a connection error) and records the request body bytes each attempt received.
type recordingTransport struct {
	steps  []recordedStep
	bodies [][]byte
	calls  int
}

type recordedStep struct {
	status  int
	headers map[string]string
	body    string
}

func (rt *recordingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
		r.Body.Close()
	}
	rt.bodies = append(rt.bodies, body)

	step := rt.steps[rt.calls]
	rt.calls++

	if step.status == 0 {
		return nil, errors.New("unable to connect to server")
	}

	header := make(http.Header)
	for k, v := range step.headers {
		header.Set(k, v)
	}
	return &http.Response{
		StatusCode:    step.status,
		Status:        fmt.Sprintf("%d %s", step.status, http.StatusText(step.status)),
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        header,
		Body:          io.NopCloser(strings.NewReader(step.body)),
		ContentLength: int64(len(step.body)),
		Request:       r,
	}, nil
}

func TestWithRetries(t *testing.T) {
	ctx := context.Background()

	// tiny backoffs to keep the test fast; two retries allowed
	retries := httpx.NewFixedRetries(time.Millisecond, time.Millisecond)

	// a retryable status that eventually succeeds
	inner := &recordingTransport{steps: []recordedStep{
		{status: 503, body: "fail"},
		{status: 503, body: "fail"},
		{status: 200, body: "ok"},
	}}
	req, err := httpx.NewRequest(ctx, "GET", "http://temba.io/", nil, nil)
	require.NoError(t, err)
	resp, err := httpx.WithRetries(inner, retries).RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 3, inner.calls)

	// the final response body is still readable and un-drained
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "ok", string(body))

	// retries exhausted returns the last response (still readable)
	inner = &recordingTransport{steps: []recordedStep{
		{status: 503, body: "a"},
		{status: 503, body: "b"},
		{status: 503, body: "c"},
	}}
	req, _ = httpx.NewRequest(ctx, "GET", "http://temba.io/", nil, nil)
	resp, err = httpx.WithRetries(inner, retries).RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, 503, resp.StatusCode)
	assert.Equal(t, 3, inner.calls)
	body, _ = io.ReadAll(resp.Body)
	assert.Equal(t, "c", string(body))

	// no retry when the first response doesn't warrant it
	inner = &recordingTransport{steps: []recordedStep{{status: 200, body: "ok"}}}
	req, _ = httpx.NewRequest(ctx, "GET", "http://temba.io/", nil, nil)
	resp, err = httpx.WithRetries(inner, retries).RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 1, inner.calls)

	// no retry for a non-idempotent request (POST without an idempotency key)
	inner = &recordingTransport{steps: []recordedStep{{status: 503, body: "fail"}}}
	req, _ = httpx.NewRequest(ctx, "POST", "http://temba.io/", strings.NewReader("data"), nil)
	resp, err = httpx.WithRetries(inner, retries).RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, 503, resp.StatusCode)
	assert.Equal(t, 1, inner.calls)

	// retry on a connection error for an idempotent request
	inner = &recordingTransport{steps: []recordedStep{
		{status: 0}, // connection error
		{status: 200, body: "ok"},
	}}
	req, _ = httpx.NewRequest(ctx, "GET", "http://temba.io/", nil, nil)
	resp, err = httpx.WithRetries(inner, retries).RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 2, inner.calls)

	// a nil config makes it a pass-through with no retries
	inner = &recordingTransport{steps: []recordedStep{{status: 503, body: "fail"}}}
	req, _ = httpx.NewRequest(ctx, "GET", "http://temba.io/", nil, nil)
	resp, err = httpx.WithRetries(inner, nil).RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, 503, resp.StatusCode)
	assert.Equal(t, 1, inner.calls)
}

func TestWithRetriesBodyReplay(t *testing.T) {
	ctx := context.Background()
	retries := httpx.NewFixedRetries(time.Millisecond, time.Millisecond)

	inner := &recordingTransport{steps: []recordedStep{
		{status: 503, body: "fail"},
		{status: 200, body: "ok"},
	}}
	// a POST is idempotent here via the header, and strings.NewReader gives the request a GetBody so it can be replayed
	req, err := httpx.NewRequest(ctx, "POST", "http://temba.io/", strings.NewReader("payload"), map[string]string{"Idempotency-Key": "abc"})
	require.NoError(t, err)
	resp, err := httpx.WithRetries(inner, retries).RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	require.Equal(t, 2, inner.calls)

	// the full body was replayed on every attempt
	assert.Equal(t, "payload", string(inner.bodies[0]))
	assert.Equal(t, "payload", string(inner.bodies[1]))
}

func TestWithRetriesUnrewindableBody(t *testing.T) {
	ctx := context.Background()
	retries := httpx.NewFixedRetries(time.Millisecond, time.Millisecond)

	inner := &recordingTransport{steps: []recordedStep{
		{status: 503, body: "fail"},
		{status: 200, body: "ok"},
	}}
	// a body that isn't one of the rewindable types, so http.NewRequest leaves GetBody nil
	req, err := http.NewRequestWithContext(ctx, "POST", "http://temba.io/", io.NopCloser(strings.NewReader("payload")))
	require.NoError(t, err)
	req.Header.Set("Idempotency-Key", "abc") // ShouldRetry would allow a retry...
	require.Nil(t, req.GetBody)              // ...but the body can't be rewound

	resp, err := httpx.WithRetries(inner, retries).RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, 503, resp.StatusCode) // so we don't retry, and return the first response
	assert.Equal(t, 1, inner.calls)
}

func TestWithRetriesRetryAfter(t *testing.T) {
	ctx := context.Background()

	defer dates.SetNowFunc(time.Now)
	now := time.Date(2020, 1, 7, 15, 10, 30, 950000000, time.UTC)
	dates.SetNowFunc(dates.NewFixedNow(now))

	// a Retry-After 50ms in the future, with a backoff long enough to honour it
	retryAfter := now.Add(50 * time.Millisecond).UTC().Format(http.TimeFormat)
	retries := httpx.NewFixedRetries(50 * time.Millisecond)

	inner := &recordingTransport{steps: []recordedStep{
		{status: 429, headers: map[string]string{"Retry-After": retryAfter}, body: "slow down"},
		{status: 200, body: "ok"},
	}}
	req, err := httpx.NewRequest(ctx, "GET", "http://temba.io/", nil, nil)
	require.NoError(t, err)
	resp, err := httpx.WithRetries(inner, retries).RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 2, inner.calls)
}

func TestWithRetriesAndTraces(t *testing.T) {
	ctx := context.Background()

	defer dates.SetNowFunc(time.Now)
	dates.SetNowFunc(dates.NewSequentialNow(time.Date(2019, 10, 7, 15, 21, 30, 0, time.UTC), time.Second))

	retries := httpx.NewFixedRetries(time.Millisecond, time.Millisecond)

	inner := &recordingTransport{steps: []recordedStep{
		{status: 503, body: "fail"},
		{status: 503, body: "fail"},
		{status: 200, body: "ok"},
	}}

	// WithTraces(WithRetries(inner)) captures a single trace of the final attempt, with the retry count surfaced
	tt := httpx.WithTraces(httpx.WithRetries(inner, retries))
	req, err := httpx.NewRequest(ctx, "GET", "http://temba.io/", nil, nil)
	require.NoError(t, err)
	resp, err := tt.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "ok", string(body))

	require.Len(t, tt.Traces(), 1)
	trace := tt.Traces()[0]
	assert.Equal(t, 200, trace.Response.StatusCode)
	assert.Equal(t, "ok", string(trace.ResponseBody))
	assert.Equal(t, 2, trace.Retries)
}

func TestWithRetriesContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// a long backoff we should never actually wait out because the context is already cancelled
	retries := httpx.NewFixedRetries(time.Minute)

	inner := &recordingTransport{steps: []recordedStep{
		{status: 503, body: "fail"},
		{status: 200, body: "ok"},
	}}
	req, err := httpx.NewRequest(ctx, "GET", "http://temba.io/", nil, nil)
	require.NoError(t, err)

	cancel() // cancel before the backoff wait

	resp, err := httpx.WithRetries(inner, retries).RoundTrip(req)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, resp)
	assert.Equal(t, 1, inner.calls) // only the first attempt happened
}

func TestWithRetriesNilInner(t *testing.T) {
	ctx := context.Background()
	server := newTestHTTPServer(52030)
	defer server.Close()

	// a nil inner transport falls back to http.DefaultTransport
	tr := httpx.WithRetries(nil, httpx.NewFixedRetries(time.Millisecond))
	req, err := httpx.NewRequest(ctx, "GET", server.URL+"?cmd=success", nil, nil)
	require.NoError(t, err)
	resp, err := tr.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, `{ "ok": "true" }`, string(body))
}
