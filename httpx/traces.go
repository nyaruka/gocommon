package httpx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/nyaruka/gocommon/dates"
)

// Trace holds the complete trace of an HTTP request/response
type Trace struct {
	Request       *http.Request
	RequestTrace  []byte
	Response      *http.Response
	ResponseTrace []byte
	ResponseBody  []byte // response body stored separately
	StartTime     time.Time
	EndTime       time.Time
	Retries       int
}

func (t *Trace) String() string {
	b := &strings.Builder{}
	b.WriteString(fmt.Sprintf(">>>>>>>> %s %s\n", t.Request.Method, t.Request.URL))
	b.WriteString(string(t.RequestTrace))
	b.WriteString("\n<<<<<<<<\n")
	b.WriteString(string(t.ResponseTrace))
	b.WriteString(string(t.ResponseBody))
	return b.String()
}

// RequestSize returns the size in bytes of the request (headers and body).
func (t *Trace) RequestSize() int {
	return len(t.RequestTrace)
}

// ResponseSize returns the size in bytes of the response (headers and body). If the body was discarded by the caller
// (e.g. because reading it exceeded a limit set with WithReadLimit and failed with ErrResponseSize) then the server
// declared Content-Length, if any, is used as the best available record of its true size. Chunked or decompressed
// responses declare no length and so will under-report.
func (t *Trace) ResponseSize() int {
	bodySize := len(t.ResponseBody)
	if t.ResponseBody == nil && t.Response != nil && t.Response.ContentLength > 0 {
		bodySize = int(t.Response.ContentLength)
	}
	return len(t.ResponseTrace) + bodySize
}

// SanitizedRequest returns a valid UTF-8 string version of the request, substituting the body with a placeholder
// if it isn't valid UTF-8. It also strips any NULL characters as not all external dependencies can handle those.
func (t *Trace) SanitizedRequest(placeholder string) string {
	// split request trace into headers and body
	var headers, body []byte
	parts := bytes.SplitN(t.RequestTrace, []byte("\r\n\r\n"), 2)
	headers = append(parts[0], []byte("\r\n\r\n")...)

	if len(parts) > 1 {
		body = parts[1]
	} else {
		body = nil
	}

	return santizedTrace(headers, body, placeholder)
}

// SanitizedResponse returns a valid UTF-8 string version of the response, substituting the body with a placeholder
// if it isn't valid UTF-8. It also strips any NULL characters as not all external dependencies can handle those.
func (t *Trace) SanitizedResponse(placeholder string) string {
	return santizedTrace(t.ResponseTrace, t.ResponseBody, placeholder)
}

func santizedTrace(header []byte, body []byte, bodyPlaceHolder string) string {
	b := &bytes.Buffer{}

	// ensure headers section is valid
	b.Write(replaceNullChars(bytes.ToValidUTF8(header, []byte(`�`))))

	// only include body if it's valid UTF-8 as it could be a binary file or anything
	if utf8.Valid(body) {
		b.Write(replaceNullChars(body))
	} else {
		b.Write([]byte(bodyPlaceHolder))
	}

	return b.String()
}

func replaceNullChars(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte{0}, []byte(`�`))
}

// TracesTransport is an http.RoundTripper which captures a Trace of each request and response, delegating to an
// inner transport. The response body is buffered so that it remains readable by the caller. It is safe for
// concurrent use by multiple goroutines, as the http.RoundTripper contract requires.
type TracesTransport struct {
	inner  http.RoundTripper
	mutex  sync.Mutex
	traces []*Trace
}

// WithTraces wraps an http.RoundTripper so that each request and response is captured as a *Trace, retrievable via
// Traces(). The response body is buffered so it remains readable by the caller, and the full body that was read is
// captured into the trace. To bound how many bytes are read from an untrusted endpoint, wrap the inner transport with
// WithReadLimit, e.g. WithTraces(WithReadLimit(inner, n)). If inner is nil then http.DefaultTransport is used.
func WithTraces(inner http.RoundTripper) *TracesTransport {
	if inner == nil {
		inner = http.DefaultTransport
	}
	return &TracesTransport{inner: inner}
}

