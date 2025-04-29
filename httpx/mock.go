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

	"github.com/nyaruka/gocommon/jsonx"
	"github.com/nyaruka/gocommon/stringsx"
)

// MockRequestor is a requestor which can be mocked with responses for given URLs
type MockRequestor struct {
	mocks       map[string][]*MockResponse
	requests    []*http.Request
	ignoreLocal bool
}

// NewMockRequestor creates a new mock requestor with the given mocks
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

	url := request.URL.String()

	// find the most specific match against this URL
	match := stringsx.GlobSelect(url, slices.Collect(maps.Keys(r.mocks))...)
	mockedResponses := r.mocks[match]

	if len(mockedResponses) == 0 {
		panic(fmt.Sprintf("missing mock for URL %s", url))
	}

	// pop the next mocked response for this URL
	mocked := mockedResponses[0]
	remaining := mockedResponses[1:]

	if len(remaining) > 0 {
		r.mocks[match] = remaining
	} else {
		delete(r.mocks, match)
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
	for _, mocks := range r.mocks {
		if len(mocks) > 0 {
			return true
		}
	}
	return false
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
