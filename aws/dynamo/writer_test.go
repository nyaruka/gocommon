package dynamo_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/aws/dynamo"
	"github.com/nyaruka/gocommon/aws/dynamo/dyntest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	client, err := dynamo.NewClient("root", "tembatemba", "us-east-1", "http://localhost:6000")
	require.NoError(t, err)

	createTestTable(t, client, "TestWriter")

	spool := dynamo.NewSpool(client, "./_test_spool", 30*time.Second)
	spool.Start()

	defer spool.Delete()

	writer := dynamo.NewWriter(client, "TestWriter", 100*time.Millisecond, 10, spool)
	writer.Start()

	for i := range 10 {
		rem := writer.Queue(&dynamo.Item{Key: dynamo.Key{PK: "test", SK: "item" + fmt.Sprint(i)}, OrgID: 1, Data: map[string]any{"Name": "Item " + fmt.Sprint(i), "Count": i}})
		assert.NotZero(t, rem)
	}

	// add duplicate of last item to test deduping
	_ = writer.Queue(&dynamo.Item{Key: dynamo.Key{PK: "test", SK: "item9"}, OrgID: 1, Data: map[string]any{"Name": "Item 9 v2", "Count": 9}})

	// allow time for writes to process
	time.Sleep(200 * time.Millisecond)

	numWritten, numSpooled := writer.Stats()
	assert.Equal(t, int64(10), numWritten)
	assert.Equal(t, int64(0), numSpooled)

	// verify all items were actually written
	dyntest.AssertCount(t, client, "TestWriter", 10)

	// check that last version of item9 was written
	item, err := dynamo.GetItem(t.Context(), client, "TestWriter", dynamo.Key{PK: "test", SK: "item9"})
	assert.NoError(t, err)
	assert.Equal(t, "Item 9 v2", item.Data["Name"])

	for i := range 5 {
		writer.Queue(&dynamo.Item{Key: dynamo.Key{PK: "test", SK: "item" + fmt.Sprint(i)}, OrgID: 1, Data: map[string]any{"Name": "Item " + fmt.Sprint(i), "Count": i}})
	}

	writer.Flush()

	numWritten, numSpooled = writer.Stats()
	assert.Equal(t, int64(15), numWritten)
	assert.Equal(t, int64(0), numSpooled)

	// Break writing by deleting the underlying table
	dyntest.Drop(t, client, "TestWriter")

	for i := range 5 {
		writer.Queue(&dynamo.Item{Key: dynamo.Key{PK: "test", SK: "item" + fmt.Sprint(i)}, OrgID: 1, Data: map[string]any{"Name": "Item " + fmt.Sprint(i), "Count": i}})
	}

	// Allow time for writes to fail
	time.Sleep(200 * time.Millisecond)

	// And check they were spooled
	numWritten, numSpooled = writer.Stats()
	assert.Equal(t, int64(15), numWritten)
	assert.Equal(t, int64(5), numSpooled)
	assert.Equal(t, 5, spool.Size())

	writer.Stop()
	spool.Stop()
}
