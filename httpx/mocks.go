package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"sync"

	"github.com/nyaruka/gocommon/jsonx"
	"github.com/nyaruka/gocommon/stringsx"
)

// MockRequestor is a requestor which can be mocked with responses for given URLs
//
// Deprecated: use the composable MockTransport (via WithMocks) instead; MockRequestor will be removed in a future release.
type MockRequestor struct {
	mocks       map[string][]*MockResponse
	requests    []*http.Request
	ignoreLocal bool
}

// NewMockRequestor creates a new mock requestor with the given mocks
//
// Deprecated: use WithMocks to build a MockTransport instead; MockRequestor will be removed in a future release.
func NewMockRequestor(mocks map[string][]*MockResponse) *MockRequestor {
	return &MockRequestor{mocks: mocks}
}

// SetIgnoreLocal sets whether the requestor should ignore requests on localhost and delegate these
// the the default requestor.
func (r *MockRequestor) SetIgnoreLocal(ignore bool) {
	r.ignoreLocal = ignore
}

// Do returns the mocked reponse for the given request
func (r *MockRequestor) Do(client *http.Client, request *http.Request) (*http.Response, error) {
	if r.ignoreLocal && isLocalRequest(request) {
		return DefaultRequestor.Do(client, request)
	}

	return r.RoundTrip(request)
}

// RoundTrip allows this to be used as a http.RoundTripper
func (r *MockRequestor) RoundTrip(request *http.Request) (*http.Response, error) {
	r.requests = append(r.requests, request)

	mocked := takeMock(r.mocks, request)
	if mocked == nil {
		panic(fmt.Sprintf("missing mock for URL %s", request.URL.String()))
	}

	if mocked.Status == 0 {
		return nil, errors.New("unable to connect to server")
	}

	return mocked.Make(request), nil
}

// Requests returns the received requests
func (r *MockRequestor) Requests() []*http.Request {
	return r.requests
}

// HasUnused returns true if there are unused mocks leftover
func (r *MockRequestor) HasUnused() bool {
	return hasUnusedMocks(r.mocks)
}

// Clone returns a clone of this requestor
func (r *MockRequestor) Clone() *MockRequestor {
	cloned := make(map[string][]*MockResponse)
	for url, ms := range r.mocks {
		cloned[url] = ms
	}
	return NewMockRequestor(cloned)
}

func (r *MockRequestor) MarshalJSON() ([]byte, error) {
	return jsonx.Marshal(&r.mocks)
}

func (r *MockRequestor) UnmarshalJSON(data []byte) error {
	return jsonx.Unmarshal(data, &r.mocks)
}

var _ Requestor = (*MockRequestor)(nil)

// takeMock pops the next mocked response matching the request's URL, mutating mocks. Returns nil if none match.
func takeMock(mocks map[string][]*MockResponse, request *http.Request) *MockResponse {
	url := request.URL.String()

	// find the most specific match against this URL
	match := stringsx.GlobSelect(url, slices.Collect(maps.Keys(mocks))...)
	mockedResponses := mocks[match]
	if len(mockedResponses) == 0 {
		return nil
	}

	// pop the next mocked response for this URL
	mocked := mockedResponses[0]
	remaining := mockedResponses[1:]
	if len(remaining) > 0 {
		mocks[match] = remaining
	} else {
		delete(mocks, match)
	}

	return mocked
}

// hasUnusedMocks returns whether any unused mocked responses remain
func hasUnusedMocks(mocks map[string][]*MockResponse) bool {
	for _, ms := range mocks {
		if len(ms) > 0 {
			return true
		}
	}
	return false
}

// MockTransport is an http.RoundTripper which answers requests from a set of mocked responses, delegating to an
// inner transport for requests it doesn't handle. It's intended to be the composable replacement for MockRequestor.
// It is safe for concurrent use by multiple goroutines, as the http.RoundTripper contract requires.
type MockTransport struct {
	inner       http.RoundTripper
	mutex       sync.Mutex // guards mocks and requests
	mocks       map[string][]*MockResponse
	requests    []*http.Request
	ignoreLocal bool
	passthrough bool
}

// MockOption configures a MockTransport created with WithMocks.
type MockOption func(*MockTransport)

// MockPassthrough makes a mocking transport delegate a request with no matching mock to the inner transport
// instead of panicking (the default).
func MockPassthrough() MockOption {
	return func(t *MockTransport) { t.passthrough = true }
}

// MockIgnoreLocal makes a mocking transport delegate requests to localhost to the inner transport rather than
// trying to mock them.
func MockIgnoreLocal() MockOption {
	return func(t *MockTransport) { t.ignoreLocal = true }
}

