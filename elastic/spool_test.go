package elastic_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/dates"
	"github.com/nyaruka/gocommon/elastic"
	"github.com/nyaruka/gocommon/uuids"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpool(t *testing.T) {
	uuids.SetGenerator(uuids.NewSeededGenerator(1234, dates.NewSequentialNow(time.Date(2025, 7, 25, 12, 0, 0, 0, time.UTC), time.Second)))

	createTestIndex(t, testClient, "test-spool")
	defer deleteTestIndex(t, testClient, "test-spool")

	spool := elastic.NewSpool(testClient, "./_test_spool", 30*time.Second)

	defer spool.Delete()

	err := spool.Start()
	require.NoError(t, err)

	err = spool.Add([]*elastic.Document{
		{Index: "test-spool", ID: "1", Routing: "test", Body: []byte(`{"name": "Thing 1", "count": 123}`)},
		{Index: "test-spool", ID: "2", Routing: "test", Body: []byte(`{"name": "Thing 2", "count": 234}`)},
	})
	require.NoError(t, err)

	err = spool.Add([]*elastic.Document{
		{Index: "test-spool", ID: "3", Routing: "test", Body: []byte(`{"name": "Thing 3", "count": 345}`)},
	})
	require.NoError(t, err)

	assert.Equal(t, 3, spool.Size())
	assert.FileExists(t, "./_test_spool/01984174-5600-7000-8e0f-6b2abe4360d8#2.jsonl")
	assert.FileExists(t, "./_test_spool/01984174-59e8-7000-9a98-cfcce3019710#1.jsonl")

	spool.Stop()

	// start new spool to verify it can read existing spool files
	spool = elastic.NewSpool(testClient, "./_test_spool", 100*time.Millisecond)
	err = spool.Start()
	require.NoError(t, err)

	assert.Equal(t, 3, spool.Size())

	// give spool time to try a flush
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 0, spool.Size())

	// refresh and verify all items were indexed
	refreshIndex(t, testClient, "test-spool")
	assertCount(t, testClient, "test-spool", 3)

	spool.Stop()
}

func TestSpoolStartDirectoryErrors(t *testing.T) {
	// a file in place of the directory means it can't be created
	notADir := filepath.Join(t.TempDir(), "spool")
	require.NoError(t, os.WriteFile(notADir, []byte("!"), 0644))

	spool := elastic.NewSpool(nil, notADir, 30*time.Second)
	err := spool.Start()
	assert.ErrorContains(t, err, "error creating spool directory")

	// an existing but unwritable directory should fail the writability probe.. but skip if running as
	// root because then permission bits are ignored
	if os.Geteuid() == 0 {
		t.Skip("running as root so can't test unwritable directory")
	}

	unwritable := filepath.Join(t.TempDir(), "spool")
	require.NoError(t, os.Mkdir(unwritable, 0555))

	spool = elastic.NewSpool(nil, unwritable, 30*time.Second)
	err = spool.Start()
	assert.ErrorContains(t, err, "is not writable")
}
