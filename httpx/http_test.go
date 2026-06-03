package httpx_test

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nyaruka/gocommon/httpx"
	"github.com/stretchr/testify/assert"
)

func newTestHTTPServer(port int) *httptest.Server {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var contentType string
		var data []byte

		cmd := r.URL.Query().Get("cmd")
		switch cmd {
		case "success":
			contentType = "text/plain; charset=utf-8"
			data = []byte(`{ "ok": "true" }`)
		case "nullchars":
			contentType = "text/plain; charset=utf-8"
			data = []byte("ab\x00cd\x00\x00")
		case "badutf8":
			w.Header().Set("Bad-Header", "\x80\x81")
			contentType = "text/plain; charset=utf-8"
			data = []byte("ab\x80\x81cd")
		case "binary":
			contentType = "application/octet-stream"
			data = make([]byte, 1000)
			for i := 0; i < 1000; i++ {
				data[i] = byte(i % 255)
			}

			w.Header().Set("Content-Length", "1000")
		}

		w.Header().Set("Date", "Wed, 11 Apr 2018 18:24:30 GMT")
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))

	if port > 0 {
		// manually create a listener for our test server so that our output is predictable
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			panic(err.Error())
		}
		server.Listener = l
	}
	server.Start()
	return server
}

func TestDetectContentType(t *testing.T) {
	tcs := []struct {
		intput       []byte
		stdlib       string // for comparison
		expectedType string
		expectedExt  string
	}{
		{nil, "text/plain; charset=utf-8", "text/plain", ".txt"},
		{[]byte(`hello`), "text/plain; charset=utf-8", "text/plain; charset=utf-8", ".txt"},
		{[]byte(`{"foo": "bar"}`), "text/plain; charset=utf-8", "application/json", ".json"},
		{[]byte{0x1f, 0x8b}, "application/octet-stream", "application/gzip", ".gz"},
		{[]byte("GIF87a"), "image/gif", "image/gif", ".gif"},
		{[]byte{0xFF, 0xD8, 0xFF}, "image/jpeg", "image/jpeg", ".jpg"},
		{[]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, "image/png", "image/png", ".png"},
		{[]byte{0xFF, 0xF1}, "text/plain; charset=utf-8", "audio/aac", ".aac"},

		// 10 random bytes
		{[]byte{0x83, 0x6f, 0x04, 0xf9, 0x1c, 0x8a, 0x72, 0xd5, 0xe9, 0xe8}, "application/octet-stream", "application/octet-stream", ""},
	}

	for _, tc := range tcs {
		actualType, actualExt := httpx.DetectContentType(tc.intput)

		assert.Equal(t, tc.stdlib, http.DetectContentType(tc.intput), "stdlib content type mismatch for input %s", string(tc.intput))
		assert.Equal(t, tc.expectedType, actualType, "content type mismatch for input %s", string(tc.intput))
		assert.Equal(t, tc.expectedExt, actualExt, "extension mismatch for input %s", string(tc.intput))
	}
}

func TestBasicAuth(t *testing.T) {
	assert.Equal(t, "Og==", httpx.BasicAuth("", ""))
	assert.Equal(t, "Ym9iOnBhc3MxMjM=", httpx.BasicAuth("bob", "pass123"))
}
