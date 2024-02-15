package httpx_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nyaruka/gocommon/httpx"
	"github.com/nyaruka/gocommon/jsonx"

	"github.com/stretchr/testify/assert"
)

func TestMockRequestor(t *testing.T) {
	defer httpx.SetRequestor(httpx.DefaultRequestor)

	// start a real HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`COOL`))
	}))
	defer server.Close()

	// can create requestor with constructor
	requestor1 := httpx.NewMockRequestor(map[string][]*httpx.MockResponse{
		"http://google.com": {
			httpx.NewMockResponse(200, nil, []byte("this is google")),
			httpx.NewMockResponse(201, nil, []byte("this is google again")),
		},
		"http://yahoo.com": {
			httpx.NewMockResponse(202, nil, []byte("this is yahoo")),
			httpx.MockConnectionError,
		},
		"http://*": {
			httpx.NewMockResponse(203, nil, []byte("this is partial")),
		},
		"*": {
			httpx.NewMockResponse(204, nil, []byte("this is wild")),
		},
		server.URL + "/thing": {
			httpx.NewMockResponse(205, nil, []byte("this is local")),
		},
		"http://example.com": {
			httpx.NewMockResponse(200, nil, []byte("this should match ignoring the params requested")),
		},
	})

	httpx.SetRequestor(requestor1)

	req1, _ := http.NewRequest("GET", "http://google.com", nil)
	response, err := httpx.Do(http.DefaultClient, req1, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)

	body, err := io.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.Equal(t, "this is google", string(body))

	assert.Equal(t, []*http.Request{req1}, requestor1.Requests())
	assert.True(t, requestor1.HasUnused())

	// request another mocked URL
	req2, _ := http.NewRequest("GET", "http://yahoo.com", nil)
	response, err = httpx.Do(http.DefaultClient, req2, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 202, response.StatusCode)
	assert.Equal(t, []*http.Request{req1, req2}, requestor1.Requests())

	// request second mock for first URL
	req3, _ := http.NewRequest("GET", "http://google.com", nil)
	response, err = httpx.Do(http.DefaultClient, req3, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 201, response.StatusCode)

	// request mocked connection error
	req4, _ := http.NewRequest("GET", "http://yahoo.com", nil)
	response, err = httpx.Do(http.DefaultClient, req4, nil, nil)
	assert.EqualError(t, err, "unable to connect to server")
	assert.Nil(t, response)

	// request mocked localhost request
	req5, _ := http.NewRequest("GET", server.URL+"/thing", nil)
	response, err = httpx.Do(http.DefaultClient, req5, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 205, response.StatusCode)

	// match against http://*
	req6, _ := http.NewRequest("GET", "http://yahoo.com", nil)
	response, err = httpx.Do(http.DefaultClient, req6, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 203, response.StatusCode)

	// match against *
	req7, _ := http.NewRequest("GET", "http://yahoo.com", nil)
	response, err = httpx.Do(http.DefaultClient, req7, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 204, response.StatusCode)

	req8, _ := http.NewRequest("GET", "http://example.com?id=1&active=yes", nil)
	response, err = httpx.Do(http.DefaultClient, req8, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)

	assert.False(t, requestor1.HasUnused())

	// panic if we've run out of mocks for a URL
	req9, _ := http.NewRequest("GET", "http://google.com", nil)
	assert.Panics(t, func() { httpx.Do(http.DefaultClient, req9, nil, nil) })

	requestor1.SetIgnoreLocal(true)

	// now a request to the local server should actually get there
	req10, _ := http.NewRequest("GET", server.URL+"/thing", nil)
	response, err = httpx.Do(http.DefaultClient, req10, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
}

func TestMockRequestorMarshaling(t *testing.T) {
	// can create requestor with constructor
	requestor1 := httpx.NewMockRequestor(map[string][]*httpx.MockResponse{
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
	})

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
	requestor2 := &httpx.MockRequestor{}
	err := json.Unmarshal(asJSON, requestor2)
	assert.NoError(t, err)
	assert.Equal(t, requestor1, requestor2)

	// test re-marshaling
	marshaled, err := jsonx.Marshal(requestor2)
	assert.NoError(t, err)
	assert.JSONEq(t, string(asJSON), string(marshaled))
}
