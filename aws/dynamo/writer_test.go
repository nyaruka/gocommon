package dynamo_test

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/aws/dynamo"
	"github.com/nyaruka/gocommon/aws/dynamo/dyntest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Thing struct {
	ID   int
	Name string
}

func (t *Thing) MarshalDynamo() (*dynamo.Item, error) {
	return &dynamo.Item{
		Key:   dynamo.Key{PK: "test", SK: fmt.Sprintf("item%d", t.ID)},
		OrgID: 1,
		Data:  map[string]any{"Name": t.Name},
	}, nil
}

func TestWriter(t *testing.T) {
	client, err := dynamo.NewClient(t.Context(), "http://dynamodb:8000")
	require.NoError(t, err)

	createTestTable(t, client, "TestWriter")

	spool := dynamo.NewSpool(client, filepath.Join(t.TempDir(), "spool"), 30*time.Second)
	require.NoError(t, spool.Start())

	// long max age because we flush explicitly.. flushing on max age is covered by the syncx batcher tests
	writer := dynamo.NewWriter(client, "TestWriter", time.Minute, 10, spool)

	assert.Equal(t, client, writer.Client())
	assert.Equal(t, "TestWriter", writer.Table())

	writer.Start()

	for i := range 10 {
		rem, err := writer.Queue(&Thing{ID: i, Name: fmt.Sprintf("Item %d", i)})
		assert.NoError(t, err)
		assert.NotZero(t, rem)
	}

	// add duplicate of last item to test deduping
	_, err = writer.Queue(&Thing{ID: 9, Name: "Item 9 v2"})
	assert.NoError(t, err)

	writer.Flush()

	numWritten, numSpooled := writer.Stats()
	assert.Equal(t, int64(10), numWritten)
	assert.Equal(t, int64(0), numSpooled)

	// verify all items were actually written
	dyntest.AssertCount(t, client, "TestWriter", 10)

	// check that last version of item9 was written
	item, err := dynamo.GetItem(t.Context(), client, "TestWriter", dynamo.Key{PK: "test", SK: "item9"})
	require.NoError(t, err)
	require.NotNil(t, item)
	assert.Equal(t, "Item 9 v2", item.Data["Name"])

	for i := range 5 {
		writer.Queue(&Thing{ID: i + 10, Name: fmt.Sprintf("Item %d", i+10)})
	}

	writer.Flush()

	numWritten, numSpooled = writer.Stats()
	assert.Equal(t, int64(15), numWritten)
	assert.Equal(t, int64(0), numSpooled)

	// break writing by deleting the underlying table
	dyntest.Drop(t, client, "TestWriter")

	for i := range 5 {
		writer.Queue(&Thing{ID: i + 15, Name: fmt.Sprintf("Item %d", i+15)})
	}

	writer.Flush()

	// and check they were spooled
	numWritten, numSpooled = writer.Stats()
	assert.Equal(t, int64(15), numWritten)
	assert.Equal(t, int64(5), numSpooled)
	assert.Equal(t, 5, spool.Size())

	writer.Stop()
	spool.Stop()
}

func TestWriterDeletes(t *testing.T) {
	ctx := t.Context()

	client, err := dynamo.NewClient(ctx, "http://dynamodb:8000")
	require.NoError(t, err)

	createTestTable(t, client, "TestWriterDeletes")

	spool := dynamo.NewSpool(client, filepath.Join(t.TempDir(), "spool"), 30*time.Second)
	require.NoError(t, spool.Start())

	writer := dynamo.NewWriter(client, "TestWriterDeletes", time.Minute, 10, spool)
	writer.Start()

	for i := range 5 {
		_, err := writer.Queue(&Thing{ID: i, Name: fmt.Sprintf("Item %d", i)})
		assert.NoError(t, err)
	}

	writer.Flush()

	dyntest.AssertCount(t, client, "TestWriterDeletes", 5)

	// queue deletes of two existing items
	rem, err := writer.QueueDelete(dynamo.Key{PK: "test", SK: "item0"})
	assert.NoError(t, err)
	assert.NotZero(t, rem)
	_, err = writer.QueueDelete(dynamo.Key{PK: "test", SK: "item1"})
	assert.NoError(t, err)

	// and in the same batch, a put then a delete of the same key (delete queued later wins)...
	_, err = writer.Queue(&Thing{ID: 2, Name: "Item 2 v2"})
	assert.NoError(t, err)
	_, err = writer.QueueDelete(dynamo.Key{PK: "test", SK: "item2"})
	assert.NoError(t, err)

	// ... and a delete then a put of the same key (put queued later wins)
	_, err = writer.QueueDelete(dynamo.Key{PK: "test", SK: "item3"})
	assert.NoError(t, err)
	_, err = writer.Queue(&Thing{ID: 3, Name: "Item 3 v2"})
	assert.NoError(t, err)

	writer.Flush()

	numWritten, numSpooled := writer.Stats()
	assert.Equal(t, int64(9), numWritten) // 5 puts + 3 deletes + 1 put after deduping
	assert.Equal(t, int64(0), numSpooled)

	dyntest.AssertCount(t, client, "TestWriterDeletes", 2)

	item, err := dynamo.GetItem(ctx, client, "TestWriterDeletes", dynamo.Key{PK: "test", SK: "item2"})
	require.NoError(t, err)
	assert.Nil(t, item)

	item, err = dynamo.GetItem(ctx, client, "TestWriterDeletes", dynamo.Key{PK: "test", SK: "item3"})
	require.NoError(t, err)
	require.NotNil(t, item)
	assert.Equal(t, "Item 3 v2", item.Data["Name"])

	// break writing by dropping the table and check that a failed delete is spooled
	dyntest.Drop(t, client, "TestWriterDeletes")

	_, err = writer.QueueDelete(dynamo.Key{PK: "test", SK: "item4"})
	assert.NoError(t, err)

	writer.Flush()

	numWritten, numSpooled = writer.Stats()
	assert.Equal(t, int64(9), numWritten)
	assert.Equal(t, int64(1), numSpooled)
	assert.Equal(t, 1, spool.Size())

	writer.Stop()

	// recreate the table with item4 still in it and check that flushing the spool replays the delete
	createTestTable(t, client, "TestWriterDeletes")
	defer dyntest.Drop(t, client, "TestWriterDeletes")

	err = dynamo.PutItem(ctx, client, "TestWriterDeletes", &dynamo.Item{Key: dynamo.Key{PK: "test", SK: "item4"}, OrgID: 1, Data: map[string]any{"Name": "Item 4"}})
	require.NoError(t, err)

	require.NoError(t, spool.Flush())
	assert.Equal(t, 0, spool.Size())

	item, err = dynamo.GetItem(ctx, client, "TestWriterDeletes", dynamo.Key{PK: "test", SK: "item4"})
	require.NoError(t, err)
	assert.Nil(t, item)

	spool.Stop()
}
