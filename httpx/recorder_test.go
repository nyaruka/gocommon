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
	// set by handler
	var readBody []byte
	var trace *httpx.Trace

	// set by each test case
	var testReconstruct bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder, err := httpx.NewRecorder(r, w, testReconstruct)
		require.NoError(t, err)
		w = recorder.ResponseWriter

		// read body to ensure that we still dump it even if it's been read
		readBody, _ = io.ReadAll(r.Body)

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
		url                   string
		headers               map[string]string
		requestBody           io.Reader
		reconstruct           bool
		expectedReadBody      string
		expectedRequestURL    string
		expectedRequestTrace  string
		expectedResponseTrace string
	}{
		{ // 0
			method:               http.MethodGet,
			url:                  server.URL,
			headers:              map[string]string{"X-Nyaruka-Test": "true"},
			reconstruct:          false,
			expectedReadBody:     "",
			expectedRequestURL:   "/",
			expectedRequestTrace: fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s:%s\r\nAccept-Encoding: gzip\r\nUser-Agent: Go-http-client/1.1\r\nX-Nyaruka-Test: true\r\n\r\n", su.Hostname(), su.Port()),
		},
		{ // 1
			method:               http.MethodGet,
			url:                  server.URL,
			headers:              map[string]string{"Host": "textit.com"}, // will be overwritten by client request
			reconstruct:          true,
			expectedReadBody:     "",
			expectedRequestURL:   fmt.Sprintf("http://%s:%s/", su.Hostname(), su.Port()),
			expectedRequestTrace: fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s:%s\r\nAccept-Encoding: gzip\r\nUser-Agent: Go-http-client/1.1\r\n\r\n", su.Hostname(), su.Port()),
		},
		{ // 2
			method:               http.MethodPost,
			url:                  server.URL + "/path/test.json?q=1",
			headers:              map[string]string{},
			requestBody:          strings.NewReader(url.Values{"Secret": []string{"Sesame"}}.Encode()),
			reconstruct:          false,
			expectedReadBody:     "Secret=Sesame",
			expectedRequestURL:   "/path/test.json?q=1",
			expectedRequestTrace: fmt.Sprintf("POST /path/test.json?q=1 HTTP/1.1\r\nHost: %s:%s\r\nAccept-Encoding: gzip\r\nContent-Length: 13\r\nUser-Agent: Go-http-client/1.1\r\n\r\nSecret=Sesame", su.Hostname(), su.Port()),
		},
		{ // 3
			method:               http.MethodPost,
			url:                  server.URL + "/path/test.json?q=1",
			headers:              map[string]string{"X-Forwarded-Path": "/original/path.json?q=1"},
			requestBody:          strings.NewReader(url.Values{"Secret": []string{"Sesame"}}.Encode()),
			reconstruct:          false,
			expectedReadBody:     "Secret=Sesame",
			expectedRequestURL:   "/path/test.json?q=1",
			expectedRequestTrace: fmt.Sprintf("POST /path/test.json?q=1 HTTP/1.1\r\nHost: %s:%s\r\nAccept-Encoding: gzip\r\nContent-Length: 13\r\nUser-Agent: Go-http-client/1.1\r\nX-Forwarded-Path: /original/path.json?q=1\r\n\r\nSecret=Sesame", su.Hostname(), su.Port()),
		},
		{ // 4
			method:               http.MethodPost,
			url:                  server.URL + "/path/test.json?q=1",
			headers:              map[string]string{"X-Forwarded-Path": "/original/path.json?z=1"},
			requestBody:          strings.NewReader(url.Values{"Secret": []string{"Sesame"}}.Encode()),
			reconstruct:          true,
			expectedReadBody:     "Secret=Sesame",
			expectedRequestURL:   fmt.Sprintf("http://%s:%s/original/path.json?z=1", su.Hostname(), su.Port()),
			expectedRequestTrace: fmt.Sprintf("POST /original/path.json?z=1 HTTP/1.1\r\nHost: %s:%s\r\nAccept-Encoding: gzip\r\nContent-Length: 13\r\nUser-Agent: Go-http-client/1.1\r\n\r\nSecret=Sesame", su.Hostname(), su.Port()),
		},
		{ // 5
			method:               http.MethodPost,
			url:                  server.URL + "/path/test.json?q=1",
			headers:              map[string]string{"X-Forwarded-Proto": "https", "X-Forwarded-Host": "textit.in"},
			reconstruct:          true,
			expectedReadBody:     "",
			expectedRequestURL:   "https://textit.in/path/test.json?q=1",
			expectedRequestTrace: fmt.Sprintf("POST /path/test.json?q=1 HTTP/1.1\r\nHost: %s:%s\r\nAccept-Encoding: gzip\r\nContent-Length: 0\r\nUser-Agent: Go-http-client/1.1\r\n\r\n", su.Hostname(), su.Port()),
		},
	}

	for i, tc := range tcs {
		testReconstruct = tc.reconstruct

		req, err := httpx.NewRequest(tc.method, tc.url, tc.requestBody, tc.headers)
		require.NoError(t, err)

		_, err = httpx.Do(http.DefaultClient, req, nil, nil)
		assert.NoError(t, err)

		assert.Equal(t, tc.expectedReadBody, string(readBody), "read body mismatch in test case %d", i)
		assert.Equal(t, tc.expectedRequestURL, trace.Request.URL.String(), "url mismatch in test case %d", i)
		assert.Equal(t, tc.expectedRequestTrace, string(trace.RequestTrace), "response trace mismatch in test case %d", i)
		assert.Equal(t, 200, trace.Response.StatusCode)
		assert.Equal(t, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nDate: Wed, 11 Apr 2018 18:24:30 GMT\r\n\r\n", string(trace.ResponseTrace))
		assert.Equal(t, `{"status": "OK"}`, string(trace.ResponseBody))
		assert.NotNil(t, trace.StartTime)
		assert.NotNil(t, trace.EndTime)
	}
}
