package elastic_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/nyaruka/gocommon/elastic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	createTestIndex(t, testClient, "test-writer")
	defer deleteTestIndex(t, testClient, "test-writer")

	spool := elastic.NewSpool(testClient, "./_test_spool", 30*time.Second)
	err := spool.Start()
	require.NoError(t, err)

	defer spool.Delete()

	writer := elastic.NewWriter(testClient, 25, 100*time.Millisecond, 10, spool)

	assert.Equal(t, testClient, writer.Client())

	writer.Start()

	for i := range 10 {
		rem := writer.Queue(&elastic.Document{Index: "test-writer", ID: fmt.Sprintf("%d", i), Routing: "test", Body: []byte(fmt.Sprintf(`{"value": %d}`, i))})
		assert.NotZero(t, rem)
	}

	// allow time for writes to process
	time.Sleep(200 * time.Millisecond)

	numWritten, numSpooled := writer.Stats()
	assert.Equal(t, int64(10), numWritten)
	assert.Equal(t, int64(0), numSpooled)

	// refresh and verify count
	refreshIndex(t, testClient, "test-writer")
	assertCount(t, testClient, "test-writer", 10)

	for i := range 5 {
		writer.Queue(&elastic.Document{Index: "test-writer", ID: fmt.Sprintf("%d", i+10), Routing: "test", Body: []byte(fmt.Sprintf(`{"value": %d}`, i+10))})
	}

	writer.Flush()

	numWritten, numSpooled = writer.Stats()
	assert.Equal(t, int64(15), numWritten)
	assert.Equal(t, int64(0), numSpooled)

	writer.Stop()

	// simulate transport failure by creating a writer pointed at an unreachable endpoint
	badClient, err := elastic.NewClient("http://localhost:19999")
	require.NoError(t, err)

	badWriter := elastic.NewWriter(badClient, 25, 100*time.Millisecond, 10, spool)
	badWriter.Start()

	for i := range 5 {
		badWriter.Queue(&elastic.Document{Index: "test-writer", ID: fmt.Sprintf("%d", i+15), Routing: "test", Body: []byte(fmt.Sprintf(`{"value": %d}`, i+15))})
	}

	// allow time for writes to fail
	time.Sleep(200 * time.Millisecond)

	// and check they were spooled
	numWritten, numSpooled = badWriter.Stats()
	assert.Equal(t, int64(0), numWritten)
	assert.Equal(t, int64(5), numSpooled)
	assert.Equal(t, 5, spool.Size())

	badWriter.Stop()
	spool.Stop()
}

func createTestIndex(t *testing.T, client *elasticsearch.Client, name string) {
	t.Helper()

	resp, err := client.Indices.Create(name, client.Indices.Create.WithContext(t.Context()))
	require.NoError(t, err)
	resp.Body.Close()
	require.False(t, resp.IsError(), "failed to create index %s: %s", name, resp.String())
}

func createTestIndexStrict(t *testing.T, client *elasticsearch.Client, name string) {
	t.Helper()

	resp, err := client.Indices.Create(name,
		client.Indices.Create.WithContext(t.Context()),
		client.Indices.Create.WithBody(strings.NewReader(`{"mappings": {"dynamic": "strict", "properties": {"name": {"type": "text"}}}}`)),
	)
	require.NoError(t, err)
	resp.Body.Close()
	require.False(t, resp.IsError(), "failed to create strict index %s: %s", name, resp.String())
}

func deleteTestIndex(t *testing.T, client *elasticsearch.Client, name string) {
	t.Helper()

	resp, err := client.Indices.Delete([]string{name}, client.Indices.Delete.WithContext(t.Context()))
	require.NoError(t, err)
	resp.Body.Close()
}

func refreshIndex(t *testing.T, client *elasticsearch.Client, name string) {
	t.Helper()

	resp, err := client.Indices.Refresh(client.Indices.Refresh.WithIndex(name), client.Indices.Refresh.WithContext(t.Context()))
	require.NoError(t, err)
	resp.Body.Close()
}

func assertCount(t *testing.T, client *elasticsearch.Client, name string, expected int) {
	t.Helper()

	resp, err := client.Count(client.Count.WithIndex(name), client.Count.WithContext(t.Context()))
	require.NoError(t, err)
	defer resp.Body.Close()

	var result struct {
		Count int `json:"count"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, expected, result.Count)
}
