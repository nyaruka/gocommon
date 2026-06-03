package httpx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/nyaruka/gocommon/dates"
	"github.com/nyaruka/gocommon/random"
)

// RetryConfig configures if and how retrying of requests happens
type RetryConfig struct {
	Backoffs    []time.Duration
	Jitter      float64
	ShouldRetry func(*http.Request, *http.Response, time.Duration) bool
}

// NewFixedRetries creates a new retry config with the given backoffs
func NewFixedRetries(backoffs ...time.Duration) *RetryConfig {
	return &RetryConfig{Backoffs: backoffs, ShouldRetry: DefaultShouldRetry}
}

// NewExponentialRetries creates a new retry config with the given delays
func NewExponentialRetries(initialBackoff time.Duration, count int, jitter float64) *RetryConfig {
	backoffs := make([]time.Duration, count)
	backoffs[0] = initialBackoff
	for i := 1; i < count; i++ {
		backoffs[i] = backoffs[i-1] * 2
	}

	return &RetryConfig{Backoffs: backoffs, Jitter: jitter, ShouldRetry: DefaultShouldRetry}
}

// MaxRetries gets the maximum number of retries allowed
func (r *RetryConfig) MaxRetries() int {
	return len(r.Backoffs)
}

// Backoff gets the backoff time for the nth retry
func (r *RetryConfig) Backoff(n int) time.Duration {
	if n >= len(r.Backoffs) {
		panic(fmt.Sprintf("%d not a valid retry number for this config", n))
	}

	base := r.Backoffs[n]
	jitter := time.Duration(r.Jitter * float64(random.IntN(int(base))-(int(base)/2)))
	return base + jitter
}

// DefaultShouldRetry is the default function for determining if a response should be retried
func DefaultShouldRetry(request *http.Request, response *http.Response, withDelay time.Duration) bool {
	// any response with a Retry-After header is candidate for a retry (usually used with 301, 429, 503 status codes)
	if response != nil {
		retryAfter := response.Header.Get("Retry-After")
		if retryAfter != "" {
			requestedDelay := ParseRetryAfter(retryAfter)

			// as long as the server has requested a delay which is less than or equal to what we intended
			if requestedDelay != 0 && requestedDelay <= withDelay {
				return true
			}
		}
	}

	// otherwise retry if request is idempotent and response is a failure (excluding 500 and 501)
	return isIdempotent(request) && (response == nil || response.StatusCode > 501)
}

// see https://github.com/golang/go/blob/100bf440b9a69c6dce8daeebed038d607c963b8f/src/net/http/request.go#L1395
func isIdempotent(r *http.Request) bool {
	switch r.Method {
	case "GET", "HEAD", "OPTIONS", "TRACE":
		return true
	}

	return r.Header.Get("Idempotency-Key") != "" || r.Header.Get("X-Idempotency-Key") != ""
}

// ParseRetryAfter parses value of Retry-After headers which can be date or delay in seconds
// see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Retry-After
func ParseRetryAfter(value string) time.Duration {
	asTime, err := http.ParseTime(value)
	if err == nil {
		delta := asTime.Sub(dates.Now())
		if delta >= 0 {
			return delta
		}
	} else {
		asSeconds, err := strconv.Atoi(value)
		if err == nil {
			return time.Duration(asSeconds) * time.Second
		}
	}

	return 0
}

// maxRetryDrain caps how many bytes of a to-be-discarded response body we drain before retrying. Draining lets the
// underlying connection be reused, but an unbounded drain of a large or slow error body could cost more than the
// reuse it buys, so we bound it to the same size net/http.Transport uses for the same purpose.
const maxRetryDrain = 2 << 10 // 2KB

// retryTransport is an http.RoundTripper which retries requests according to a RetryConfig, delegating each attempt
// to an inner transport. It holds no mutable state, so it's safe for concurrent use by multiple goroutines, as the
// http.RoundTripper contract requires.
type retryTransport struct {
	inner   http.RoundTripper
	retries *RetryConfig
}

