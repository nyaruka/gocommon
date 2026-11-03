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
	totals      map[tallies.OwnerID]map[string]int64
	expiries    map[tallies.OwnerID]map[string]string
	generations map[tallies.OwnerID]int64
	lastGenErr  error
	writeErr    error
	onWrite     func()
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		totals:      map[tallies.OwnerID]map[string]int64{},
		expiries:    map[tallies.OwnerID]map[string]string{},
		generations: map[tallies.OwnerID]int64{},
	}
}

func (s *memoryStore) LastGeneration(ctx context.Context, owner tallies.OwnerID) (int64, error) {
	if s.lastGenErr != nil {
		return 0, s.lastGenErr
	}
	return s.generations[owner], nil
}

func (s *memoryStore) Write(
	ctx context.Context, owner tallies.OwnerID, gen int64, amounts map[string]int64, expiries map[string]string,
) error {
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
		s.expiries[owner] = map[string]string{}
	}
	for scope, amount := range amounts {
		// like a table upsert, an expiry only applies to a total being created
		if _, exists := s.totals[owner][scope]; !exists && expiries[scope] != "" {
			s.expiries[owner][scope] = expiries[scope]
		}
		s.totals[owner][scope] += amount
	}
	s.generations[owner] = gen
	return nil
}

func TestDaily(t *testing.T) {
	minus2, plus2 := time.FixedZone("", -2*3600), time.FixedZone("", 2*3600)

	tcs := []struct {
		t         time.Time
		ttl       time.Duration
		scope     string
		expiresOn string
	}{
		{time.Date(2025, 5, 4, 0, 0, 0, 0, time.UTC), 90 * 24 * time.Hour, "foo/bar@2025-05-04", "2025-08-02"},
		{time.Date(2025, 5, 4, 23, 59, 59, 0, time.UTC), 90 * 24 * time.Hour, "foo/bar@2025-05-04", "2025-08-02"},

		// the day and its expiry are the UTC ones
		{time.Date(2025, 5, 4, 23, 0, 0, 0, minus2), 90 * 24 * time.Hour, "foo/bar@2025-05-05", "2025-08-03"},
		{time.Date(2025, 5, 4, 1, 0, 0, 0, plus2), 90 * 24 * time.Hour, "foo/bar@2025-05-03", "2025-08-01"},

		// a period which isn't whole days is counted from the start of the day
		{time.Date(2025, 12, 31, 20, 0, 0, 0, time.UTC), 36 * time.Hour, "foo/bar@2025-12-31", "2026-01-01"},

		// no period means it's kept forever
		{time.Date(2025, 5, 4, 12, 0, 0, 0, time.UTC), 0, "foo/bar@2025-05-04", ""},
	}

	for _, tc := range tcs {
		expected := tallies.Key{Scope: tc.scope, ExpiresOn: tc.expiresOn}
		assert.Equal(t, expected, tallies.Daily("foo/bar", tc.t, tc.ttl), "daily for %s", tc.t)
	}

	assert.PanicsWithValue(t, `tallies: scope "foo@bar" can't contain @`, func() {
		tallies.Daily("foo@bar", time.Now(), 0)
	})
}

func TestTotal(t *testing.T) {
	assert.Equal(t, tallies.Key{Scope: "foo/bar"}, tallies.Total("foo/bar"))
}

