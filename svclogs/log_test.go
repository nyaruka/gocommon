package svclogs_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/httpx"
	"github.com/nyaruka/gocommon/svclogs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogs(t *testing.T) {
	// tracing is installed once on the shared client; each call collects its own traces via the context
	client := &http.Client{Transport: httpx.WithTraces(httpx.WithMocks(nil, map[string][]*httpx.MockResponse{
		"http://ivr.com/start":  {httpx.NewMockResponse(200, nil, []byte("OK"))},
		"http://ivr.com/hangup": {httpx.NewMockResponse(400, nil, []byte("Oops"))},
	}))}

	do := func(t *testing.T, method, url string, headers map[string]string) *httpx.Trace {
		t.Helper()

		ctx, traces := httpx.WithTraceCollector(context.Background())
		req, err := httpx.NewRequest(ctx, method, url, nil, headers)
		require.NoError(t, err)
		_, err = client.Do(req)
		require.NoError(t, err)

		return traces.Last()
	}

	log1 := svclogs.New("type1", nil, []string{"sesame"})
	log1.HTTP(do(t, "GET", "http://ivr.com/start", map[string]string{"Authorization": "Token sesame"}))
	log1.End()

	log2 := svclogs.New("type1", nil, []string{"sesame"})
	log2.HTTP(do(t, "GET", "http://ivr.com/hangup", nil))
	log2.Error(&svclogs.Error{Message: "oops"})
	log2.End()

	assert.NotEqual(t, log1.UUID, log2.UUID)
	assert.NotEqual(t, time.Duration(0), log1.Elapsed)

	// a 2XX response and no errors isn't an error
	assert.False(t, log1.IsError())

	// a 4XX response is, as is a recorded error
	assert.True(t, log2.IsError())

	// redaction applies to values passed at construction, in both traces and errors
	assert.NotContains(t, log1.HttpLogs[0].Request, "sesame")
	assert.Contains(t, log1.HttpLogs[0].Request, "**********")

	log3 := svclogs.New("type2", nil, []string{"sesame"})
	log3.Error(&svclogs.Error{Code: "code1", ExtCode: "ext", Message: "contains sesame seeds"})
	assert.Equal(t, "contains ********** seeds", log3.Errors[0].Message)
	assert.Equal(t, "code1", log3.Errors[0].Code)
	assert.Equal(t, "ext", log3.Errors[0].ExtCode)
	assert.True(t, log3.IsError())
}
