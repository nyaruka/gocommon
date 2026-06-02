package httpx

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gabriel-vasile/mimetype"
	"github.com/nyaruka/gocommon/dates"
)

// ErrResponseSize is returned when response size exceeds provided limit
var ErrResponseSize = errors.New("response body exceeds size limit")

// ErrAccessConfig is returned when provided access config prevents request
var ErrAccessConfig = errors.New("request not permitted by access config")

// Do makes the given HTTP request using the current requestor and retry config
func Do(client *http.Client, request *http.Request, retries *RetryConfig, access *AccessConfig) (*http.Response, error) {
	r, _, err := do(client, request, retries, access)
	return r, err
}

func do(client *http.Client, request *http.Request, retries *RetryConfig, access *AccessConfig) (*http.Response, int, error) {
	if err := access.check(request); err != nil {
		return nil, 0, err
	}

	var response *http.Response
	var err error
	retry := 0

	for {
		response, err = currentRequestor.Do(client, request)

		if retries != nil && retry < retries.MaxRetries() {
			backoff := retries.Backoff(retry)

			if retries.ShouldRetry(request, response, backoff) {
				time.Sleep(backoff)
				retry++
				continue
			}
		}

		break
	}

	return response, retry, err
}

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

// DoTrace makes the given request saving traces of the complete request and response.
//
//   - If the request is successful, the trace will have a response and response body
//   - If reading the body errors, the trace will have a response but no response body
//   - If connection fails, the trace will have a request but no response or response body
func DoTrace(client *http.Client, request *http.Request, retries *RetryConfig, access *AccessConfig, maxBodyBytes int) (*Trace, error) {
	requestTrace, err := httputil.DumpRequestOut(request, true)
	if err != nil {
		return nil, err
	}

	trace := &Trace{
		Request:      request,
		RequestTrace: requestTrace,
		StartTime:    dates.Now(),
	}
	defer func() { trace.EndTime = dates.Now() }()

	response, retryCount, err := do(client, request, retries, access)
	trace.Response = response
	trace.Retries = retryCount

	if err != nil {
		return trace, err
	}

	trace.ResponseTrace, trace.ResponseBody, err = dumpResponse(response, maxBodyBytes)
	if err != nil {
		return trace, err
	}

	return trace, nil
}

// TracingTransport is an http.RoundTripper which captures a Trace of each request and response, delegating to an
// inner transport. The response body is buffered so that it remains readable by the caller.
type TracingTransport struct {
	inner        http.RoundTripper
	maxBodyBytes int
	traces       []*Trace
}

// WithTracing wraps an http.RoundTripper so that each request and response is captured as a *Trace, retrievable via
// Traces(). The response body is buffered so it remains readable by the caller; at most maxBodyBytes of it are
// captured into the trace (a value <= 0 captures the entire body). If inner is nil then http.DefaultTransport is used.
func WithTracing(inner http.RoundTripper, maxBodyBytes int) *TracingTransport {
	if inner == nil {
		inner = http.DefaultTransport
	}
	return &TracingTransport{inner: inner, maxBodyBytes: maxBodyBytes}
}

func (t *TracingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	requestTrace, err := httputil.DumpRequestOut(request, true)
	if err != nil {
		return nil, err
	}

	trace := &Trace{
		Request:      request,
		RequestTrace: requestTrace,
		StartTime:    dates.Now(),
	}
	t.traces = append(t.traces, trace)
	defer func() { trace.EndTime = dates.Now() }()

	response, err := t.inner.RoundTrip(request)
	trace.Response = response
	if err != nil {
		return response, err
	}

	// dump the response trace without the body, which we capture separately
	responseTrace, err := httputil.DumpResponse(response, false)
	if err != nil {
		return response, err
	}
	trace.ResponseTrace = responseTrace

	// read the full body so we can both capture it and hand a readable copy back to the caller
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return response, err
	}
	response.Body = io.NopCloser(bytes.NewReader(body))

	// capture up to maxBodyBytes of the body into the trace
	if t.maxBodyBytes > 0 && len(body) > t.maxBodyBytes {
		trace.ResponseBody = body[:t.maxBodyBytes]
	} else {
		trace.ResponseBody = body
	}

	return response, nil
}

// Traces returns the traces captured so far
func (t *TracingTransport) Traces() []*Trace {
	return t.traces
}

var _ http.RoundTripper = (*TracingTransport)(nil)

// NewRequest is a convenience method to create a request with the given context and headers
func NewRequest(ctx context.Context, method string, url string, body io.Reader, headers map[string]string) (*http.Request, error) {
	r, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		r.Header.Set(key, value)
	}

	return r, nil
}

func dumpResponse(response *http.Response, maxBodyBytes int) ([]byte, []byte, error) {
	// dump response trace without body which will be parsed separately
	responseTrace, err := httputil.DumpResponse(response, false)
	if err != nil {
		return nil, nil, err
	}
	responseBody, err := readBody(response, maxBodyBytes)
	if err != nil {
		return responseTrace, nil, err
	}

	return responseTrace, responseBody, nil
}

// attempts to read the body of an HTTP response
func readBody(response *http.Response, maxBodyBytes int) ([]byte, error) {
	defer response.Body.Close()

	if maxBodyBytes > 0 {
		// we will only read up to our max body bytes limit
		bodyReader := io.LimitReader(response.Body, int64(maxBodyBytes)+1)

		bodyBytes, err := io.ReadAll(bodyReader)
		if err != nil {
			return nil, err
		}

		// if we have no remaining bytes, error because the body was too big
		if bodyReader.(*io.LimitedReader).N <= 0 {
			return nil, ErrResponseSize
		}

		return bodyBytes, nil
	}

	// if there is no limit, read the entire body
	return io.ReadAll(response.Body)
}

// Requestor is anything that can make an HTTP request with a client
type Requestor interface {
	Do(*http.Client, *http.Request) (*http.Response, error)
}

type defaultRequestor struct{}

func (r defaultRequestor) Do(client *http.Client, request *http.Request) (*http.Response, error) {
	return client.Do(request)
}

// DefaultRequestor is the default HTTP requestor
var DefaultRequestor Requestor = defaultRequestor{}
var currentRequestor = DefaultRequestor

// SetRequestor sets the requestor used by Request
func SetRequestor(requestor Requestor) {
	currentRequestor = requestor
}

// DetectContentType is a replacement for http.DetectContentType which leans on the github.com/gabriel-vasile/mimetype
// library to support more types, and additionally returns the extension (including leading period) associated with the
// detected type.
func DetectContentType(d []byte) (string, string) {
	mime := mimetype.Detect(d)
	return mime.String(), mime.Extension()
}

// BasicAuth returns the Authorization header value for HTTP Basic auth
func BasicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