func (t *TracesTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	requestTrace, err := httputil.DumpRequestOut(request, true)
	if err != nil {
		// the http.RoundTripper contract requires the request body to be closed even on error paths
		if request.Body != nil {
			request.Body.Close()
		}
		return nil, err
	}

	// carry a counter so that an inner retryTransport, if composed inside us, can report how many retries it made,
	// which we then surface as Trace.Retries
	ctx, retryCount := contextWithRetryCount(request.Context())
	request = request.WithContext(ctx)

	trace := &Trace{
		Request:      request,
		RequestTrace: requestTrace,
		StartTime:    dates.Now(),
	}
	t.mutex.Lock()
	t.traces = append(t.traces, trace)
	t.mutex.Unlock()

	// also record into the per-call collector carried by the request context, if there is one, so that a caller
	// sharing this transport can pick out the traces of just its own call
	if c := traceCollectorFromContext(request.Context()); c != nil {
		c.add(trace)
	}

	defer func() { trace.EndTime = dates.Now() }()

	response, err := t.inner.RoundTrip(request)
	trace.Response = response
	trace.Retries = int(retryCount.Load())
	if err != nil {
		// the inner transport failed to obtain a response
		return nil, err
	}

	// dump the response trace without the body, which we capture separately; ignore any dump error as we still
	// have a usable response to hand back to the caller
	trace.ResponseTrace, _ = httputil.DumpResponse(response, false)

	// read the full body so we can both capture it and hand a readable copy back to the caller
	body, readErr := io.ReadAll(response.Body)
	response.Body.Close()

	// capture the full body that was read into the trace
	trace.ResponseBody = body

	// restore a readable body for the caller; if reading it failed, replay the partial bytes and the error so the
	// caller sees exactly what it would have without tracing
	if readErr != nil {
		response.Body = io.NopCloser(io.MultiReader(bytes.NewReader(body), errReader{readErr}))
	} else {
		response.Body = io.NopCloser(bytes.NewReader(body))
	}

	return response, nil
}

// Traces returns a snapshot of the traces captured so far
func (t *TracesTransport) Traces() []*Trace {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return slices.Clone(t.traces)
}

var _ http.RoundTripper = (*TracesTransport)(nil)

type traceCollectorKey struct{}

// TraceCollector accumulates the traces of the requests made under a particular context. It is safe for concurrent
// use by multiple goroutines.
type TraceCollector struct {
	mutex  sync.Mutex
	traces []*Trace
}

// WithTraceCollector returns a copy of ctx carrying a fresh TraceCollector, along with that collector. Any request
// made with the returned context through a transport wrapped by WithTraces records its trace into the collector.
//
// This is how a caller captures the traces of just its own call while the client, and the tracing transport on it,
// are shared - so the tracing can be installed once when the client is built rather than assembled per call. It also
// works when the request is issued by code the caller doesn't control, such as a vendor SDK given the client at
// construction: pass the context into the SDK call and the traces come back here.
//
//	client := &http.Client{Transport: httpx.WithTraces(base)} // once
//
//	ctx, traces := httpx.WithTraceCollector(ctx)              // per call
//	resp, err := client.Do(req.WithContext(ctx))
//	trace := traces.Last()
func WithTraceCollector(ctx context.Context) (context.Context, *TraceCollector) {
	c := &TraceCollector{}
	return context.WithValue(ctx, traceCollectorKey{}, c), c
}

// traceCollectorFromContext returns the collector carried in ctx, or nil if none was installed.
func traceCollectorFromContext(ctx context.Context) *TraceCollector {
	c, _ := ctx.Value(traceCollectorKey{}).(*TraceCollector)
	return c
}

func (c *TraceCollector) add(t *Trace) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.traces = append(c.traces, t)
}

// Traces returns a snapshot of the traces collected so far, in the order the requests were made.
func (c *TraceCollector) Traces() []*Trace {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return slices.Clone(c.traces)
}

// Last returns the most recently started trace, or nil if none were collected. Where a request was redirected that's
// the final hop, which is usually the only one worth logging - the earlier hops being the 3xx responses that led to
// it. A nil return means no request reached the tracing transport at all.
func (c *TraceCollector) Last() *Trace {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if len(c.traces) == 0 {
		return nil
	}
	return c.traces[len(c.traces)-1]
}

// errReader is an io.Reader that always returns its error, used to replay a body read failure to the caller
type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }
