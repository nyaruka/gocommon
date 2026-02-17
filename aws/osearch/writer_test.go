package osearch_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/aws/osearch"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	client, err := osearch.NewClient("", "", "", "http://opensearch:9200")
	require.NoError(t, err)

	createTestIndex(t, client, "test-writer")
	defer deleteTestIndex(t, client, "test-writer")

	spool := osearch.NewSpool(client, "./_test_spool", 30*time.Second)
	err = spool.Start()
	require.NoError(t, err)

	defer spool.Delete()

	writer := osearch.NewWriter(client, "test-writer", osearch.ActionIndex, 25, 100*time.Millisecond, 10, spool)

	assert.Equal(t, client, writer.Client())
	assert.Equal(t, "test-writer", writer.Index())

	writer.Start()

	for i := range 10 {
		rem := writer.Queue([]byte(fmt.Sprintf(`{"value": %d}`, i)))
		assert.NotZero(t, rem)
	}

	// allow time for writes to process
	time.Sleep(200 * time.Millisecond)

	numWritten, numSpooled := writer.Stats()
	assert.Equal(t, int64(10), numWritten)
	assert.Equal(t, int64(0), numSpooled)

	// refresh and verify count
	refreshIndex(t, client, "test-writer")
	assertCount(t, client, "test-writer", 10)

	for i := range 5 {
		writer.Queue([]byte(fmt.Sprintf(`{"value": %d}`, i+10)))
	}

	writer.Flush()

	numWritten, numSpooled = writer.Stats()
	assert.Equal(t, int64(15), numWritten)
	assert.Equal(t, int64(0), numSpooled)

	writer.Stop()

	// simulate transport failure by creating a writer pointed at an unreachable endpoint
	badClient, err := osearch.NewClient("", "", "", "http://localhost:19999")
	require.NoError(t, err)

	badWriter := osearch.NewWriter(badClient, "test-writer", osearch.ActionIndex, 25, 100*time.Millisecond, 10, spool)
	badWriter.Start()

	for i := range 5 {
		badWriter.Queue([]byte(fmt.Sprintf(`{"value": %d}`, i+15)))
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

func createTestIndex(t *testing.T, client *opensearchapi.Client, name string) {
	t.Helper()

	_, err := client.Indices.Create(t.Context(), opensearchapi.IndicesCreateReq{
		Index: name,
	})
	require.NoError(t, err)
}

func deleteTestIndex(t *testing.T, client *opensearchapi.Client, name string) {
	t.Helper()

	_, err := client.Indices.Delete(t.Context(), opensearchapi.IndicesDeleteReq{
		Indices: []string{name},
	})
	require.NoError(t, err)
}

func refreshIndex(t *testing.T, client *opensearchapi.Client, name string) {
	t.Helper()

	_, err := client.Indices.Refresh(t.Context(), &opensearchapi.IndicesRefreshReq{
		Indices: []string{name},
	})
	require.NoError(t, err)
}

func assertCount(t *testing.T, client *opensearchapi.Client, name string, expected int) {
	t.Helper()

	resp, err := client.Indices.Count(t.Context(), &opensearchapi.IndicesCountReq{
		Indices: []string{name},
	})
	require.NoError(t, err)
	assert.Equal(t, expected, resp.Count)
}
