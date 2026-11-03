// Package tallies buffers per-owner totals of things which only go up in Valkey and periodically flushes them to a
// store, so that frequent increments cost one write per owner per flush rather than a write each.
//
// Keys, for a key base of foo:
//   - {foo}:pending - set of owners with buffered tallies
//   - {foo}:<owner> - hash of scope to amount, incremented by writers
//   - {foo}:<owner>:expiries - hash of scope to expiry date, set by the first write of each scope with one
//   - {foo}:<owner>:flushing - hash being flushed, with a _gen field holding its flush generation
//   - {foo}:<owner>:expiries:flushing - expiries being flushed with it
//
// Everything which increments or flushes the same tallies must use the same key base.
//
// Scopes separate their parts with / and end with @ and the day if they're daily, e.g. foo/bar@2025-05-04.
//
// Each writer decides how long its own scopes are kept by giving them an expiry date. Only the first expiry given for a
// scope since it was last flushed is kept, and a store applies it when it first creates the scope's total, so later
// expiries for a scope it already has are ignored. A write without an expiry doesn't stop another write of the same
// scope from giving one, so a scope should only be written by code which agrees on its expiry.
//
// There must only be one flush of the same tallies at a time, so a flush must be cancelled before any lock which
// ensures that expires.
package tallies

import (
	"context"
	"fmt"
	"strings"
	"time"

	valkey "github.com/gomodule/redigo/redis"
)

const generationField = "_gen"

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

func (t *Tallies) pendingKey() string                   { return fmt.Sprintf("{%s}:pending", t.keyBase) }
func (t *Tallies) liveKey(owner OwnerID) string         { return fmt.Sprintf("{%s}:%d", t.keyBase, owner) }
func (t *Tallies) liveExpiriesKey(owner OwnerID) string { return t.liveKey(owner) + ":expiries" }
func (t *Tallies) flushingKey(owner OwnerID) string     { return t.liveKey(owner) + ":flushing" }
func (t *Tallies) flushingExpiriesKey(owner OwnerID) string {
	return t.liveExpiriesKey(owner) + ":flushing"
}

// Key is a scope to increment and when its total expires.
type Key struct {
	Scope     string
	ExpiresOn string // as YYYY-MM-DD, or empty if none is given
}

// Daily returns the key for the total of the given scope on the UTC day of the given time, which expires the given
// period after that day, or has no expiry if it's zero. Panics if the given scope contains @ as it's reserved for
// separating the day.
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

// Total returns the key for the all-time total of the given scope, which has no expiry.
func Total(scope string) Key {
	return Key{Scope: scope}
}

// Increment adds the given amounts to the tallies of the given owner.
func (t *Tallies) Increment(ctx context.Context, vc valkey.Conn, owner OwnerID, amounts map[Key]int64) error {
	if len(amounts) == 0 {
		return nil
	}
	expiries := make(map[string]string, len(amounts))
	for key := range amounts {
		if key.Scope == generationField {
			return fmt.Errorf("%s is a reserved scope", generationField)
		}
		if key.ExpiresOn != "" {
			if _, err := time.Parse(time.DateOnly, key.ExpiresOn); err != nil {
				return fmt.Errorf("scope %s has an invalid expiry %q", key.Scope, key.ExpiresOn)
			}
		}
		if expiresOn, seen := expiries[key.Scope]; seen && expiresOn != key.ExpiresOn {
			return fmt.Errorf("scope %s is given different expiries", key.Scope)
		}
		expiries[key.Scope] = key.ExpiresOn
	}

	vc.Send("MULTI")
	for key, amount := range amounts {
		vc.Send("HINCRBY", t.liveKey(owner), key.Scope, amount)
		if key.ExpiresOn != "" {
			vc.Send("HSETNX", t.liveExpiriesKey(owner), key.Scope, key.ExpiresOn)
		}
	}
	vc.Send("SADD", t.pendingKey(), owner)

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
