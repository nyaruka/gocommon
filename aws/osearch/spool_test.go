package osearch_test

import (
	"testing"
	"time"

	"github.com/nyaruka/gocommon/aws/osearch"
	"github.com/nyaruka/gocommon/dates"
	"github.com/nyaruka/gocommon/uuids"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpool(t *testing.T) {
	uuids.SetGenerator(uuids.NewSeededGenerator(1234, dates.NewSequentialNow(time.Date(2025, 7, 25, 12, 0, 0, 0, time.UTC), time.Second)))

	client, err := osearch.NewClient("", "", "", "http://opensearch:9200")
	require.NoError(t, err)

	createTestIndex(t, client, "test-spool")
	defer deleteTestIndex(t, client, "test-spool")

	spool := osearch.NewSpool(client, "./_test_spool", 30*time.Second)

	defer spool.Delete()

	err = spool.Start()
	require.NoError(t, err)

	err = spool.Add("test-spool", osearch.ActionIndex, [][]byte{
		[]byte(`{"name": "Thing 1", "count": 123}`),
		[]byte(`{"name": "Thing 2", "count": 234}`),
	})
	require.NoError(t, err)

	err = spool.Add("test-spool", osearch.ActionIndex, [][]byte{
		[]byte(`{"name": "Thing 3", "count": 345}`),
	})
	require.NoError(t, err)

	assert.Equal(t, 3, spool.Size())
	assert.FileExists(t, "./_test_spool/01984174-5600-7000-aded-7d8b151cbd5b#2@index@test-spool.jsonl")
	assert.FileExists(t, "./_test_spool/01984174-59e8-7000-b664-880fc7581c77#1@index@test-spool.jsonl")

	spool.Stop()

	// start new spool to verify it can read existing spool files
	spool = osearch.NewSpool(client, "./_test_spool", 100*time.Millisecond)
	err = spool.Start()
	require.NoError(t, err)

	assert.Equal(t, 3, spool.Size())

	// give spool time to try a flush
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 0, spool.Size())

	// refresh and verify all items were indexed
	refreshIndex(t, client, "test-spool")
	assertCount(t, client, "test-spool", 3)

	spool.Stop()
}
