package osearch_test

import (
	"testing"

	"github.com/nyaruka/gocommon/aws/osearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBulkIndex(t *testing.T) {
	ctx := t.Context()

	client, err := osearch.NewClient("", "", "", "http://opensearch:9200")
	require.NoError(t, err)

	createTestIndex(t, client, "test-bulk")
	defer deleteTestIndex(t, client, "test-bulk")

	// empty batch is a no-op
	numWritten, retryable, err := osearch.BulkIndex(ctx, client, []*osearch.Document{})
	assert.NoError(t, err)
	assert.Equal(t, 0, numWritten)
	assert.Nil(t, retryable)

	// index some documents
	numWritten, retryable, err = osearch.BulkIndex(ctx, client, []*osearch.Document{
		{Index: "test-bulk", ID: "1", Routing: "org1", Body: []byte(`{"name": "Item 1", "count": 100}`)},
		{Index: "test-bulk", ID: "2", Routing: "org1", Body: []byte(`{"name": "Item 2", "count": 200}`)},
		{Index: "test-bulk", ID: "3", Routing: "org1", Body: []byte(`{"name": "Item 3", "count": 300}`)},
		{Index: "test-bulk", ID: "4", Routing: "org1", Body: []byte(`{"name": "Item 4", "count": 400}`)},
	})
	assert.NoError(t, err)
	assert.Equal(t, 4, numWritten)
	assert.Empty(t, retryable)

	refreshIndex(t, client, "test-bulk")
	assertCount(t, client, "test-bulk", 4)

	// test with nil batch
	numWritten, retryable, err = osearch.BulkIndex(ctx, client, nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, numWritten)
	assert.Nil(t, retryable)
}

func TestTest(t *testing.T) {
	ctx := t.Context()

	client, err := osearch.NewClient("", "", "", "http://opensearch:9200")
	require.NoError(t, err)

	err = osearch.Test(ctx, client, "test-nonexistent")
	assert.Error(t, err)

	createTestIndex(t, client, "test-exists")
	defer deleteTestIndex(t, client, "test-exists")

	err = osearch.Test(ctx, client, "test-exists")
	assert.NoError(t, err)
}
