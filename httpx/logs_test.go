package httpx_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/nyaruka/gocommon/httpx"
	"github.com/nyaruka/gocommon/stringsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogs(t *testing.T) {
	defer httpx.SetRequestor(httpx.DefaultRequestor)

	httpx.SetRequestor(httpx.NewMockRequestor(map[string][]*httpx.MockResponse{
		"http://temba.io/code/987654321/long-url/rwhrehreh/erhether/yreyrreyeyreureuetutrurtueyre/y": {
			httpx.NewMockResponse(400, nil, []byte("long response long response long response long response long response long response long response")),
		},
		"http://temba.io/code/987654321/": {
			httpx.NewMockResponse(200, nil, []byte(`{"value": "987654321", "secret": "43t34wf#@f3"}`)),
			httpx.NewMockResponse(400, nil, []byte("The code is 987654321, I said 987654321")),
		},
	}))

	req1, err := httpx.NewRequest(
		"GET", "http://temba.io/code/987654321/long-url/rwhrehreh/erhether/yreyrreyeyreureuetutrurtueyre/y",
		strings.NewReader("long request long request long request long request long request long request long request long request "),
		nil,
	)
	require.NoError(t, err)
	trace1, err := httpx.DoTrace(http.DefaultClient, req1, nil, nil, -1)
	require.NoError(t, err)

	// check that URL and traces are truncated
	log1 := httpx.NewLog(trace1, 32, 64, nil)
	assert.Equal(t, "http://temba.io/code/98765432...", log1.URL)
	assert.Equal(t, "GET /code/987654321/long-url/rwhrehreh/erhether/yreyrreyeyreu...", log1.Request)
	assert.Equal(t, "HTTP/1.0 400 Bad Request\r\nContent-Length: 97\r\n\r\nlong response...", log1.Response)

	// create a request with a code we need to redact in the URL and in the header
	req2, err := httpx.NewRequest("GET", "http://temba.io/code/987654321/", nil, map[string]string{"X-Code": "987654321"})
	require.NoError(t, err)
	trace2, err := httpx.DoTrace(http.DefaultClient, req2, nil, nil, -1)
	require.NoError(t, err)

	// create a request with a code we need to redact in the URL and in the request body
	req3, err := httpx.NewRequest("GET", "http://temba.io/code/987654321/", strings.NewReader("My code is 987654321"), nil)
	require.NoError(t, err)
	trace3, err := httpx.DoTrace(http.DefaultClient, req3, nil, nil, -1)
	require.NoError(t, err)

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
	assert.Equal(t, []byte(nil), httpx.ReplaceEscapedNulls(nil, []byte(`?`)))
	assert.Equal(t, []byte(`abcdef`), httpx.ReplaceEscapedNulls([]byte(`abc\u0000def`), nil))
	assert.Equal(t, []byte(`abc?def`), httpx.ReplaceEscapedNulls([]byte(`abc\u0000def`), []byte(`?`)))
	assert.Equal(t, []byte(`�ɇ�ɇ`), httpx.ReplaceEscapedNulls([]byte(`\u0000\u0000`), []byte(`�ɇ`)))
	assert.Equal(t, []byte(`abc  \\u0000 \\ \\\\u0000 def`), httpx.ReplaceEscapedNulls([]byte(`abc \u0000 \\u0000 \\\u0000 \\\\u0000 def`), nil))
	assert.Equal(t, []byte(`0000`), httpx.ReplaceEscapedNulls([]byte(`\u00000000`), nil))
}
