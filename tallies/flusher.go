package tallies

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	valkey "github.com/gomodule/redigo/redis"
)

//go:embed lua/move.lua
var luaMove string
var scriptMove = valkey.NewScript(4, luaMove)

//go:embed lua/done.lua
var luaDone string
var scriptDone = valkey.NewScript(4, luaDone)

// Store is where flushed tallies are written. Each flush of an owner has a generation which increases by one each time
// it is written, so that a flush which failed after writing isn't written again by its retry.
type Store interface {
	// LastGeneration returns the generation last recorded by Write for the given owner, or zero if there is none.
	LastGeneration(ctx context.Context, owner OwnerID) (int64, error)

	// Write must add the given amounts to the owner's totals and record gen in one transaction, and do nothing if gen
	// is less than or equal to the recorded generation. Expiries are YYYY-MM-DD dates by scope, for scopes which have
	// one, to be applied to totals which don't exist yet.
	Write(ctx context.Context, owner OwnerID, gen int64, amounts map[string]int64, expiries map[string]string) error
}

// Flusher writes buffered tallies to a store.
type Flusher struct {
	store     Store
	batchSize int
}

// NewFlusher creates a new flusher which flushes up to batchSize owners per call to Flush.
func NewFlusher(store Store, batchSize int) *Flusher {
	if batchSize <= 0 {
		panic("tallies: flusher batch size must be positive")
	}
	return &Flusher{store: store, batchSize: batchSize}
}

// Flush flushes the tallies of pending owners, returning the number of owners flushed.
func (f *Flusher) Flush(ctx context.Context, vc valkey.Conn) (int, error) {
	owners, err := valkey.Int64s(valkey.DoContext(vc, ctx, "SRANDMEMBER", pendingKey, f.batchSize))
	if err != nil {
		return 0, fmt.Errorf("error getting pending owners: %w", err)
	}

	var errs []error
	for _, owner := range owners {
		if err := f.flushOwner(ctx, vc, OwnerID(owner)); err != nil {
			errs = append(errs, fmt.Errorf("error flushing tallies for owner %d: %w", owner, err))
		}
	}

	return len(owners) - len(errs), errors.Join(errs...)
}

func (f *Flusher) flushOwner(ctx context.Context, vc valkey.Conn, owner OwnerID) error {
	lastGen, err := f.store.LastGeneration(ctx, owner)
	if err != nil {
		return fmt.Errorf("error getting last generation: %w", err)
	}

	moved, err := valkey.Values(scriptMove.DoContext(
		ctx, vc, liveKey(owner), liveExpiriesKey(owner), flushingKey(owner), flushingExpiriesKey(owner), lastGen,
	))
	if err != nil {
		return fmt.Errorf("error moving live hashes: %w", err)
	}
	amounts, err := valkey.Int64Map(moved[0], nil)
	if err != nil {
		return fmt.Errorf("error reading flushing hash: %w", err)
	}
	expiries, err := valkey.StringMap(moved[1], nil)
	if err != nil {
		return fmt.Errorf("error reading flushing expiries: %w", err)
	}

	if len(amounts) > 0 {
		gen := amounts[generationField]
		delete(amounts, generationField)

		if err := f.store.Write(ctx, owner, gen, amounts, expiries); err != nil {
			return fmt.Errorf("error writing tallies: %w", err)
		}
	}

	_, err = scriptDone.DoContext(
		ctx, vc, liveKey(owner), flushingKey(owner), flushingExpiriesKey(owner), pendingKey, owner,
	)
	if err != nil {
		return fmt.Errorf("error deleting flushing hashes: %w", err)
	}
	return nil
}
