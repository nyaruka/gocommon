package dynamo_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/nyaruka/gocommon/aws/dynamo"
	"github.com/nyaruka/gocommon/dates"
	"github.com/nyaruka/gocommon/uuids"
	"github.com/stretchr/testify/assert"
)

func TestSpool(t *testing.T) {
	ctx := context.Background()

	uuids.SetGenerator(uuids.NewSeededGenerator(1234, dates.NewSequentialNow(time.Date(2025, 7, 25, 12, 0, 0, 0, time.UTC), time.Second)))

	client, err := dynamo.NewClient("root", "tembatemba", "us-east-1", "http://localhost:6000")
	assert.NoError(t, err)

	defer dynamo.Drop(ctx, client, "TestThings")

	createTestTable(t, client, "TestThings")

	item1, _ := dynamo.Marshal(&ThingItem{ThingKey: ThingKey{PK: "P11", SK: "SAA"}, Name: "Thing 1", Count: 123})
	item2, _ := dynamo.Marshal(&ThingItem{ThingKey: ThingKey{PK: "P22", SK: "SBB"}, Name: "Thing 2", Count: 234})
	item3, _ := dynamo.Marshal(&ThingItem{ThingKey: ThingKey{PK: "P33", SK: "SAA"}, Name: "Thing 3", Count: 345})

	wg := &sync.WaitGroup{}
	spool := dynamo.NewSpool(ctx, client, "./_test_spool", 30*time.Second, wg)

	defer spool.Delete()

	err = spool.Start()
	assert.NoError(t, err)

	err = spool.Add("TestThings", []map[string]types.AttributeValue{item1, item2})
	assert.NoError(t, err)
	err = spool.Add("TestThings", []map[string]types.AttributeValue{item3})
	assert.NoError(t, err)

	assert.Equal(t, 3, spool.Size())
	assert.FileExists(t, "./_test_spool/01984174-5600-7000-aded-7d8b151cbd5b#2@TestThings.jsonl")
	assert.FileExists(t, "./_test_spool/01984174-59e8-7000-b664-880fc7581c77#1@TestThings.jsonl")

	spool.Stop()
	wg.Wait()

	// Start new spool to verify it can read the existing spool files
	spool = dynamo.NewSpool(ctx, client, "./_test_spool", 100*time.Millisecond, wg)
	spool.Start()
	assert.Equal(t, 3, spool.Size())

	// Give spool time to try a flush
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 0, spool.Size())

	obj, err := dynamo.GetItem[ThingKey, ThingItem](ctx, client, "TestThings", ThingKey{PK: "P11", SK: "SAA"})
	assert.NoError(t, err)
	assert.Equal(t, "Thing 1", obj.Name)

	spool.Stop()
	wg.Wait()
}
