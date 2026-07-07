package spool_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/dates"
	"github.com/nyaruka/gocommon/spool"
	"github.com/nyaruka/gocommon/uuids"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type thing struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// flusher is a controllable FlushFunc for testing
type flusher struct {
	mu      sync.Mutex
	batches [][]*thing
	failing map[string]bool // names of things which should fail to flush
	err     error           // error to return for whole batches
}

func (f *flusher) flush(ctx context.Context, batch []*thing) ([]*thing, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.err != nil {
		return nil, f.err
	}

	f.batches = append(f.batches, batch)

	var failed []*thing
	for _, t := range batch {
		if f.failing[t.Name] {
			failed = append(failed, t)
		}
	}
	return failed, nil
}

func (f *flusher) numBatches() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.batches)
}

func TestSpool(t *testing.T) {
	uuids.SetGenerator(uuids.NewSeededGenerator(1234, dates.NewSequentialNow(time.Date(2025, 7, 25, 12, 0, 0, 0, time.UTC), time.Second)))
	defer uuids.SetGenerator(uuids.DefaultGenerator)

	dir := filepath.Join(t.TempDir(), "spool")
	fl := &flusher{failing: map[string]bool{}}

	s := spool.New(dir, time.Hour, spool.MarshalJSON[*thing], spool.UnmarshalJSON[*thing], fl.flush)
	require.NoError(t, s.Start())
	defer s.Stop()

	require.NoError(t, s.Add([]*thing{{Name: "Thing 1", Count: 123}, {Name: "Thing 2", Count: 234}}))
	require.NoError(t, s.Add([]*thing{{Name: "Thing 3", Count: 345}}))

	assert.Equal(t, 3, s.Size())
	assert.FileExists(t, filepath.Join(dir, "01984174-5600-7000-aded-7d8b151cbd5b#2.jsonl"))
	assert.FileExists(t, filepath.Join(dir, "01984174-59e8-7000-b664-880fc7581c77#1.jsonl"))

	data, err := os.ReadFile(filepath.Join(dir, "01984174-5600-7000-aded-7d8b151cbd5b#2.jsonl"))
	require.NoError(t, err)
	assert.Equal(t, "{\"name\":\"Thing 1\",\"count\":123}\n{\"name\":\"Thing 2\",\"count\":234}\n", string(data))

	// files whose names don't match the spool pattern are ignored
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.txt"), []byte("!"), 0644))

	// if flushing errors for whole batches, files are kept for retry
	fl.err = errors.New("boom")
	require.NoError(t, s.Flush())
	assert.Equal(t, 3, s.Size())
	assert.Equal(t, 0, fl.numBatches())

	// if flushing fails for a single item, it's respooled to a new file
	fl.err = nil
	fl.failing["Thing 2"] = true
	require.NoError(t, s.Flush())
	assert.Equal(t, 2, fl.numBatches())
	assert.Equal(t, 1, s.Size())
	assert.NoFileExists(t, filepath.Join(dir, "01984174-5600-7000-aded-7d8b151cbd5b#2.jsonl"))
	assert.NoFileExists(t, filepath.Join(dir, "01984174-59e8-7000-b664-880fc7581c77#1.jsonl"))

	respooled, err := filepath.Glob(filepath.Join(dir, "*#1.jsonl"))
	require.NoError(t, err)
	assert.Len(t, respooled, 1)

	// and flushed on a subsequent attempt
	fl.failing = map[string]bool{}
	require.NoError(t, s.Flush())
	assert.Equal(t, 3, fl.numBatches())
	assert.Equal(t, 0, s.Size())

	assert.Equal(t, [][]*thing{
		{{Name: "Thing 1", Count: 123}, {Name: "Thing 2", Count: 234}},
		{{Name: "Thing 3", Count: 345}},
		{{Name: "Thing 2", Count: 234}},
	}, fl.batches)
}

