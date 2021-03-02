package httpx_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/nyaruka/gocommon/httpx"

	"github.com/stretchr/testify/assert"
)

func TestRecorder(t *testing.T) {
	var request *http.Request
	var trace *httpx.Trace
	var err error

	readBody := false
	saveRequest := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request = r
		recorder := httpx.NewRecorder(r, w)
		w = recorder.ResponseWriter

		if saveRequest {
			recorder.SaveRequest()
		}

		if readBody {
			ioutil.ReadAll(r.Body)
		}

		w.Header().Set("Date", "Wed, 11 Apr 2018 18:24:30 GMT")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "OK"}`))

		trace, err = recorder.End()
	}))

	su, _ := url.Parse(server.URL)

	tcs := []struct {
		ReadBody     bool
		SaveRequest  bool
		RequestTrace string
	}{
		{
			ReadBody:     false,
			SaveRequest:  false,
			RequestTrace: fmt.Sprintf("POST / HTTP/1.1\r\nHost: %s:%s\r\nAccept-Encoding: gzip\r\nContent-Length: 13\r\nUser-Agent: Go-http-client/1.1\r\n\r\nSecret=Sesame", su.Hostname(), su.Port()),
		},
		{
			ReadBody:     true,
			SaveRequest:  false,
			RequestTrace: fmt.Sprintf("POST / HTTP/1.1\r\nHost: %s:%s\r\nAccept-Encoding: gzip\r\nContent-Length: 13\r\nUser-Agent: Go-http-client/1.1\r\n\r\n", su.Hostname(), su.Port()),
		},
		{
			ReadBody:     true,
			SaveRequest:  true,
			RequestTrace: fmt.Sprintf("POST / HTTP/1.1\r\nHost: %s:%s\r\nAccept-Encoding: gzip\r\nContent-Length: 13\r\nUser-Agent: Go-http-client/1.1\r\n\r\nSecret=Sesame", su.Hostname(), su.Port()),
		},
	}

	for _, tc := range tcs {
		readBody = tc.ReadBody
		saveRequest = tc.SaveRequest

		req, _ := httpx.NewRequest("POST", server.URL, strings.NewReader(url.Values{"Secret": []string{"Sesame"}}.Encode()), nil)
		httpx.Do(http.DefaultClient, req, nil, nil)

		assert.NoError(t, err)
		assert.Equal(t, request, trace.Request)
		assert.Equal(t, tc.RequestTrace, string(trace.RequestTrace))
		assert.Equal(t, 200, trace.Response.StatusCode)
		assert.Equal(t, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nDate: Wed, 11 Apr 2018 18:24:30 GMT\r\n\r\n", string(trace.ResponseTrace))
		assert.Equal(t, `{"status": "OK"}`, string(trace.ResponseBody))
	}
}
