// Package tallies buffers per-owner totals of things which only go up in Valkey and periodically flushes them to a
// store, so that frequent increments cost one write per owner per flush rather than a write each.
//
// Keys:
//   - tallies:pending - set of owners with buffered tallies
//   - tallies:<owner> - hash of scope to amount, incremented by writers
//   - tallies:<owner>:expiries - hash of scope to expiry date, set by the first write of each scope with one
//   - tallies:<owner>:flushing - hash being flushed, with a _gen field holding its flush generation
//   - tallies:<owner>:expiries:flushing - expiries being flushed with it
//
// Scopes separate their parts with / and end with @ and the day if they're daily, e.g. foo/bar@2025-05-04.
//
// Each writer decides how long its own scopes are kept by giving them an expiry date. Only the first expiry given for a
// scope since it was last flushed is kept, and a store applies it when it first creates the scope's total, so later
// expiries for a scope it already has are ignored.
//
// There must only be one flusher at a time, so a flush must be cancelled before any lock which ensures that expires.
package tallies

import (
	"context"
	"fmt"
	"strings"
	"time"

	valkey "github.com/gomodule/redigo/redis"
)

const (
	pendingKey      = "tallies:pending"
	generationField = "_gen"
)

// OwnerID is the identifier of the owner that tallies belong to.
type OwnerID int64

func liveKey(owner OwnerID) string             { return fmt.Sprintf("tallies:%d", owner) }
func liveExpiriesKey(owner OwnerID) string     { return liveKey(owner) + ":expiries" }
func flushingKey(owner OwnerID) string         { return liveKey(owner) + ":flushing" }
func flushingExpiriesKey(owner OwnerID) string { return liveExpiriesKey(owner) + ":flushing" }

// Key is a scope to increment and when its total expires.
type Key struct {
	Scope     string
	ExpiresOn string // as YYYY-MM-DD, or empty if it's kept forever
}

// Daily returns the key for the total of the given scope on the UTC day of the given time, which expires the given
// period after that day, or never if it's zero. Panics if the given scope contains @ as it's reserved for separating
// the day.
func Daily(scope string, t time.Time, ttl time.Duration) Key {
	if strings.Contains(scope, "@") {
		panic(fmt.Sprintf("tallies: scope %q can't contain @", scope))
	}

	day := t.UTC().Truncate(24 * time.Hour)
	key := Key{Scope: scope + "@" + day.Format(time.DateOnly)}
	if ttl != 0 {
		key.ExpiresOn = day.Add(ttl).Format(time.DateOnly)
	}
	return key
}

// Total returns the key for the all-time total of the given scope, which is kept forever.
func Total(scope string) Key {
	return Key{Scope: scope}
}

// Increment adds the given amounts to the tallies of the given owner.
func Increment(ctx context.Context, vc valkey.Conn, owner OwnerID, amounts map[Key]int64) error {
	if len(amounts) == 0 {
		return nil
	}
	for key := range amounts {
		if key.Scope == generationField {
			return fmt.Errorf("%s is a reserved scope", generationField)
		}
	}

	vc.Send("MULTI")
	for key, amount := range amounts {
		vc.Send("HINCRBY", liveKey(owner), key.Scope, amount)
		if key.ExpiresOn != "" {
			vc.Send("HSETNX", liveExpiriesKey(owner), key.Scope, key.ExpiresOn)
		}
	}
	vc.Send("SADD", pendingKey, owner)

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
