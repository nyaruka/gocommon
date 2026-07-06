package dynamo_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/nyaruka/gocommon/aws/dynamo"
	"github.com/nyaruka/gocommon/aws/dynamo/dyntest"
	"github.com/nyaruka/gocommon/dates"
	"github.com/nyaruka/gocommon/uuids"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpool(t *testing.T) {
	ctx := t.Context()

	uuids.SetGenerator(uuids.NewSeededGenerator(1234, dates.NewSequentialNow(time.Date(2025, 7, 25, 12, 0, 0, 0, time.UTC), time.Second)))

	client, err := dynamo.NewClient(ctx, "http://localstack:4566")
	assert.NoError(t, err)

	defer dyntest.Drop(t, client, "TestSpool")

	createTestTable(t, client, "TestSpool")

	item1, _ := attributevalue.MarshalMap(&dynamo.Item{Key: dynamo.Key{PK: "P11", SK: "SAA"}, OrgID: 1, Data: map[string]any{"name": "Thing 1", "count": 123}})
	item2, _ := attributevalue.MarshalMap(&dynamo.Item{Key: dynamo.Key{PK: "P22", SK: "SBB"}, OrgID: 1, Data: map[string]any{"name": "Thing 2", "count": 234}})
	item3, _ := attributevalue.MarshalMap(&dynamo.Item{Key: dynamo.Key{PK: "P33", SK: "SAA"}, OrgID: 1, Data: map[string]any{"name": "Thing 3", "count": 345}})

	spool := dynamo.NewSpool(client, "./_test_spool", 30*time.Second)

	defer spool.Delete()

	err = spool.Start()
	assert.NoError(t, err)

	err = spool.Add("TestSpool", []map[string]types.AttributeValue{item1, item2})
	assert.NoError(t, err)
	err = spool.Add("TestSpool", []map[string]types.AttributeValue{item3})
	assert.NoError(t, err)

	assert.Equal(t, 3, spool.Size())
	assert.FileExists(t, "./_test_spool/01984174-5600-7000-aded-7d8b151cbd5b#2@TestSpool.jsonl")
	assert.FileExists(t, "./_test_spool/01984174-59e8-7000-b664-880fc7581c77#1@TestSpool.jsonl")

	spool.Stop()

	// Start new spool to verify it can read the existing spool files
	spool = dynamo.NewSpool(client, "./_test_spool", 100*time.Millisecond)
	spool.Start()
	assert.Equal(t, 3, spool.Size())

	// Give spool time to try a flush
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 0, spool.Size())

	obj, err := dynamo.GetItem(ctx, client, "TestSpool", dynamo.Key{PK: "P11", SK: "SAA"})
	assert.NoError(t, err)
	assert.Equal(t, "Thing 1", obj.Data["name"])

	spool.Stop()
}

func TestSpoolStartDirectoryErrors(t *testing.T) {
	// a file in place of the directory means it can't be created
	notADir := filepath.Join(t.TempDir(), "spool")
	require.NoError(t, os.WriteFile(notADir, []byte("!"), 0644))

	spool := dynamo.NewSpool(nil, notADir, 30*time.Second)
	err := spool.Start()
	assert.ErrorContains(t, err, "error creating spool directory")

	// an existing but unwritable directory should fail the writability probe.. but skip if running as
	// root because then permission bits are ignored
	if os.Geteuid() == 0 {
		t.Skip("running as root so can't test unwritable directory")
	}

	unwritable := filepath.Join(t.TempDir(), "spool")
	require.NoError(t, os.Mkdir(unwritable, 0555))

	spool = dynamo.NewSpool(nil, unwritable, 30*time.Second)
	err = spool.Start()
	assert.ErrorContains(t, err, "is not writable")
}
