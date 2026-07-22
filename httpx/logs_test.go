package httpx_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/nyaruka/gocommon/httpx"
	"github.com/nyaruka/gocommon/stringsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogs(t *testing.T) {
	ctx := context.Background()

	tt := httpx.WithTraces(httpx.WithMocks(http.DefaultTransport, map[string][]*httpx.MockResponse{
		"http://temba.io/code/987654321/long-url/rwhrehreh/erhether/yreyrreyeyreureuetutrurtueyre/y": {
			httpx.NewMockResponse(400, nil, []byte("long response long response long response long response long response long response long response")),
		},
		"http://temba.io/code/987654321/": {
			httpx.NewMockResponse(200, nil, []byte(`{"value": "987654321", "secret": "43t34wf#@f3"}`)),
			httpx.NewMockResponse(400, nil, []byte("The code is 987654321, I said 987654321")),
		},
	}))

	// trace makes the given request through the tracing transport and returns the captured trace
	trace := func(req *http.Request) *httpx.Trace {
		resp, err := tt.RoundTrip(req)
		require.NoError(t, err)
		resp.Body.Close()
		traces := tt.Traces()
		return traces[len(traces)-1]
	}

	req1, err := httpx.NewRequest(
		ctx,
		"GET", "http://temba.io/code/987654321/long-url/rwhrehreh/erhether/yreyrreyeyreureuetutrurtueyre/y",
		strings.NewReader("long request long request long request long request long request long request long request long request "),
		nil,
	)
	require.NoError(t, err)
	trace1 := trace(req1)

	// check that URL and traces are truncated
	log1 := httpx.NewLog(trace1, 32, 64, nil)
	assert.Equal(t, "http://temba.io/code/98765432...", log1.URL)
	assert.Equal(t, "GET /code/987654321/long-url/rwhrehreh/erhether/yreyrreyeyreu...", log1.Request)
	assert.Equal(t, "HTTP/1.0 400 Bad Request\r\nContent-Length: 97\r\n\r\nlong response...", log1.Response)

	// but the true sizes are recorded
	assert.Equal(t, httpx.TraceSizes{Request: 288, Response: 145}, log1.Sizes)

	// if a response body was discarded (e.g. read limit exceeded), the server declared Content-Length is
	// used to record the true response size
	trace1.ResponseBody = nil
	log1 = httpx.NewLog(trace1, 32, 64, nil)
	assert.Equal(t, httpx.TraceSizes{Request: 288, Response: 145}, log1.Sizes)

	// create a request with a code we need to redact in the URL and in the header
	req2, err := httpx.NewRequest(ctx, "GET", "http://temba.io/code/987654321/", nil, map[string]string{"X-Code": "987654321"})
	require.NoError(t, err)
	trace2 := trace(req2)

	// create a request with a code we need to redact in the URL and in the request body
	req3, err := httpx.NewRequest(ctx, "GET", "http://temba.io/code/987654321/", strings.NewReader("My code is 987654321"), nil)
	require.NoError(t, err)
	trace3 := trace(req3)

	redactor := stringsx.NewRedactor("****************", "987654321", "43t34wf#@f3")

	log2 := httpx.NewLog(trace2, 2048, 10000, redactor)
	assert.Equal(t, "http://temba.io/code/****************/", log2.URL)
	assert.Equal(t, "GET /code/****************/ HTTP/1.1\r\nHost: temba.io\r\nUser-Agent: Go-http-client/1.1\r\nX-Code: ****************\r\nAccept-Encoding: gzip\r\n\r\n", log2.Request)
	assert.Equal(t, "HTTP/1.0 200 OK\r\nContent-Length: 47\r\n\r\n{\"value\": \"****************\", \"secret\": \"****************\"}", log2.Response)

	log3 := httpx.NewLog(trace3, 2048, 10000, redactor)
	assert.Equal(t, "http://temba.io/code/****************/", log3.URL)
	assert.Equal(t, "GET /code/****************/ HTTP/1.1\r\nHost: temba.io\r\nUser-Agent: Go-http-client/1.1\r\nContent-Length: 20\r\nAccept-Encoding: gzip\r\n\r\nMy code is ****************", log3.Request)
	assert.Equal(t, "HTTP/1.0 400 Bad Request\r\nContent-Length: 39\r\n\r\nThe code is ****************, I said ****************", log3.Response)
}

func TestReplaceEscapedNulls(t *testing.T) {
	assert.Equal(t, ``, httpx.ReplaceEscapedNulls(``, `?`))
	assert.Equal(t, `abcdef`, httpx.ReplaceEscapedNulls(`abc\u0000def`, ``))
	assert.Equal(t, `abc?def`, httpx.ReplaceEscapedNulls(`abc\u0000def`, `?`))
	assert.Equal(t, `�ɇ�ɇ`, httpx.ReplaceEscapedNulls(`\u0000\u0000`, `�ɇ`))
	assert.Equal(t, `abc  \\u0000 \\ \\\\u0000 def`, httpx.ReplaceEscapedNulls(`abc \u0000 \\u0000 \\\u0000 \\\\u0000 def`, ``))
	assert.Equal(t, `0000`, httpx.ReplaceEscapedNulls(`\u00000000`, ``))
}