func TestIncrement(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	require.NoError(t, tallies.Increment(ctx, vc, 1, map[tallies.Key]int64{}))
	assertvk.SMembers(t, vc, "tallies:pending", []string{})

	err := tallies.Increment(ctx, vc, 1, map[tallies.Key]int64{tallies.Total("_gen"): 1})
	assert.EqualError(t, err, "_gen is a reserved scope")
	err = tallies.Increment(ctx, vc, 1, map[tallies.Key]int64{{Scope: "_gen", ExpiresOn: "2025-06-01"}: 1})
	assert.EqualError(t, err, "_gen is a reserved scope")

	err = tallies.Increment(ctx, vc, 1, map[tallies.Key]int64{{Scope: "foo", ExpiresOn: "2025-13-01"}: 1})
	assert.EqualError(t, err, `scope foo has an invalid expiry "2025-13-01"`)

	err = tallies.Increment(ctx, vc, 1, map[tallies.Key]int64{
		{Scope: "foo", ExpiresOn: "2025-06-01"}: 1,
		{Scope: "foo", ExpiresOn: "2025-07-01"}: 1,
	})
	assert.EqualError(t, err, "scope foo is given different expiries")

	err = tallies.Increment(ctx, vc, 1, map[tallies.Key]int64{
		{Scope: "foo", ExpiresOn: "2025-06-01"}: 1,
		{Scope: "foo"}:                          1,
	})
	assert.EqualError(t, err, "scope foo is given different expiries")

	assertvk.NotExists(t, vc, "tallies:1")

	require.NoError(t, tallies.Increment(ctx, vc, 1, map[tallies.Key]int64{
		tallies.Total("foo"): 2,
		{Scope: "bar@2025-05-04", ExpiresOn: "2025-06-01"}: 1,
	}))

	assertvk.HGetAll(t, vc, "tallies:1", map[string]string{"foo": "2", "bar@2025-05-04": "1"})
	assertvk.HGetAll(t, vc, "tallies:1:expiries", map[string]string{"bar@2025-05-04": "2025-06-01"})
	assertvk.SMembers(t, vc, "tallies:pending", []string{"1"})

	// the first expiry written for a scope wins
	require.NoError(t, tallies.Increment(ctx, vc, 1, map[tallies.Key]int64{
		{Scope: "bar@2025-05-04", ExpiresOn: "2025-07-01"}: 1,
		{Scope: "foo"}: 1,
	}))

	assertvk.HGetAll(t, vc, "tallies:1", map[string]string{"foo": "3", "bar@2025-05-04": "2"})
	assertvk.HGetAll(t, vc, "tallies:1:expiries", map[string]string{"bar@2025-05-04": "2025-06-01"})

	// as does an expiry written after a scope was first written without one
	require.NoError(t, tallies.Increment(ctx, vc, 1, map[tallies.Key]int64{{Scope: "foo", ExpiresOn: "2025-07-01"}: 1}))

	assertvk.HGetAll(t, vc, "tallies:1:expiries", map[string]string{
		"bar@2025-05-04": "2025-06-01", "foo": "2025-07-01",
	})
}

