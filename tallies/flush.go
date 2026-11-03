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

// Flush writes the tallies of up to batchSize pending owners to the given store, returning the number of owners
// flushed.
func (t *Tallies) Flush(ctx context.Context, vc valkey.Conn, store Store, batchSize int) (int, error) {
	if batchSize <= 0 {
		panic("tallies: flush batch size must be positive")
	}

	owners, err := valkey.Int64s(valkey.DoContext(vc, ctx, "SRANDMEMBER", t.pendingKey(), batchSize))
	if err != nil {
		return 0, fmt.Errorf("error getting pending owners: %w", err)
	}

	var errs []error
	for _, owner := range owners {
		if err := t.flushOwner(ctx, vc, store, OwnerID(owner)); err != nil {
			errs = append(errs, fmt.Errorf("error flushing tallies for owner %d: %w", owner, err))
		}
	}

	return len(owners) - len(errs), errors.Join(errs...)
}

func (t *Tallies) flushOwner(ctx context.Context, vc valkey.Conn, store Store, owner OwnerID) error {
	lastGen, err := store.LastGeneration(ctx, owner)
	if err != nil {
		return fmt.Errorf("error getting last generation: %w", err)
	}

	moved, err := valkey.Values(scriptMove.DoContext(ctx, vc,
		t.liveKey(owner), t.liveExpiriesKey(owner), t.flushingKey(owner), t.flushingExpiriesKey(owner), lastGen,
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

		if err := store.Write(ctx, owner, gen, amounts, expiries); err != nil {
			return fmt.Errorf("error writing tallies: %w", err)
		}
	}

	_, err = scriptDone.DoContext(
		ctx, vc, t.liveKey(owner), t.flushingKey(owner), t.flushingExpiriesKey(owner), t.pendingKey(), owner,
	)
	if err != nil {
		return fmt.Errorf("error deleting flushing hashes: %w", err)
	}
	return nil
}
