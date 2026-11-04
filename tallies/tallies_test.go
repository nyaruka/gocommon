package tallies_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/tallies"
	"github.com/nyaruka/vkutil/assertvk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain opts into coordinated test database claims, matching the dev stack's valkey.
func TestMain(m *testing.M) {
	assertvk.Coordinate(16, 17, 63)

	os.Exit(m.Run())
}

// memoryStore is a store which keeps totals in memory and can be made to fail.
type memoryStore struct {
	totals       map[tallies.OwnerID]map[string]int64
	lastBatch    int64
	batches      []int // sizes of the batches written
	lastBatchErr error
	writeErr     error
	onWrite      func()
}

func newMemoryStore() *memoryStore {
	return &memoryStore{totals: map[tallies.OwnerID]map[string]int64{}}
}

func (s *memoryStore) LastBatch(ctx context.Context) (int64, error) {
	return s.lastBatch, s.lastBatchErr
}

func (s *memoryStore) Write(ctx context.Context, id int64, amounts []tallies.Amount) (bool, error) {
	if s.writeErr != nil {
		return false, s.writeErr
	}
	if s.onWrite != nil {
		s.onWrite()
	}
	if id <= s.lastBatch {
		return false, nil
	}
	for _, a := range amounts {
		if s.totals[a.Owner] == nil {
			s.totals[a.Owner] = map[string]int64{}
		}
		s.totals[a.Owner][a.Scope] += a.Count
	}
	s.lastBatch = id
	s.batches = append(s.batches, len(amounts))
	return true, nil
}

func TestDaily(t *testing.T) {
	minus2, plus2 := time.FixedZone("", -2*3600), time.FixedZone("", 2*3600)

	assert.Equal(t, "foo/bar@2025-05-04", tallies.Daily("foo/bar", time.Date(2025, 5, 4, 0, 0, 0, 0, time.UTC)))
	assert.Equal(t, "foo/bar@2025-05-04", tallies.Daily("foo/bar", time.Date(2025, 5, 4, 23, 59, 59, 0, time.UTC)))

	// the day is the UTC one
	assert.Equal(t, "foo/bar@2025-05-05", tallies.Daily("foo/bar", time.Date(2025, 5, 4, 23, 0, 0, 0, minus2)))
	assert.Equal(t, "foo/bar@2025-05-03", tallies.Daily("foo/bar", time.Date(2025, 5, 4, 1, 0, 0, 0, plus2)))

	assert.PanicsWithValue(t, `tallies: scope "foo@bar" can't contain @`, func() { tallies.Daily("foo@bar", time.Now()) })
}

func TestIncrement(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	tl := tallies.New("tallies")
	assert.Equal(t, "tallies", tl.KeyBase())

	require.NoError(t, tl.Increment(ctx, vc, 1, map[string]int64{}))
	assertvk.NotExists(t, vc, "{tallies}:live")

	require.NoError(t, tl.Increment(ctx, vc, 1, map[string]int64{"foo": 2, "bar@2025-05-04": 1}))
	require.NoError(t, tl.Increment(ctx, vc, 1, map[string]int64{"foo": 1}))
	require.NoError(t, tl.Increment(ctx, vc, 23, map[string]int64{"foo": 5, "a|b": 1}))

	assertvk.HGetAll(t, vc, "{tallies}:live", map[string]string{
		"1|foo": "3", "1|bar@2025-05-04": "1", "23|foo": "5", "23|a|b": "1",
	})

	// a failing command inside the transaction is reported even though the transaction itself succeeds
	vc.Do("HSET", "{tallies}:live", "1|foo", "bar")

	err := tl.Increment(ctx, vc, 1, map[string]int64{"foo": 1})
	assert.EqualError(t, err, "error incrementing tallies for owner 1: ERR hash value is not an integer")
}

