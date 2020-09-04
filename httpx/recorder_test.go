package httpx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/nyaruka/gocommon/httpx"

	"github.com/stretchr/testify/assert"
)

func TestRecorder(t *testing.T) {
	var request *http.Request
	var trace *httpx.Trace
	var err error

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request = r
		recorder := httpx.NewRecorder(r, w)
		w = recorder.ResponseWriter

		w.Header().Set("Date", "Wed, 11 Apr 2018 18:24:30 GMT")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "OK"}`))

		trace, err = recorder.End()
	}))

	req, _ := httpx.NewRequest("GET", server.URL, nil, nil)
	httpx.Do(http.DefaultClient, req, nil, nil)

	su, _ := url.Parse(server.URL)

	assert.NoError(t, err)
	assert.Equal(t, request, trace.Request)
	assert.Equal(t, fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s:%s\r\nAccept-Encoding: gzip\r\nUser-Agent: Go-http-client/1.1\r\n\r\n", su.Hostname(), su.Port()), string(trace.RequestTrace))
	assert.Equal(t, 200, trace.Response.StatusCode)
	assert.Equal(t, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nDate: Wed, 11 Apr 2018 18:24:30 GMT\r\n\r\n", string(trace.ResponseTrace))
	assert.Equal(t, `{"status": "OK"}`, string(trace.ResponseBody))
}