func TestSpoolRestart(t *testing.T) {
	uuids.SetGenerator(uuids.NewSeededGenerator(1234, dates.NewSequentialNow(time.Date(2025, 7, 25, 12, 0, 0, 0, time.UTC), time.Second)))
	defer uuids.SetGenerator(uuids.DefaultGenerator)

	dir := filepath.Join(t.TempDir(), "spool")
	fl := &flusher{}

	s := spool.New(dir, time.Hour, spool.MarshalJSON[*thing], spool.UnmarshalJSON[*thing], fl.flush)
	require.NoError(t, s.Start())
	require.NoError(t, s.Add([]*thing{{Name: "Thing 1", Count: 123}, {Name: "Thing 2", Count: 234}}))
	require.NoError(t, s.Add([]*thing{{Name: "Thing 3", Count: 345}}))
	s.Stop()

	// new spool instance picks up size from the existing files and flushes them in the background
	s = spool.New(dir, 100*time.Millisecond, spool.MarshalJSON[*thing], spool.UnmarshalJSON[*thing], fl.flush)
	require.NoError(t, s.Start())
	defer s.Stop()

	assert.Equal(t, 3, s.Size())

	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.Equal(c, 0, s.Size())
		assert.Equal(c, 2, fl.numBatches())
	}, 5*time.Second, 25*time.Millisecond)
}

func TestSpoolCorruptFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "spool")
	fl := &flusher{}

	s := spool.New(dir, time.Hour, spool.MarshalJSON[*thing], spool.UnmarshalJSON[*thing], fl.flush)
	require.NoError(t, s.Start())
	defer s.Stop()

	require.NoError(t, s.Add([]*thing{{Name: "Thing 1", Count: 123}}))

	// a file that can't be read shouldn't prevent other files being flushed
	require.NoError(t, os.WriteFile(filepath.Join(dir, "corrupt#1.jsonl"), []byte("{invalid"), 0644))

	require.NoError(t, s.Flush())
	assert.Equal(t, 1, fl.numBatches())
	assert.Equal(t, 1, s.Size()) // the corrupt file's claimed count
	assert.FileExists(t, filepath.Join(dir, "corrupt#1.jsonl"))
}

func TestSpoolAddMarshalError(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "spool")

	marshal := func(t *thing) ([]byte, error) {
		if t.Name == "bad" {
			return nil, errors.New("can't marshal")
		}
		return spool.MarshalJSON(t)
	}

	s := spool.New(dir, time.Hour, marshal, spool.UnmarshalJSON[*thing], (&flusher{}).flush)
	require.NoError(t, s.Start())
	defer s.Stop()

	// a marshaling failure part way through a batch shouldn't leave a partial file behind
	err := s.Add([]*thing{{Name: "Thing 1", Count: 123}, {Name: "bad"}})
	assert.ErrorContains(t, err, "can't marshal")
	assert.Equal(t, 0, s.Size())

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 0)
}

func TestSpoolStartDirectoryErrors(t *testing.T) {
	fl := &flusher{}

	// a file in place of the directory means it can't be created
	notADir := filepath.Join(t.TempDir(), "spool")
	require.NoError(t, os.WriteFile(notADir, []byte("!"), 0644))

	s := spool.New(notADir, time.Hour, spool.MarshalJSON[*thing], spool.UnmarshalJSON[*thing], fl.flush)
	err := s.Start()
	assert.ErrorContains(t, err, "error creating spool directory")

	// an existing but unwritable directory should fail the writability probe.. but skip if running as
	// root because then permission bits are ignored
	if os.Geteuid() == 0 {
		t.Skip("running as root so can't test unwritable directory")
	}

	unwritable := filepath.Join(t.TempDir(), "spool")
	require.NoError(t, os.Mkdir(unwritable, 0555))

	s = spool.New(unwritable, time.Hour, spool.MarshalJSON[*thing], spool.UnmarshalJSON[*thing], fl.flush)
	err = s.Start()
	assert.ErrorContains(t, err, "is not writable")
}