func TestFlush(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	tl := tallies.New("tallies")
	store := newMemoryStore()

	assert.Panics(t, func() { tl.Flush(ctx, vc, store, 0) })

	increment := func(owner tallies.OwnerID, amounts map[string]int64) {
		require.NoError(t, tl.Increment(ctx, vc, owner, amounts))
	}
	flush := func(batchSize, expected int) {
		flushed, err := tl.Flush(ctx, vc, store, batchSize)
		require.NoError(t, err)
		assert.Equal(t, expected, flushed)
	}
	assertCleared := func() {
		assertvk.NotExists(t, vc, "{tallies}:live")
		assertvk.NotExists(t, vc, "{tallies}:flushing")
		assertvk.NotExists(t, vc, "{tallies}:batch")
	}

	// nothing to flush
	flush(10, 0)
	assert.Equal(t, int64(0), store.lastBatch)

	increment(1, map[string]int64{"foo": 2, "bar@2025-05-04": 1})
	increment(1, map[string]int64{"foo": 1})
	increment(2, map[string]int64{"foo": 5, "a|b": 1})

	flush(10, 4)

	assert.Equal(t, map[tallies.OwnerID]map[string]int64{
		1: {"foo": 3, "bar@2025-05-04": 1},
		2: {"foo": 5, "a|b": 1},
	}, store.totals)
	assert.Equal(t, int64(1), store.lastBatch)
	assert.Equal(t, []int{4}, store.batches)
	assertvk.Get(t, vc, "{tallies}:batch_id", "1")
	assertCleared()

	// a flush drains everything buffered when it started, a batch at a time
	increment(1, map[string]int64{"foo": 1, "a": 1, "b": 1})
	increment(3, map[string]int64{"foo": 1, "a": 1})

	flush(2, 5)

	assert.Equal(t, int64(4), store.lastBatch)
	assert.Equal(t, []int{4, 2, 2, 1}, store.batches)
	assert.Equal(t, map[string]int64{"foo": 4, "bar@2025-05-04": 1, "a": 1, "b": 1}, store.totals[1])
	assert.Equal(t, map[string]int64{"foo": 1, "a": 1}, store.totals[3])
	assertCleared()

	// but not increments made during it, which wait for the next one
	store.onWrite = func() { increment(1, map[string]int64{"foo": 10}) }
	increment(1, map[string]int64{"foo": 1})

	flush(10, 1)

	assert.Equal(t, int64(5), store.totals[1]["foo"])
	assertvk.HGetAll(t, vc, "{tallies}:live", map[string]string{"1|foo": "10"})

	store.onWrite = nil
	flush(10, 1)

	assert.Equal(t, int64(15), store.totals[1]["foo"])
	assert.Equal(t, int64(6), store.lastBatch)
	assertCleared()

	// failing to get the last batch leaves everything as it was
	increment(1, map[string]int64{"foo": 1})
	store.lastBatchErr = errors.New("boom")

	_, err := tl.Flush(ctx, vc, store, 10)
	assert.EqualError(t, err, "error getting last batch: boom")
	assertvk.HGetAll(t, vc, "{tallies}:live", map[string]string{"1|foo": "1"})

	store.lastBatchErr = nil

	// a failed write leaves its batch, and the rest of the snapshot, for the next flush to write
	increment(2, map[string]int64{"foo": 1, "a|b": 1})
	store.writeErr = errors.New("boom")

	flushed, err := tl.Flush(ctx, vc, store, 2)
	assert.EqualError(t, err, "error writing batch 7: boom")
	assert.Equal(t, 0, flushed)

	assertvk.HLen(t, vc, "{tallies}:batch", 2)
	assertvk.HLen(t, vc, "{tallies}:flushing", 1)
	assertvk.Get(t, vc, "{tallies}:batch_id", "7")

	// and does so before taking a new snapshot, which includes increments made in the meantime
	store.writeErr = nil
	increment(2, map[string]int64{"foo": 1})

	flush(2, 4)

	assert.Equal(t, int64(16), store.totals[1]["foo"])
	assert.Equal(t, map[string]int64{"foo": 7, "a|b": 2}, store.totals[2])
	assert.Equal(t, int64(9), store.lastBatch)
	assertCleared()

	// a batch which was written but not deleted is deleted without being written again
	vc.Do("HSET", "{tallies}:batch", "1|foo", 100)
	vc.Do("SET", "{tallies}:batch_id", 9)
	increment(1, map[string]int64{"foo": 1})

	flush(10, 1)

	assert.Equal(t, int64(17), store.totals[1]["foo"])
	assert.Equal(t, int64(10), store.lastBatch)
	assertCleared()

	// a batch without an id can't be written safely, so stops flushing until it's dealt with
	vc.Do("HSET", "{tallies}:batch", "1|foo", 100)
	vc.Do("DEL", "{tallies}:batch_id")

	_, err = tl.Flush(ctx, vc, store, 10)
	assert.EqualError(t, err, "error reading id of leftover batch from {tallies}:batch_id: redigo: nil returned")

	vc.Do("DEL", "{tallies}:batch")
	vc.Do("SET", "{tallies}:batch_id", 10)

	// batch ids carry on from the last one used if the store is behind, e.g. because it was reset
	store.lastBatch = 3
	increment(1, map[string]int64{"foo": 1})

	flush(10, 1)

	assert.Equal(t, int64(11), store.lastBatch)
	assertvk.Get(t, vc, "{tallies}:batch_id", "11")

	// or from the store's if Valkey is behind, e.g. because it lost data
	vc.Do("DEL", "{tallies}:batch_id")
	increment(1, map[string]int64{"foo": 1})

	flush(10, 1)

	assert.Equal(t, int64(12), store.lastBatch)
	assertvk.Get(t, vc, "{tallies}:batch_id", "12")
	assert.Equal(t, int64(19), store.totals[1]["foo"])
}

