package httpx_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/nyaruka/gocommon/httpx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecorder(t *testing.T) {
	var trace *httpx.Trace

	readBody := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder, err := httpx.NewRecorder(r, w)
		require.NoError(t, err)
		w = recorder.ResponseWriter

		if readBody {
			io.ReadAll(r.Body)
		}

		w.Header().Set("Date", "Wed, 11 Apr 2018 18:24:30 GMT")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "OK"}`))

		err = recorder.End()
		require.NoError(t, err)

		trace = recorder.Trace
	}))

	su, _ := url.Parse(server.URL)

	tcs := []struct {
		method                string
		requestBody           io.Reader
		readBody              bool
		expectedRequestTrace  string
		expectedResponseTrace string
	}{
		{
			method:                http.MethodGet,
			readBody:              false,
			expectedRequestTrace:  fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s:%s\r\nAccept-Encoding: gzip\r\nUser-Agent: Go-http-client/1.1\r\n\r\n", su.Hostname(), su.Port()),
			expectedResponseTrace: "",
		},
		{
			method:                http.MethodPost,
			requestBody:           strings.NewReader(url.Values{"Secret": []string{"Sesame"}}.Encode()),
			readBody:              false,
			expectedRequestTrace:  fmt.Sprintf("POST / HTTP/1.1\r\nHost: %s:%s\r\nAccept-Encoding: gzip\r\nContent-Length: 13\r\nUser-Agent: Go-http-client/1.1\r\n\r\nSecret=Sesame", su.Hostname(), su.Port()),
			expectedResponseTrace: "",
		},
		{
			method:                http.MethodPost,
			requestBody:           strings.NewReader(url.Values{"Secret": []string{"Sesame"}}.Encode()),
			readBody:              true,
			expectedRequestTrace:  fmt.Sprintf("POST / HTTP/1.1\r\nHost: %s:%s\r\nAccept-Encoding: gzip\r\nContent-Length: 13\r\nUser-Agent: Go-http-client/1.1\r\n\r\nSecret=Sesame", su.Hostname(), su.Port()),
			expectedResponseTrace: "",
		},
	}

	for _, tc := range tcs {
		readBody = tc.readBody
		var req *http.Request

		req, _ = httpx.NewRequest(tc.method, server.URL, tc.requestBody, nil)

		_, err := httpx.Do(http.DefaultClient, req, nil, nil)
		assert.NoError(t, err)

		assert.Equal(t, tc.expectedRequestTrace, string(trace.RequestTrace))
		assert.Equal(t, 200, trace.Response.StatusCode)
		assert.Equal(t, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nDate: Wed, 11 Apr 2018 18:24:30 GMT\r\n\r\n", string(trace.ResponseTrace))
		assert.Equal(t, `{"status": "OK"}`, string(trace.ResponseBody))
		assert.NotNil(t, trace.StartTime)
		assert.NotNil(t, trace.EndTime)
	}
}
