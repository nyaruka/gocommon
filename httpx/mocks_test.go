package httpx_test

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/nyaruka/gocommon/httpx"
	"github.com/nyaruka/gocommon/jsonx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMocksTransport(t *testing.T) {
	// a matched request is answered from the mock and recorded
	mt := httpx.WithMocks(http.DefaultTransport, map[string][]*httpx.MockResponse{
		"https://temba.io": {httpx.NewMockResponse(200, nil, []byte("hi"))},
	})
	req, err := http.NewRequest("GET", "https://temba.io", nil)
	require.NoError(t, err)
	resp, err := mt.RoundTrip(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Len(t, mt.Requests(), 1)
	assert.False(t, mt.HasUnused())

	// the caller's map is copied, not consumed - so it can be safely reused across runs without Clone()
	original := map[string][]*httpx.MockResponse{
		"https://temba.io": {httpx.NewMockResponse(200, nil, nil)},
	}
	mt = httpx.WithMocks(http.DefaultTransport, original)
	req, err = http.NewRequest("GET", "https://temba.io", nil)
	require.NoError(t, err)
	_, err = mt.RoundTrip(req)
	require.NoError(t, err)
	assert.False(t, mt.HasUnused())                // the transport's copy is exhausted
	assert.Len(t, original["https://temba.io"], 1) // but the caller's map is untouched

	// a mocked connection error is returned as an error
	mt = httpx.WithMocks(http.DefaultTransport, map[string][]*httpx.MockResponse{
		"https://temba.io": {httpx.MockConnectionError},
	})
	req, err = http.NewRequest("GET", "https://temba.io", nil)
	require.NoError(t, err)
	resp, err = mt.RoundTrip(req)
	assert.EqualError(t, err, "unable to connect to server")
	assert.Nil(t, resp)

	// a nil inner transport falls back to http.DefaultTransport; the mock answers so it's never called
	mt = httpx.WithMocks(nil, map[string][]*httpx.MockResponse{
		"https://temba.io": {httpx.NewMockResponse(200, nil, nil)},
	})
	assert.NotNil(t, mt)
	req, err = http.NewRequest("GET", "https://temba.io", nil)
	require.NoError(t, err)
	resp, err = mt.RoundTrip(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// by default a request with no matching mock panics
	mt = httpx.WithMocks(http.DefaultTransport, nil)
	req, err = http.NewRequest("GET", "https://temba.io", nil)
	require.NoError(t, err)
	assert.Panics(t, func() { mt.RoundTrip(req) })

	// with MockPassthrough, a request with no matching mock is delegated to the inner transport
	inner := httpx.WithMocks(http.DefaultTransport, map[string][]*httpx.MockResponse{
		"https://temba.io": {httpx.NewMockResponse(418, nil, nil)},
	})
	mt = httpx.WithMocks(inner, nil, httpx.MockPassthrough())
	req, err = http.NewRequest("GET", "https://temba.io", nil)
	require.NoError(t, err)
	resp, err = mt.RoundTrip(req)
	assert.NoError(t, err)
	assert.Equal(t, 418, resp.StatusCode)
	assert.Len(t, inner.Requests(), 1)
	assert.Empty(t, mt.Requests())

	// a matched request is still answered from the mock even in passthrough mode
	inner = httpx.WithMocks(http.DefaultTransport, map[string][]*httpx.MockResponse{})
	mt = httpx.WithMocks(inner, map[string][]*httpx.MockResponse{
		"https://temba.io": {httpx.NewMockResponse(200, nil, nil)},
	}, httpx.MockPassthrough())
	req, err = http.NewRequest("GET", "https://temba.io", nil)
	require.NoError(t, err)
	resp, err = mt.RoundTrip(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Len(t, mt.Requests(), 1)
	assert.Empty(t, inner.Requests())

	// with MockIgnoreLocal, a local request is delegated to the inner transport rather than mocked
	inner = httpx.WithMocks(http.DefaultTransport, map[string][]*httpx.MockResponse{
		"http://localhost/health": {httpx.NewMockResponse(200, nil, nil)},
	})
	mt = httpx.WithMocks(inner, map[string][]*httpx.MockResponse{}, httpx.MockIgnoreLocal())
	req, err = http.NewRequest("GET", "http://localhost/health", nil)
	require.NoError(t, err)
	resp, err = mt.RoundTrip(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Len(t, inner.Requests(), 1)
	assert.Empty(t, mt.Requests())
}

func TestMocksTransportConcurrent(t *testing.T) {
	// a MocksTransport shared across a client used by multiple goroutines must be safe for concurrent use, as the
	// http.RoundTripper contract requires - run under -race to detect any unsynchronized access to mocks/requests
	const n = 50
	mocks := make([]*httpx.MockResponse, n)
	for i := range mocks {
		mocks[i] = httpx.NewMockResponse(200, nil, []byte("hi"))
	}
	mt := httpx.WithMocks(http.DefaultTransport, map[string][]*httpx.MockResponse{
		"https://temba.io": mocks,
	})

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("GET", "https://temba.io", nil)
			resp, err := mt.RoundTrip(req)
			if assert.NoError(t, err) {
				io.ReadAll(resp.Body)
				resp.Body.Close()
			}
		}()
	}
	wg.Wait()

	assert.Len(t, mt.Requests(), n)
	assert.False(t, mt.HasUnused())
}

func TestMockResponseMarshaling(t *testing.T) {
	mocks := map[string][]*httpx.MockResponse{
		"http://google.com": {
			httpx.NewMockResponse(200, nil, []byte("this is google")),
			httpx.NewMockResponse(201, nil, []byte("this is google again")),
			&httpx.MockResponse{
				Status: 202,
				Body:   []byte(`{"foo": "bar"}`),
			},
		},
		"http://yahoo.com": {
			httpx.NewMockResponse(202, nil, []byte("this is yahoo")),
			httpx.MockConnectionError,
		},
	}

	asJSON := []byte(`{
		"http://google.com": [
			{"status": 200, "body": "this is google"},
			{"status": 201, "body": "this is google again"},
			{"status": 202, "body": {"foo": "bar"}}
		],
		"http://yahoo.com": [
			{"status": 202, "body": "this is yahoo"},
			{"status": 0, "body": ""}
		]
	}`)

	// test unmarshaling
	unmarshaled := map[string][]*httpx.MockResponse{}
	err := json.Unmarshal(asJSON, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, mocks, unmarshaled)

	// test re-marshaling
	marshaled, err := jsonx.Marshal(unmarshaled)
	assert.NoError(t, err)
	assert.JSONEq(t, string(asJSON), string(marshaled))
}
