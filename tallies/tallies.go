// Package tallies buffers per-owner totals of things which only go up in Valkey and flushes them to a store in
// batches, so that frequent increments cost one store write per scope per flush rather than a write each.
//
// Keys, for a key base of foo:
//   - {foo}:live - hash of <owner>|<scope> to amount, incremented by writers
//   - {foo}:flushing - snapshot of live being flushed
//   - {foo}:batch - fields moved out of flushing which are being written to the store
//   - {foo}:batch_id - id of the most recently created batch
//
// Everything which increments or flushes the same tallies must use the same key base.
//
// Scopes separate their parts with / and end with @ and the day if they're daily, e.g. foo/bar@2025-05-04.
//
// A flush renames live to flushing and then writes it to the store a batch at a time. Batch ids only go up, and the
// store records each one in the same transaction as its amounts, so a batch left behind by a failed flush is written
// again only if the store doesn't have it yet. Concurrent flushes of the same tallies
// can't double count or lose amounts, but only one makes progress at a time, so they should be prevented with a lock.
//
// A reader gets consistent totals of an owner's scopes by:
//  1. Reading their fields from live, flushing and batch, whether batch exists, and batch_id, in one transaction
//  2. Reading their totals and the last batch from the store, from the same snapshot
//  3. Including the amounts read from batch only if it exists and the last batch is less than batch_id (0 if missing)
//  4. Reading again if the last batch is greater than batch_id, as a batch was written between the two reads
package tallies

import (
	"context"
	"fmt"
	"strings"
	"time"

	valkey "github.com/gomodule/redigo/redis"
)

// OwnerID is the identifier of the owner that tallies belong to.
type OwnerID int64

// Tallies are per-owner totals buffered under a key base.
type Tallies struct {
	keyBase string
}

// New creates new tallies with the given key base. Panics if the key base is empty or contains { or }, as then it
// can't be used as the hash tag which keeps all its keys in the same slot.
func New(keyBase string) *Tallies {
	if keyBase == "" || strings.ContainsAny(keyBase, "{}") {
		panic(fmt.Sprintf("tallies: invalid key base %q", keyBase))
	}
	return &Tallies{keyBase: keyBase}
}

// KeyBase returns the key base, which a store can use to keep the last batch of each set of tallies separately.
func (t *Tallies) KeyBase() string { return t.keyBase }

func (t *Tallies) liveKey() string     { return fmt.Sprintf("{%s}:live", t.keyBase) }
func (t *Tallies) flushingKey() string { return fmt.Sprintf("{%s}:flushing", t.keyBase) }
func (t *Tallies) batchKey() string    { return fmt.Sprintf("{%s}:batch", t.keyBase) }
func (t *Tallies) batchIDKey() string  { return fmt.Sprintf("{%s}:batch_id", t.keyBase) }

func field(owner OwnerID, scope string) string { return fmt.Sprintf("%d|%s", owner, scope) }

// Daily returns the scope for the total of the given scope on the UTC day of the given time. Panics if the given scope
// contains @ as it's reserved for separating the day.
func Daily(scope string, t time.Time) string {
	if strings.Contains(scope, "@") {
		panic(fmt.Sprintf("tallies: scope %q can't contain @", scope))
	}
	return scope + "@" + t.UTC().Format(time.DateOnly)
}

// Increment adds the given amounts by scope to the tallies of the given owner.
func (t *Tallies) Increment(ctx context.Context, vc valkey.Conn, owner OwnerID, amounts map[string]int64) error {
	if len(amounts) == 0 {
		return nil
	}

	vc.Send("MULTI")
	for scope, amount := range amounts {
		vc.Send("HINCRBY", t.liveKey(), field(owner, scope), amount)
	}

	replies, err := valkey.Values(valkey.DoContext(vc, ctx, "EXEC"))
	if err != nil {
		return fmt.Errorf("error incrementing tallies for owner %d: %w", owner, err)
	}

	// a command failing inside a transaction doesn't fail EXEC, only its own reply
	for _, reply := range replies {
		if err, ok := reply.(valkey.Error); ok {
			return fmt.Errorf("error incrementing tallies for owner %d: %w", owner, err)
		}
	}
	return nil
}
