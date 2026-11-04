package tallies

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"strings"

	valkey "github.com/gomodule/redigo/redis"
)

//go:embed lua/snapshot.lua
var luaSnapshot string
var scriptSnapshot = valkey.NewScript(2, luaSnapshot)

//go:embed lua/batch.lua
var luaBatch string
var scriptBatch = valkey.NewScript(3, luaBatch)

//go:embed lua/done.lua
var luaDone string
var scriptDone = valkey.NewScript(2, luaDone)

// Amount is an amount to add to an owner's total of a scope.
type Amount struct {
	Owner OwnerID
	Scope string
	Count int64
}

// Store is where flushed tallies are written, a batch at a time.
type Store interface {
	// LastBatch returns the id of the last batch written, or zero if there is none.
	LastBatch(ctx context.Context) (int64, error)

	// Write must add the given amounts to their totals and record id as the last batch in one transaction, returning
	// whether it did. It must do nothing if the last batch is already greater than or equal to id, checking that
	// atomically with recording id (e.g. with a conditional update of the last batch) as concurrent flushes can write
	// the same batch at the same time.
	Write(ctx context.Context, id int64, amounts []Amount) (bool, error)
}

// Flush writes everything buffered when it starts to the given store in batches of up to batchSize amounts, along
// with anything left by a failed flush, returning the number of amounts written. Increments made during a flush are
// left for the next one.
func (t *Tallies) Flush(ctx context.Context, vc valkey.Conn, store Store, batchSize int) (int, error) {
	if batchSize <= 0 {
		panic("tallies: flush batch size must be positive")
	}

	lastBatch, err := store.LastBatch(ctx)
	if err != nil {
		return 0, fmt.Errorf("error getting last batch: %w", err)
	}

	flushed, err := t.finishBatch(ctx, vc, store, &lastBatch)
	if err != nil {
		return flushed, err
	}

	// drain any snapshot left by a failed flush before taking a new one
	n, err := t.drain(ctx, vc, store, batchSize, &lastBatch)
	flushed += n
	if err != nil {
		return flushed, err
	}

	if _, err := scriptSnapshot.DoContext(ctx, vc, t.liveKey(), t.flushingKey()); err != nil {
		return flushed, fmt.Errorf("error taking snapshot: %w", err)
	}

	n, err = t.drain(ctx, vc, store, batchSize, &lastBatch)
	flushed += n
	return flushed, err
}

// finishBatch writes a batch left by a failed flush, unless the store already has it, and deletes it
func (t *Tallies) finishBatch(ctx context.Context, vc valkey.Conn, store Store, lastBatch *int64) (int, error) {
	vc.Send("MULTI")
	vc.Send("HGETALL", t.batchKey())
	vc.Send("GET", t.batchIDKey())
	replies, err := valkey.Values(valkey.DoContext(vc, ctx, "EXEC"))
	if err != nil {
		return 0, fmt.Errorf("error reading leftover batch: %w", err)
	}

	kvs, err := valkey.Strings(replies[0], nil)
	if err != nil {
		return 0, fmt.Errorf("error reading leftover batch: %w", err)
	}
	if len(kvs) == 0 {
		return 0, nil
	}
	id, err := valkey.Int64(replies[1], nil)
	if err != nil {
		return 0, fmt.Errorf("error reading leftover batch id: %w", err)
	}

	written := 0
	if id > *lastBatch {
		if written, err = t.writeBatch(ctx, store, id, kvs); err != nil {
			return 0, err
		}
		*lastBatch = id
	}

	return written, t.deleteBatch(ctx, vc, id)
}

// drain writes the flushing snapshot to the store a batch at a time until it's empty
func (t *Tallies) drain(
	ctx context.Context, vc valkey.Conn, store Store, batchSize int, lastBatch *int64,
) (int, error) {
	flushed := 0
	for {
		reply, err := valkey.Values(scriptBatch.DoContext(
			ctx, vc, t.flushingKey(), t.batchKey(), t.batchIDKey(), *lastBatch, batchSize,
		))
		if err != nil {
			return flushed, fmt.Errorf("error creating batch: %w", err)
		}
		id, _ := valkey.Int64(reply[0], nil)
		kvs, err := valkey.Strings(reply[1], nil)
		if err != nil {
			return flushed, fmt.Errorf("error reading batch %d: %w", id, err)
		}
		if len(kvs) == 0 {
			return flushed, nil
		}

		n, err := t.writeBatch(ctx, store, id, kvs)
		if err != nil {
			return flushed, err
		}
		flushed += n
		*lastBatch = id

		if err := t.deleteBatch(ctx, vc, id); err != nil {
			return flushed, err
		}
	}
}

func (t *Tallies) writeBatch(ctx context.Context, store Store, id int64, kvs []string) (int, error) {
	amounts := make([]Amount, 0, len(kvs)/2)
	for i := 0; i < len(kvs); i += 2 {
		ownerStr, scope, ok := strings.Cut(kvs[i], "|")
		owner, err := strconv.ParseInt(ownerStr, 10, 64)
		if !ok || err != nil {
			return 0, fmt.Errorf("invalid field %q in batch %d", kvs[i], id)
		}
		count, err := strconv.ParseInt(kvs[i+1], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid amount %q in batch %d", kvs[i+1], id)
		}
		amounts = append(amounts, Amount{Owner: OwnerID(owner), Scope: scope, Count: count})
	}

	written, err := store.Write(ctx, id, amounts)
	if err != nil {
		return 0, fmt.Errorf("error writing batch %d: %w", id, err)
	}
	if !written {
		return 0, nil
	}
	return len(amounts), nil
}

func (t *Tallies) deleteBatch(ctx context.Context, vc valkey.Conn, id int64) error {
	if _, err := scriptDone.DoContext(ctx, vc, t.batchKey(), t.batchIDKey(), id); err != nil {
		return fmt.Errorf("error deleting batch %d: %w", id, err)
	}
	return nil
}