// WithMocks wraps an http.RoundTripper so that requests matching one of the given mocks are answered from the
// mock instead of being sent. If inner is nil then http.DefaultTransport is used. By default a request with no
// matching mock panics, mirroring MockRequestor; pass MockPassthrough to instead delegate such requests to the
// inner transport. The mocks map is copied, so the caller's map is never consumed and can be safely reused.
func WithMocks(inner http.RoundTripper, mocks map[string][]*MockResponse, opts ...MockOption) *MockTransport {
	if inner == nil {
		inner = http.DefaultTransport
	}
	t := &MockTransport{inner: inner, mocks: maps.Clone(mocks)}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t *MockTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	// delegate local requests to the inner transport when ignoring local
	if t.ignoreLocal && isLocalRequest(request) {
		return t.inner.RoundTrip(request)
	}

	// take the next matching mock and record the request under lock, but don't hold it across any delegation to
	// the inner transport which may block on I/O
	t.mutex.Lock()
	mocked := takeMock(t.mocks, request)
	if mocked != nil {
		t.requests = append(t.requests, request)
	}
	t.mutex.Unlock()

	if mocked == nil {
		// no matching mock - either pass through to the inner transport or panic to catch the unexpected request
		if t.passthrough {
			return t.inner.RoundTrip(request)
		}
		panic(fmt.Sprintf("missing mock for URL %s", request.URL.String()))
	}

	if mocked.Status == 0 {
		return nil, errors.New("unable to connect to server")
	}

	return mocked.Make(request), nil
}

// Requests returns a snapshot of the requests that were answered from mocks
func (t *MockTransport) Requests() []*http.Request {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return slices.Clone(t.requests)
}

// HasUnused returns true if there are unused mocks leftover
func (t *MockTransport) HasUnused() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return hasUnusedMocks(t.mocks)
}

var _ http.RoundTripper = (*MockTransport)(nil)

type MockResponse struct {
	Status       int
	Headers      map[string]string
	Body         []byte
	BodyIsString bool
	BodyRepeat   int
}

// Make mocks making the given request and returning this as the response
func (m *MockResponse) Make(request *http.Request) *http.Response {
	header := make(http.Header, len(m.Headers))
	for k, v := range m.Headers {
		header.Set(k, v)
	}

	body := m.Body
	if m.BodyRepeat > 1 {
		body = bytes.Repeat(body, m.BodyRepeat)
	}

	return &http.Response{
		Request:       request,
		Status:        fmt.Sprintf("%d %s", m.Status, http.StatusText(m.Status)),
		StatusCode:    m.Status,
		Proto:         "HTTP/1.0",
		ProtoMajor:    1,
		ProtoMinor:    0,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}

// MockConnectionError mocks a connection error
var MockConnectionError = &MockResponse{Status: 0, Headers: nil, Body: []byte{}, BodyIsString: true, BodyRepeat: 0}

// NewMockResponse creates a new mock response
func NewMockResponse(status int, headers map[string]string, body []byte) *MockResponse {
	return &MockResponse{Status: status, Headers: headers, Body: body, BodyIsString: true, BodyRepeat: 0}
}

func isLocalRequest(r *http.Request) bool {
	hostname := r.URL.Hostname()
	return hostname == "localhost" || hostname == "127.0.0.1"
}

//------------------------------------------------------------------------------------------
// JSON Encoding / Decoding
//------------------------------------------------------------------------------------------

type mockResponseEnvelope struct {
	Status     int               `json:"status" validate:"required"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       json.RawMessage   `json:"body" validate:"required"`
	BodyRepeat int               `json:"body_repeat,omitempty"`
}

func (m *MockResponse) MarshalJSON() ([]byte, error) {
	var body []byte
	if m.BodyIsString {
		body, _ = jsonx.Marshal(string(m.Body))
	} else {
		body = m.Body
	}

	return jsonx.Marshal(&mockResponseEnvelope{
		Status:     m.Status,
		Headers:    m.Headers,
		Body:       body,
		BodyRepeat: m.BodyRepeat,
	})
}

func (m *MockResponse) UnmarshalJSON(data []byte) error {
	e := &mockResponseEnvelope{}
	if err := json.Unmarshal(data, e); err != nil {
		return err
	}

	m.Status = e.Status
	m.Headers = e.Headers
	m.BodyRepeat = e.BodyRepeat

	if len(e.Body) > 0 && e.Body[0] == '"' {
		var bodyAsString string
		json.Unmarshal(e.Body, &bodyAsString)
		m.Body = []byte(bodyAsString)
		m.BodyIsString = true
	} else {
		m.Body = e.Body
	}

	return nil
}
