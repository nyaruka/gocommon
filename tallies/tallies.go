// Package tallies buffers per-owner totals of things which only go up in Valkey and periodically flushes them to a
// store, so that frequent increments cost one write per owner per flush rather than a write each.
//
// Keys:
//   - tallies:pending - set of owners with buffered tallies
//   - tallies:<owner> - hash of scope to amount, incremented by writers
//   - tallies:<owner>:flushing - hash being flushed, with a _gen field holding its flush generation
//
// Scopes separate their parts with / and end with @ and the day if they're daily, e.g. foo/bar@2025-05-04.
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

func liveKey(owner OwnerID) string     { return fmt.Sprintf("tallies:%d", owner) }
func flushingKey(owner OwnerID) string { return liveKey(owner) + ":flushing" }

// Daily returns the scope for the total of the given scope on the UTC day of the given time. Panics if the given scope
// contains @ as it's reserved for separating the day.
func Daily(scope string, t time.Time) string {
	if strings.Contains(scope, "@") {
		panic(fmt.Sprintf("tallies: scope %q can't contain @", scope))
	}
	return scope + "@" + t.UTC().Format(time.DateOnly)
}

// ParseDaily splits a scope returned by Daily into its base scope and day, with ok false if it isn't one.
func ParseDaily(scope string) (string, time.Time, bool) {
	base, date, found := strings.Cut(scope, "@")
	if !found || base == "" {
		return "", time.Time{}, false
	}
	day, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return "", time.Time{}, false
	}
	return base, day, true
}

// Increment adds the given amounts to the tallies of the given owner.
func Increment(ctx context.Context, vc valkey.Conn, owner OwnerID, amounts map[string]int64) error {
	if len(amounts) == 0 {
		return nil
	}
	if _, ok := amounts[generationField]; ok {
		return fmt.Errorf("%s is a reserved scope", generationField)
	}

	vc.Send("MULTI")
	for scope, amount := range amounts {
		vc.Send("HINCRBY", liveKey(owner), scope, amount)
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