func TestFlushConcurrently(t *testing.T) {
	ctx := t.Context()
	db := assertvk.ClaimDB(t)
	vc, vc2 := db.Pool().Get(), db.Pool().Get()
	defer vc.Close()
	defer vc2.Close()

	tl := tallies.New("tallies")
	store := newMemoryStore()

	require.NoError(t, tl.Increment(ctx, vc, 1, map[string]int64{"a": 1, "b": 1, "c": 1}))

	// another flush which runs while this one is writing its first batch writes that batch itself and then the rest,
	// so this one's write of it does nothing and it finds nothing left
	var otherFlushed int
	var otherErr error
	store.onWrite = func() {
		store.onWrite = nil
		otherFlushed, otherErr = tl.Flush(ctx, vc2, store, 1)
	}

	flushed, err := tl.Flush(ctx, vc, store, 1)
	assert.NoError(t, err)
	assert.Equal(t, 0, flushed)
	assert.NoError(t, otherErr)
	assert.Equal(t, 3, otherFlushed)

	assert.Equal(t, map[string]int64{"a": 1, "b": 1, "c": 1}, store.totals[1])
	assert.Equal(t, int64(3), store.lastBatch)
	assertvk.NotExists(t, vc, "{tallies}:flushing")
	assertvk.NotExists(t, vc, "{tallies}:batch")
}

func TestFlushLargeBatches(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	tl := tallies.New("tallies")
	store := newMemoryStore()

	amounts := make(map[string]int64, 2500)
	for i := range 2500 {
		amounts[fmt.Sprintf("scope%d", i)] = 1
	}
	require.NoError(t, tl.Increment(ctx, vc, 1, amounts))

	flushed, err := tl.Flush(ctx, vc, store, 2000)
	require.NoError(t, err)
	assert.Equal(t, 2500, flushed)
	assert.Equal(t, []int{2000, 500}, store.batches)
	assert.Len(t, store.totals[1], 2500)
}

func TestKeyBases(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	for _, keyBase := range []string{"", "{foo}", "foo}"} {
		assert.PanicsWithValue(t, fmt.Sprintf("tallies: invalid key base %q", keyBase), func() { tallies.New(keyBase) })
	}

	foo, bar := tallies.New("foo"), tallies.New("bar")
	store := newMemoryStore()

	require.NoError(t, foo.Increment(ctx, vc, 1, map[string]int64{"x": 1}))
	require.NoError(t, bar.Increment(ctx, vc, 1, map[string]int64{"x": 2}))

	assertvk.HGetAll(t, vc, "{foo}:live", map[string]string{"1|x": "1"})
	assertvk.HGetAll(t, vc, "{bar}:live", map[string]string{"1|x": "2"})

	// each only flushes its own tallies
	flushed, err := foo.Flush(ctx, vc, store, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, flushed)
	assert.Equal(t, map[string]int64{"x": 1}, store.totals[1])
	assertvk.NotExists(t, vc, "{foo}:live")
	assertvk.HGetAll(t, vc, "{bar}:live", map[string]string{"1|x": "2"})
}
