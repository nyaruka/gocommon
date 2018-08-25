package analytics

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLibrato(t *testing.T) {
	var testRequest *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		testRequest = httptest.NewRequest(r.Method, r.URL.String(), bytes.NewBuffer(body))
		testRequest.Header = r.Header
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	// create a new collector
	wg := sync.WaitGroup{}
	collector := NewLibrato(server.URL, "username", "password", "host", 10*time.Millisecond, &wg)
	collector.Start()

	defer func() {
		collector.Stop()
		wg.Wait()
	}()

	// queue up some events
	collector.Gauge("event10", 10)
	collector.Gauge("event11", 11)
	collector.Gauge("event12", 12)

	// sleep a bit
	time.Sleep(100 * time.Millisecond)

	// our server should have been called, check the parameters
	assert.NotNil(t, testRequest)
	assert.Equal(t, "POST", testRequest.Method)

	body, err := ioutil.ReadAll(testRequest.Body)
	assert.NoError(t, err)

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)

	assert.NoError(t, err)
	assert.Equal(t, "host", response["source"])

	gauges := response["gauges"].([]interface{})

	assert.Equal(t, "event10", gauges[0].(map[string]interface{})["name"])
	assert.Equal(t, float64(12), gauges[2].(map[string]interface{})["value"])
}
