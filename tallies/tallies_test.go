package tallies_test

import (
	"context"
	"errors"
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
	totals      map[tallies.OwnerID]map[string]int64
	generations map[tallies.OwnerID]int64
	lastGenErr  error
	writeErr    error
	onWrite     func()
}

func newMemoryStore() *memoryStore {
	return &memoryStore{totals: map[tallies.OwnerID]map[string]int64{}, generations: map[tallies.OwnerID]int64{}}
}

func (s *memoryStore) LastGeneration(ctx context.Context, owner tallies.OwnerID) (int64, error) {
	if s.lastGenErr != nil {
		return 0, s.lastGenErr
	}
	return s.generations[owner], nil
}

func (s *memoryStore) Write(ctx context.Context, owner tallies.OwnerID, gen int64, amounts map[string]int64) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	if s.onWrite != nil {
		s.onWrite()
	}
	if gen <= s.generations[owner] {
		return nil
	}
	if s.totals[owner] == nil {
		s.totals[owner] = map[string]int64{}
	}
	for scope, amount := range amounts {
		s.totals[owner][scope] += amount
	}
	s.generations[owner] = gen
	return nil
}

func TestDaily(t *testing.T) {
	assert.Equal(t, "foo/bar@2025-05-04", tallies.Daily("foo/bar", time.Date(2025, 5, 4, 23, 59, 0, 0, time.UTC)))
	assert.Equal(t, "foo/bar@2025-05-05", tallies.Daily("foo/bar", time.Date(2025, 5, 4, 23, 0, 0, 0, time.FixedZone("", -2*3600))))
}

func TestParseDaily(t *testing.T) {
	base, day, ok := tallies.ParseDaily("foo/bar:in@2025-05-04")
	assert.True(t, ok)
	assert.Equal(t, "foo/bar:in", base)
	assert.Equal(t, time.Date(2025, 5, 4, 0, 0, 0, 0, time.UTC), day)

	for _, scope := range []string{"foo/bar", "foo/2025-05-04", "foo@", "foo@2025-13-01", "foo@2025-05-04@2025-05-05"} {
		_, _, ok := tallies.ParseDaily(scope)
		assert.False(t, ok, "expected %s to not parse", scope)
	}
}

