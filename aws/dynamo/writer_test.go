package dynamo_test

import (
	"fmt"
	"sync"
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

	wg := &sync.WaitGroup{}

	spool := dynamo.NewSpool(client, "./_test_spool", 30*time.Second)
	spool.Start(wg)

	defer spool.Delete()

	writer := dynamo.NewWriter(client, "TestWriter", 100*time.Millisecond, 10, spool)
	writer.Start(wg)

	for i := range 10 {
		rem, err := writer.Write(&ThingItem{ThingKey: ThingKey{PK: "test", SK: "item" + fmt.Sprint(i)}, Name: "Item " + fmt.Sprint(i), Count: i})
		assert.NoError(t, err)
		assert.NotZero(t, rem)
	}

	// Allow time for writes to process
	time.Sleep(200 * time.Millisecond)

	numWritten, numSpooled := writer.Stats()
	assert.Equal(t, int64(10), numWritten)
	assert.Equal(t, int64(0), numSpooled)

	// Verify all items were actually written
	dyntest.AssertCount(t, client, "TestWriter", 10)

	// Break writing by deleting the underlying table
	dyntest.Drop(t, client, "TestWriter")

	for i := range 5 {
		writer.Write(&ThingItem{ThingKey: ThingKey{PK: "test", SK: "item" + fmt.Sprint(i)}, Name: "Item " + fmt.Sprint(i), Count: i})
	}

	// Allow time for writes to fail
	time.Sleep(200 * time.Millisecond)

	// And check they were spooled
	numWritten, numSpooled = writer.Stats()
	assert.Equal(t, int64(10), numWritten)
	assert.Equal(t, int64(5), numSpooled)
	assert.Equal(t, 5, spool.Size())

	writer.Stop()
	spool.Stop()
	wg.Wait()

}
