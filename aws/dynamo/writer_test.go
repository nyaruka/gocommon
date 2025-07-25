package dynamo_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/aws/dynamo"
	"github.com/nyaruka/gocommon/dates"
	"github.com/nyaruka/gocommon/uuids"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	ctx := context.Background()

	client, err := dynamo.NewClient("root", "tembatemba", "us-east-1", "http://localhost:6000")
	require.NoError(t, err)

	createTestTable(t, client, "TestWriterThings")
	tbl := dynamo.NewTable[ThingKey, ThingItem](client, "TestWriterThings")

	// Wait a bit for table to be ready
	time.Sleep(100 * time.Millisecond)

	wg := &sync.WaitGroup{}

	spool := dynamo.NewSpool[ThingItem]("./_test_spool", wg)

	uuids.SetGenerator(uuids.NewSeededGenerator(1234, dates.NewSequentialNow(time.Date(2025, 7, 25, 12, 0, 0, 0, time.UTC), time.Second)))

	defer func() {
		tbl.Delete(ctx)
		spool.Delete()
	}()

	writer := dynamo.NewWriter(ctx, tbl, 100*time.Millisecond, 10, spool, wg)

	err = writer.Start()
	assert.NoError(t, err)
	assert.DirExists(t, "./_test_spool")

	for i := range 30 {
		writer.Write(&ThingItem{
			ThingKey: ThingKey{PK: "test", SK: "item" + fmt.Sprint(i)},
			Name:     "Item " + fmt.Sprint(i),
			Count:    i,
		})
	}

	// Allow time for writes to process
	time.Sleep(200 * time.Millisecond)

	// Verify all items were written
	count, err := tbl.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 30, count)

	// Break writing by deleting the underlying table
	err = tbl.Delete(ctx)
	require.NoError(t, err)

	for i := range 30 {
		writer.Write(&ThingItem{
			ThingKey: ThingKey{PK: "test", SK: "item" + fmt.Sprint(i)},
			Name:     "Item " + fmt.Sprint(i),
			Count:    i,
		})
	}

	// Allow time for writes to process
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 30, spool.Size())
	assert.FileExists(t, "./_test_spool/01984174-5600-7000-aded-7d8b151cbd5b@25.jsonl")
	assert.FileExists(t, "./_test_spool/01984174-59e8-7000-b664-880fc7581c77@5.jsonl")

	// stop writer and its spool
	writer.Stop()
	wg.Wait()

	// start new spool to verify it can read the existing spool files
	spool = dynamo.NewSpool[ThingItem]("./_test_spool", wg)
	spool.Start()
	assert.Equal(t, 30, spool.Size())

	spool.Stop()
	wg.Wait()
}