func TestFlush(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	store := newMemoryStore()
	flusher := tallies.NewFlusher(store, 10)

	increment := func(owner tallies.OwnerID, amounts map[string]int64) {
		require.NoError(t, tallies.Increment(ctx, vc, owner, amounts))
	}
	flush := func(expected int) {
		flushed, err := flusher.Flush(ctx, vc)
		require.NoError(t, err)
		assert.Equal(t, expected, flushed)
	}
	assertStored := func(owner tallies.OwnerID, expected map[string]int64, expectedGen int64) {
		assert.Equal(t, expected, store.totals[owner])
		assert.Equal(t, expectedGen, store.generations[owner])
	}

	// nothing to flush
	flush(0)

	increment(1, map[string]int64{})
	assertvk.SMembers(t, vc, "tallies:pending", []string{})

	err := tallies.Increment(ctx, vc, 1, map[string]int64{"_gen": 1})
	assert.EqualError(t, err, "_gen is a reserved scope")

	increment(1, map[string]int64{"foo/1": 2, "daily@2025-05-04": 1})
	increment(1, map[string]int64{"foo/1": 1})
	increment(2, map[string]int64{"foo/1": 5})

	assertvk.HGetAll(t, vc, "tallies:1", map[string]string{"foo/1": "3", "daily@2025-05-04": "1"})
	assertvk.SMembers(t, vc, "tallies:pending", []string{"1", "2"})

	flush(2)

	assertStored(1, map[string]int64{"foo/1": 3, "daily@2025-05-04": 1}, 1)
	assertStored(2, map[string]int64{"foo/1": 5}, 1)
	assertvk.NotExists(t, vc, "tallies:1")
	assertvk.NotExists(t, vc, "tallies:1:flushing")
	assertvk.SMembers(t, vc, "tallies:pending", []string{})

	increment(1, map[string]int64{"foo/1": 1, "foo/2": 1, "daily@2025-05-04": 1})

	flush(1)

	assertStored(1, map[string]int64{"foo/1": 4, "foo/2": 1, "daily@2025-05-04": 2}, 2)
	assertStored(2, map[string]int64{"foo/1": 5}, 1)

	// a flush which committed but failed before deleting its hash leaves it behind, and it's discarded rather than
	// written again
	vc.Do("HSET", "tallies:1:flushing", "foo/1", 10, "_gen", 2)
	increment(1, map[string]int64{"foo/1": 1})

	flush(1)

	assertStored(1, map[string]int64{"foo/1": 5, "foo/2": 1, "daily@2025-05-04": 2}, 3)

	// a flush which failed before committing leaves its hash behind too, and new increments are merged into it
	vc.Do("HSET", "tallies:1:flushing", "foo/1", 10, "_gen", 4)
	vc.Do("SADD", "tallies:pending", 1)
	increment(1, map[string]int64{"foo/2": 1})

	flush(1)

	assertStored(1, map[string]int64{"foo/1": 15, "foo/2": 2, "daily@2025-05-04": 2}, 4)
	assertvk.NotExists(t, vc, "tallies:1:flushing")
	assertvk.SMembers(t, vc, "tallies:pending", []string{})

	// a failed write leaves the owner pending with its flushing hash, which other owners don't wait on
	increment(1, map[string]int64{"foo/1": 1})
	store.writeErr = errors.New("boom")

	flushed, err := flusher.Flush(ctx, vc)
	assert.EqualError(t, err, "error flushing tallies for owner 1: error writing tallies: boom")
	assert.Equal(t, 0, flushed)

	assertStored(1, map[string]int64{"foo/1": 15, "foo/2": 2, "daily@2025-05-04": 2}, 4)
	assertvk.NotExists(t, vc, "tallies:1")
	assertvk.HGetAll(t, vc, "tallies:1:flushing", map[string]string{"foo/1": "1", "_gen": "5"})
	assertvk.SMembers(t, vc, "tallies:pending", []string{"1"})

	// and it's recovered by the next flush, along with increments made in the meantime
	store.writeErr = nil
	increment(1, map[string]int64{"foo/1": 2})
	increment(2, map[string]int64{"foo/1": 1})

	flush(2)

	assertStored(1, map[string]int64{"foo/1": 18, "foo/2": 2, "daily@2025-05-04": 2}, 5)
	assertStored(2, map[string]int64{"foo/1": 6}, 2)
	assertvk.NotExists(t, vc, "tallies:1")
	assertvk.NotExists(t, vc, "tallies:1:flushing")
	assertvk.SMembers(t, vc, "tallies:pending", []string{})

	// an increment arriving while an owner is being written leaves it pending for the next flush
	store.onWrite = func() { increment(1, map[string]int64{"foo/2": 1}) }
	increment(1, map[string]int64{"foo/1": 1})

	flush(1)

	assertStored(1, map[string]int64{"foo/1": 19, "foo/2": 2, "daily@2025-05-04": 2}, 6)
	assertvk.HGetAll(t, vc, "tallies:1", map[string]string{"foo/2": "1"})
	assertvk.NotExists(t, vc, "tallies:1:flushing")
	assertvk.SMembers(t, vc, "tallies:pending", []string{"1"})

	store.onWrite = nil

	flush(1)

	assertStored(1, map[string]int64{"foo/1": 19, "foo/2": 3, "daily@2025-05-04": 2}, 7)
	assertvk.SMembers(t, vc, "tallies:pending", []string{})

	// failing to get the last generation leaves everything as it was
	increment(1, map[string]int64{"foo/1": 1})
	store.lastGenErr = errors.New("boom")

	flushed, err = flusher.Flush(ctx, vc)
	assert.EqualError(t, err, "error flushing tallies for owner 1: error getting last generation: boom")
	assert.Equal(t, 0, flushed)

	assertvk.HGetAll(t, vc, "tallies:1", map[string]string{"foo/1": "1"})
	assertvk.NotExists(t, vc, "tallies:1:flushing")
	assertvk.SMembers(t, vc, "tallies:pending", []string{"1"})

	store.lastGenErr = nil

	flush(1)

	assertStored(1, map[string]int64{"foo/1": 20, "foo/2": 3, "daily@2025-05-04": 2}, 8)
}

func TestFlushBatches(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	assert.Panics(t, func() { tallies.NewFlusher(newMemoryStore(), 0) })

	store := newMemoryStore()
	flusher := tallies.NewFlusher(store, 2)

	for owner := range tallies.OwnerID(3) {
		require.NoError(t, tallies.Increment(ctx, vc, owner+1, map[string]int64{"foo": 1}))
	}

	flushed, err := flusher.Flush(ctx, vc)
	require.NoError(t, err)
	assert.Equal(t, 2, flushed)
	assertvk.SCard(t, vc, "tallies:pending", 1)

	flushed, err = flusher.Flush(ctx, vc)
	require.NoError(t, err)
	assert.Equal(t, 1, flushed)
	assertvk.SCard(t, vc, "tallies:pending", 0)

	for owner := range tallies.OwnerID(3) {
		assert.Equal(t, map[string]int64{"foo": 1}, store.totals[owner+1])
	}
}

func TestIncrementError(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	// a failing command inside the transaction is reported even though the transaction itself succeeds
	vc.Do("HSET", "tallies:1", "foo", "bar")

	err := tallies.Increment(ctx, vc, 1, map[string]int64{"foo": 1})
	assert.EqualError(t, err, "error incrementing tallies for owner 1: ERR hash value is not an integer")
}