func TestFlush(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	store := newMemoryStore()
	flusher := tallies.NewFlusher(store, 10)

	foo1, foo2 := tallies.Total("foo/1"), tallies.Total("foo/2")
	daily := tallies.Daily("daily", time.Date(2025, 5, 4, 12, 0, 0, 0, time.UTC), 2*24*time.Hour)
	other := tallies.Daily("other", time.Date(2025, 5, 4, 12, 0, 0, 0, time.UTC), 30*24*time.Hour)

	increment := func(owner tallies.OwnerID, amounts map[tallies.Key]int64) {
		require.NoError(t, tallies.Increment(ctx, vc, owner, amounts))
	}
	flush := func(expected int) {
		flushed, err := flusher.Flush(ctx, vc)
		require.NoError(t, err)
		assert.Equal(t, expected, flushed)
	}
	assertStored := func(owner tallies.OwnerID, totals map[string]int64, expiries map[string]string, gen int64) {
		assert.Equal(t, totals, store.totals[owner])
		assert.Equal(t, expiries, store.expiries[owner])
		assert.Equal(t, gen, store.generations[owner])
	}
	assertCleared := func(owner tallies.OwnerID) {
		assertvk.NotExists(t, vc, fmt.Sprintf("tallies:%d", owner))
		assertvk.NotExists(t, vc, fmt.Sprintf("tallies:%d:expiries", owner))
		assertvk.NotExists(t, vc, fmt.Sprintf("tallies:%d:flushing", owner))
		assertvk.NotExists(t, vc, fmt.Sprintf("tallies:%d:expiries:flushing", owner))
	}

	// nothing to flush
	flush(0)

	increment(1, map[tallies.Key]int64{foo1: 2, daily: 1})
	increment(1, map[tallies.Key]int64{foo1: 1})
	increment(2, map[tallies.Key]int64{foo1: 5})

	assertvk.HGetAll(t, vc, "tallies:1", map[string]string{"foo/1": "3", "daily@2025-05-04": "1"})
	assertvk.HGetAll(t, vc, "tallies:1:expiries", map[string]string{"daily@2025-05-04": "2025-05-06"})
	assertvk.NotExists(t, vc, "tallies:2:expiries")
	assertvk.SMembers(t, vc, "tallies:pending", []string{"1", "2"})

	flush(2)

	assertStored(1,
		map[string]int64{"foo/1": 3, "daily@2025-05-04": 1},
		map[string]string{"daily@2025-05-04": "2025-05-06"},
		1,
	)
	assertStored(2, map[string]int64{"foo/1": 5}, map[string]string{}, 1)
	assertCleared(1)
	assertCleared(2)
	assertvk.SMembers(t, vc, "tallies:pending", []string{})

	// a later expiry for a scope the store already has is passed on, but the store keeps the expiry it created it with
	increment(1, map[tallies.Key]int64{foo1: 1, foo2: 1, {Scope: daily.Scope, ExpiresOn: "2025-06-01"}: 1})

	flush(1)

	assertStored(1,
		map[string]int64{"foo/1": 4, "foo/2": 1, "daily@2025-05-04": 2},
		map[string]string{"daily@2025-05-04": "2025-05-06"},
		2,
	)
	assertStored(2, map[string]int64{"foo/1": 5}, map[string]string{}, 1)
	assertCleared(1)

	// a flush which committed but failed before deleting its hashes leaves them behind, and they're discarded rather
	// than written again
	vc.Do("HSET", "tallies:1:flushing", "foo/1", 10, "other@2025-05-04", 1, "_gen", 2)
	vc.Do("HSET", "tallies:1:expiries:flushing", "other@2025-05-04", "2025-05-05")
	increment(1, map[tallies.Key]int64{foo1: 1, other: 1})

	flush(1)

	assertStored(1,
		map[string]int64{"foo/1": 5, "foo/2": 1, "daily@2025-05-04": 2, "other@2025-05-04": 1},
		map[string]string{"daily@2025-05-04": "2025-05-06", "other@2025-05-04": "2025-06-03"},
		3,
	)
	assertCleared(1)

	// a flush which failed before committing leaves its hashes behind too, and new increments are merged into them,
	// with the leftover expiries having been set first
	vc.Do("HSET", "tallies:1:flushing", "foo/1", 10, "new1@2025-05-04", 1, "_gen", 4)
	vc.Do("HSET", "tallies:1:expiries:flushing", "new1@2025-05-04", "2025-05-10")
	vc.Do("SADD", "tallies:pending", 1)
	increment(1, map[tallies.Key]int64{
		foo2: 1,
		{Scope: "new1@2025-05-04", ExpiresOn: "2025-05-20"}: 1,
		{Scope: "new2@2025-05-04", ExpiresOn: "2025-05-30"}: 1,
	})

	flush(1)

	assertStored(1,
		map[string]int64{
			"foo/1": 15, "foo/2": 2, "daily@2025-05-04": 2, "other@2025-05-04": 1,
			"new1@2025-05-04": 2, "new2@2025-05-04": 1,
		},
		map[string]string{
			"daily@2025-05-04": "2025-05-06", "other@2025-05-04": "2025-06-03",
			"new1@2025-05-04": "2025-05-10", "new2@2025-05-04": "2025-05-30",
		},
		4,
	)
	assertCleared(1)
	assertvk.SMembers(t, vc, "tallies:pending", []string{})

	// a failed write leaves the owner pending with its flushing hashes, which other owners don't wait on
	increment(1, map[tallies.Key]int64{foo1: 1, {Scope: "new3@2025-05-04", ExpiresOn: "2025-05-10"}: 1})
	store.writeErr = errors.New("boom")

	flushed, err := flusher.Flush(ctx, vc)
	assert.EqualError(t, err, "error flushing tallies for owner 1: error writing tallies: boom")
	assert.Equal(t, 0, flushed)

	assertvk.NotExists(t, vc, "tallies:1")
	assertvk.NotExists(t, vc, "tallies:1:expiries")
	assertvk.HGetAll(t, vc, "tallies:1:flushing", map[string]string{"foo/1": "1", "new3@2025-05-04": "1", "_gen": "5"})
	assertvk.HGetAll(t, vc, "tallies:1:expiries:flushing", map[string]string{"new3@2025-05-04": "2025-05-10"})
	assertvk.SMembers(t, vc, "tallies:pending", []string{"1"})

	// and it's recovered by the next flush, along with increments made in the meantime, whose expiries don't replace
	// the ones being retried
	store.writeErr = nil
	increment(1, map[tallies.Key]int64{
		foo1: 2,
		{Scope: "new3@2025-05-04", ExpiresOn: "2025-05-20"}: 1,
		{Scope: "new4@2025-05-04", ExpiresOn: "2025-05-20"}: 1,
	})
	increment(2, map[tallies.Key]int64{foo1: 1})

	flush(2)

	assertStored(1,
		map[string]int64{
			"foo/1": 18, "foo/2": 2, "daily@2025-05-04": 2, "other@2025-05-04": 1,
			"new1@2025-05-04": 2, "new2@2025-05-04": 1, "new3@2025-05-04": 2, "new4@2025-05-04": 1,
		},
		map[string]string{
			"daily@2025-05-04": "2025-05-06", "other@2025-05-04": "2025-06-03",
			"new1@2025-05-04": "2025-05-10", "new2@2025-05-04": "2025-05-30",
			"new3@2025-05-04": "2025-05-10", "new4@2025-05-04": "2025-05-20",
		},
		5,
	)
	assertStored(2, map[string]int64{"foo/1": 6}, map[string]string{}, 2)
	assertCleared(1)
	assertCleared(2)
	assertvk.SMembers(t, vc, "tallies:pending", []string{})

	// an increment arriving while an owner is being written leaves it pending for the next flush, expiry and all
	store.onWrite = func() {
		increment(1, map[tallies.Key]int64{{Scope: "new5@2025-05-04", ExpiresOn: "2025-05-20"}: 1})
	}
	increment(1, map[tallies.Key]int64{foo1: 1})

	flush(1)

	assert.Equal(t, int64(19), store.totals[1]["foo/1"])
	assert.Equal(t, int64(6), store.generations[1])
	assertvk.HGetAll(t, vc, "tallies:1", map[string]string{"new5@2025-05-04": "1"})
	assertvk.HGetAll(t, vc, "tallies:1:expiries", map[string]string{"new5@2025-05-04": "2025-05-20"})
	assertvk.NotExists(t, vc, "tallies:1:flushing")
	assertvk.NotExists(t, vc, "tallies:1:expiries:flushing")
	assertvk.SMembers(t, vc, "tallies:pending", []string{"1"})

	store.onWrite = nil

	flush(1)

	assert.Equal(t, int64(1), store.totals[1]["new5@2025-05-04"])
	assert.Equal(t, "2025-05-20", store.expiries[1]["new5@2025-05-04"])
	assert.Equal(t, int64(7), store.generations[1])
	assertCleared(1)
	assertvk.SMembers(t, vc, "tallies:pending", []string{})

	// failing to get the last generation leaves everything as it was
	increment(1, map[tallies.Key]int64{foo1: 1, {Scope: "new6@2025-05-04", ExpiresOn: "2025-05-20"}: 1})
	store.lastGenErr = errors.New("boom")

	flushed, err = flusher.Flush(ctx, vc)
	assert.EqualError(t, err, "error flushing tallies for owner 1: error getting last generation: boom")
	assert.Equal(t, 0, flushed)

	assertvk.HGetAll(t, vc, "tallies:1", map[string]string{"foo/1": "1", "new6@2025-05-04": "1"})
	assertvk.HGetAll(t, vc, "tallies:1:expiries", map[string]string{"new6@2025-05-04": "2025-05-20"})
	assertvk.NotExists(t, vc, "tallies:1:flushing")
	assertvk.SMembers(t, vc, "tallies:pending", []string{"1"})

	store.lastGenErr = nil

	flush(1)

	assert.Equal(t, int64(20), store.totals[1]["foo/1"])
	assert.Equal(t, "2025-05-20", store.expiries[1]["new6@2025-05-04"])
	assert.Equal(t, int64(8), store.generations[1])
	assertCleared(1)
}

func TestFlushBatches(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	assert.Panics(t, func() { tallies.NewFlusher(newMemoryStore(), 0) })

	store := newMemoryStore()
	flusher := tallies.NewFlusher(store, 2)

	for owner := range tallies.OwnerID(3) {
		require.NoError(t, tallies.Increment(ctx, vc, owner+1, map[tallies.Key]int64{tallies.Total("foo"): 1}))
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

	err := tallies.Increment(ctx, vc, 1, map[tallies.Key]int64{tallies.Total("foo"): 1})
	assert.EqualError(t, err, "error incrementing tallies for owner 1: ERR hash value is not an integer")
}
