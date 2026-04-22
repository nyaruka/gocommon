package elastic_test

import (
	"testing"

	"github.com/nyaruka/gocommon/elastic"
	"github.com/stretchr/testify/assert"
)

func TestBulk(t *testing.T) {
	ctx := t.Context()

	createTestIndex(t, testClient, "test-bulk")
	defer deleteTestIndex(t, testClient, "test-bulk")

	// empty batch is a no-op
	numWritten, retryable, err := elastic.Bulk(ctx, testClient, []*elastic.Document{})
	assert.NoError(t, err)
	assert.Equal(t, 0, numWritten)
	assert.Nil(t, retryable)

	// index some documents (empty action defaults to index)
	numWritten, retryable, err = elastic.Bulk(ctx, testClient, []*elastic.Document{
		{Index: "test-bulk", ID: "1", Routing: "org1", Body: []byte(`{"name": "Item 1", "count": 100}`)},
		{Index: "test-bulk", ID: "2", Routing: "org1", Body: []byte(`{"name": "Item 2", "count": 200}`)},
		{Action: elastic.ActionIndex, Index: "test-bulk", ID: "3", Routing: "org1", Body: []byte(`{"name": "Item 3", "count": 300}`)},
		{Action: elastic.ActionIndex, Index: "test-bulk", ID: "4", Routing: "org1", Body: []byte(`{"name": "Item 4", "count": 400}`)},
	})
	assert.NoError(t, err)
	assert.Equal(t, 4, numWritten)
	assert.Empty(t, retryable)

	refreshIndex(t, testClient, "test-bulk")
	assertCount(t, testClient, "test-bulk", 4)

	// delete some documents
	numWritten, retryable, err = elastic.Bulk(ctx, testClient, []*elastic.Document{
		{Action: elastic.ActionDelete, Index: "test-bulk", ID: "1", Routing: "org1"},
		{Action: elastic.ActionDelete, Index: "test-bulk", ID: "2", Routing: "org1"},
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, numWritten)
	assert.Empty(t, retryable)

	refreshIndex(t, testClient, "test-bulk")
	assertCount(t, testClient, "test-bulk", 2)

	// deleting a missing document is idempotent and not retryable
	numWritten, retryable, err = elastic.Bulk(ctx, testClient, []*elastic.Document{
		{Action: elastic.ActionDelete, Index: "test-bulk", ID: "1", Routing: "org1"}, // already deleted
		{Action: elastic.ActionDelete, Index: "test-bulk", ID: "3", Routing: "org1"}, // exists
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, numWritten)
	assert.Empty(t, retryable)

	refreshIndex(t, testClient, "test-bulk")
	assertCount(t, testClient, "test-bulk", 1)

	// mixed index and delete in one batch
	numWritten, retryable, err = elastic.Bulk(ctx, testClient, []*elastic.Document{
		{Action: elastic.ActionIndex, Index: "test-bulk", ID: "5", Routing: "org1", Body: []byte(`{"name": "Item 5", "count": 500}`)},
		{Action: elastic.ActionDelete, Index: "test-bulk", ID: "4", Routing: "org1"},
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, numWritten)
	assert.Empty(t, retryable)

	refreshIndex(t, testClient, "test-bulk")
	assertCount(t, testClient, "test-bulk", 1)

	// index with external versioning
	numWritten, retryable, err = elastic.Bulk(ctx, testClient, []*elastic.Document{
		{Index: "test-bulk", ID: "10", Routing: "org1", Version: 5, Body: []byte(`{"name": "Versioned 1", "count": 1000}`)},
		{Index: "test-bulk", ID: "11", Routing: "org1", Version: 3, Body: []byte(`{"name": "Versioned 2", "count": 1100}`)},
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, numWritten)
	assert.Empty(t, retryable)

	refreshIndex(t, testClient, "test-bulk")
	assertCount(t, testClient, "test-bulk", 3)

	// re-index with same or older version should get 409 conflicts (ignored)
	numWritten, retryable, err = elastic.Bulk(ctx, testClient, []*elastic.Document{
		{Index: "test-bulk", ID: "10", Routing: "org1", Version: 3, Body: []byte(`{"name": "Versioned 1 old", "count": 999}`)},  // older version
		{Index: "test-bulk", ID: "11", Routing: "org1", Version: 3, Body: []byte(`{"name": "Versioned 2 same", "count": 999}`)}, // same version
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, numWritten)
	assert.Empty(t, retryable) // 409s are not retryable

	// re-index with newer version should succeed
	numWritten, retryable, err = elastic.Bulk(ctx, testClient, []*elastic.Document{
		{Index: "test-bulk", ID: "10", Routing: "org1", Version: 10, Body: []byte(`{"name": "Versioned 1 new", "count": 2000}`)},
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, numWritten)
	assert.Empty(t, retryable)

	// permanent failures (e.g. strict mapping violation) are also returned as unprocessed so they can be spooled
	createTestIndexStrict(t, testClient, "test-strict")
	defer deleteTestIndex(t, testClient, "test-strict")

	numWritten, retryable, err = elastic.Bulk(ctx, testClient, []*elastic.Document{
		{Index: "test-strict", ID: "1", Routing: "org1", Body: []byte(`{"name": "Item 1"}`)},               // ok
		{Index: "test-strict", ID: "2", Routing: "org1", Body: []byte(`{"name": "Item 2", "extra": true}`)}, // strict mapping violation
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, numWritten)
	assert.Len(t, retryable, 1)
	assert.Equal(t, "2", retryable[0].ID)

	// test with nil batch
	numWritten, retryable, err = elastic.Bulk(ctx, testClient, nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, numWritten)
	assert.Nil(t, retryable)
}

func TestTest(t *testing.T) {
	ctx := t.Context()

	err := elastic.Test(ctx, testClient, "test-nonexistent")
	assert.Error(t, err)

	createTestIndex(t, testClient, "test-exists")
	defer deleteTestIndex(t, testClient, "test-exists")

	err = elastic.Test(ctx, testClient, "test-exists")
	assert.NoError(t, err)
}