// WithRetries wraps an http.RoundTripper so that requests are retried with backoff according to the given RetryConfig.
// After each attempt, while retries remain, it asks retries.ShouldRetry whether the request/response warrants another
// try and, if so, waits the configured backoff before retrying. A nil config makes it a pass-through, so it's always
// safe to wrap. If inner is nil then http.DefaultTransport is used.
//
// A request with a body is only retried if the body can be rewound, i.e. it has a non-nil GetBody (which
// http.NewRequest populates automatically for the common in-memory body types). This mirrors how net/http.Transport
// itself decides whether a request is replayable (see https://github.com/golang/go/issues/18241). Between attempts
// the discarded response body is drained and closed so the underlying connection can be reused, and the backoff wait
// is aborted early if the request's context is cancelled.
//
// To recover how many retries happened, compose this inside WithTraces:
//
//	httpx.WithTraces(httpx.WithRetries(inner, retries))
//
// In that arrangement the outer tracer captures a single Trace of the final attempt, with Trace.Retries set to the
// number of retries performed. (Composed the other way around, httpx.WithRetries(httpx.WithTraces(inner), retries),
// each attempt is captured as its own trace and the count is len(traces)-1.)
func WithRetries(inner http.RoundTripper, retries *RetryConfig) http.RoundTripper {
	if inner == nil {
		inner = http.DefaultTransport
	}
	return &retryTransport{inner: inner, retries: retries}
}

func (t *retryTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	// an outer TracesTransport may have installed a counter for us to report retries through
	count := retryCountFromContext(request.Context())

	retry := 0
	for {
		response, err := t.inner.RoundTrip(request)

		// stop if there's no retry config or we've used up our allowance
		if t.retries == nil || retry >= t.retries.MaxRetries() {
			return response, err
		}

		backoff := t.retries.Backoff(retry)
		if !t.retries.ShouldRetry(request, response, backoff) {
			return response, err
		}

		// we can only make another attempt if there's no body to resend or we can rewind it via GetBody; an empty
		// http.NoBody needs no rewinding. This mirrors net/http.Transport's own conservative behaviour for replaying
		// requests (see https://github.com/golang/go/issues/18241).
		if request.Body != nil && request.Body != http.NoBody && request.GetBody == nil {
			return response, err
		}

		// drain and close the response we're discarding so the underlying connection can be reused; cap the drain
		// (as net/http.Transport does) so a large or slow error body doesn't cost more than the reuse it buys
		if response != nil {
			io.CopyN(io.Discard, response.Body, maxRetryDrain)
			response.Body.Close()
		}

		// wait out the backoff, but abort early if the request's context is cancelled
		if werr := wait(request.Context(), backoff); werr != nil {
			return nil, werr
		}

		// rewind the body for the next attempt by cloning the request with a fresh body from GetBody; cloning
		// rather than mutating leaves the caller's request untouched
		if request.Body != nil && request.Body != http.NoBody {
			body, gerr := request.GetBody()
			if gerr != nil {
				return nil, gerr
			}
			request = request.Clone(request.Context())
			request.Body = body
		}

		retry++
		if count != nil {
			count.Add(1)
		}
	}
}

var _ http.RoundTripper = (*retryTransport)(nil)

// wait blocks for the given duration, returning early with the context's error if it is cancelled first.
func wait(ctx context.Context, d time.Duration) error {
	// check for cancellation up front so a zero or already-elapsed backoff can't race the select into returning nil
	// when the context is already done
	if err := ctx.Err(); err != nil {
		return err
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// retryCountKey is the unexported context key under which a retry counter is carried so that an outer TracesTransport
// can recover how many retries an inner retryTransport performed. See WithRetries.
type retryCountKey struct{}

// contextWithRetryCount returns a copy of ctx carrying a fresh retry counter, along with that counter. An outer
// transport (e.g. the one built by WithTraces) installs this before delegating so that an inner retryTransport can
// record how many retries it made and the outer transport can read the count back when finalizing its trace.
func contextWithRetryCount(ctx context.Context) (context.Context, *atomic.Int64) {
	count := &atomic.Int64{}
	return context.WithValue(ctx, retryCountKey{}, count), count
}

// retryCountFromContext returns the retry counter carried in ctx, or nil if none was installed.
func retryCountFromContext(ctx context.Context) *atomic.Int64 {
	count, _ := ctx.Value(retryCountKey{}).(*atomic.Int64)
	return count
}
