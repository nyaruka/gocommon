package queues

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	valkey "github.com/gomodule/redigo/redis"
)

// the most waiting owners that a single pop will make ready
const msgsV2WakeLimit = 100

// MsgsV2 implements a queue of outgoing messages where each owner (i.e. a channel) has its own TPS limit, and owners
// with queued messages are popped from fairly by the number of their sends in flight, taking turns when that's equal.
//
// A queue with base key "foo" and owner "owner1" will have the following keys:
//   - {foo}:ready - zset of owners that can be popped from, scored by number of sends in flight then last pop time
//   - {foo}:waiting - zset of owners waiting on their TPS or a scheduled item, scored by the time they can be woken
//   - {foo}:ch:owner1 - hash of owner1's state: tps, debt_until, inflight, last_pop and paused_until
//   - {foo}:q:owner1/0 - e.g. zset of bulk priority items for owner1, scored by when they can be popped
//   - {foo}:q:owner1/1 - e.g. zset of high priority items for owner1, scored by when they can be popped
//   - {foo}:q:owner1/2 - e.g. zset of requeued items for owner1, popped before anything else
//
// Items are JSON arrays of messages for a single contact. A pop takes the first message and schedules the rest a few
// seconds out so they're sent in order.
//
// An owner's TPS is charged with the cost of each send, which only the sender knows, so is passed to Done. Charges
// accrue as debt, stored as the time it will have drained by at the owner's TPS. So that sends in flight count against
// the owner, a pop charges an estimated cost of 1 which Done swaps for the actual cost, and an owner can be popped from
// while its debt with that estimate is no more than a second's worth.
type MsgsV2 struct {
	keyBase string
}

// NewMsgsV2 creates a new msgs queue with the given key base.
func NewMsgsV2(keyBase string) *MsgsV2 {
	return &MsgsV2{keyBase: fmt.Sprintf("{%s}", keyBase)}
}

//go:embed lua/msgs2_throttle.lua
var luaMsgs2Throttle string

//go:embed lua/msgs2_push.lua
var luaMsgs2Push string
var scriptMsgs2Push = valkey.NewScript(4, luaMsgs2Throttle+luaMsgs2Push)

// Push adds the passed in batch, a JSON array, to the owner's queue. A tps of 0 means no limit, and the latest push
// sets the TPS for everything queued for the owner.
func (q *MsgsV2) Push(ctx context.Context, vc valkey.Conn, owner OwnerID, tps int, priority bool, batch []byte) error {
	_, err := scriptMsgs2Push.DoContext(ctx, vc, q.queueKey(owner, priority), q.channelKey(owner), q.readyKey(), q.waitingKey(), owner, tps, batch, timeNow().UnixMicro())
	if err != nil {
		return fmt.Errorf("error pushing batch for owner %s: %w", owner, err)
	}
	return nil
}

//go:embed lua/msgs2_pop.lua
var luaMsgs2Pop string
var scriptMsgs2Pop = valkey.NewScript(2, luaMsgs2Throttle+luaMsgs2Pop)

// Pop pops the next message off our queue, returning an empty owner if there's nothing to pop. Callers must call Done
// with the owner and the cost of the send when they've sent it.
func (q *MsgsV2) Pop(ctx context.Context, vc valkey.Conn) (OwnerID, []byte, error) {
	for {
		vals, err := valkey.Values(scriptMsgs2Pop.DoContext(ctx, vc, q.readyKey(), q.waitingKey(), q.keyBase, msgsV2WakeLimit, timeNow().UnixMicro()))
		if err != nil {
			return "", nil, fmt.Errorf("error popping message: %w", err)
		}

		var owner string
		var msg []byte
		if _, err := valkey.Scan(vals, &owner, &msg); err != nil {
			return "", nil, fmt.Errorf("error scanning pop result: %w", err)
		}

		switch owner {
		case "empty":
			return "", nil, nil
		case "retry": // selected owner was throttled or had nothing to pop, and has been taken out of ready so try again
			continue
		}

		return OwnerID(owner), msg, nil
	}
}

//go:embed lua/msgs2_done.lua
var luaMsgs2Done string
var scriptMsgs2Done = valkey.NewScript(4, luaMsgs2Throttle+luaMsgs2Done)

// Done marks a message popped for the given owner as sent, charging the cost of sending it against the owner's TPS.
// What a unit of cost is depends on what the owner's TPS limits, e.g. segments or API requests, and a send that
// reached no provider costs 0. Negative costs count as 0. Callers must call this, or Requeue, for every message they
// pop in order to maintain fair distribution across owners.
func (q *MsgsV2) Done(ctx context.Context, vc valkey.Conn, owner OwnerID, cost int) error {
	_, err := scriptMsgs2Done.DoContext(ctx, vc, q.channelKey(owner), q.readyKey(), q.waitingKey(), q.requeueKey(owner), owner, cost, timeNow().UnixMicro())
	if err != nil {
		return fmt.Errorf("error marking message done for owner %s: %w", owner, err)
	}
	return nil
}

// Requeue is called instead of Done for a message popped for the given owner that was never handed to a provider. It
// does what Done does with a cost of 0, and puts back the passed in batch, a JSON array as taken by Push, to be popped
// before anything else queued for the owner. Callers that want it to wait, e.g. for a provider to recover, should pause
// the owner first. The rest of a popped batch is popped a few seconds later regardless, so a message requeued after
// that may be sent after messages that followed it. Requeuing a batch identical to one already requeued just moves that
// one to the back. Older versions don't pop requeued items, and can leave them unpopped until the owner is next pushed
// to, so every app that pops should run this version before any app requeues.
func (q *MsgsV2) Requeue(ctx context.Context, vc valkey.Conn, owner OwnerID, batch []byte) error {
	var msgs []json.RawMessage
	if err := json.Unmarshal(batch, &msgs); err != nil || len(msgs) == 0 {
		return errors.New("requeued batch must be a non-empty JSON array")
	}

	_, err := scriptMsgs2Done.DoContext(ctx, vc, q.channelKey(owner), q.readyKey(), q.waitingKey(), q.requeueKey(owner), owner, 0, timeNow().UnixMicro(), batch)
	if err != nil {
		return fmt.Errorf("error requeuing batch for owner %s: %w", owner, err)
	}
	return nil
}

//go:embed lua/msgs2_pause.lua
var luaMsgs2Pause string
var scriptMsgs2Pause = valkey.NewScript(3, luaMsgs2Pause)

// Pause stops messages being popped for the given owner until the given time, e.g. to back off a channel whose provider
// is overloaded. A pause can only be extended, and it ends on its own.
func (q *MsgsV2) Pause(ctx context.Context, vc valkey.Conn, owner OwnerID, until time.Time) error {
	_, err := scriptMsgs2Pause.DoContext(ctx, vc, q.channelKey(owner), q.readyKey(), q.waitingKey(), owner, until.UnixMicro(), timeNow().UnixMicro())
	if err != nil {
		return fmt.Errorf("error pausing owner %s: %w", owner, err)
	}
	return nil
}

//go:embed lua/msgs2_clear.lua
var luaMsgs2Clear string
var scriptMsgs2Clear = valkey.NewScript(6, luaMsgs2Clear)

// Clear removes everything queued for the given owner. A pause outlives it.
func (q *MsgsV2) Clear(ctx context.Context, vc valkey.Conn, owner OwnerID) error {
	_, err := scriptMsgs2Clear.DoContext(ctx, vc, q.requeueKey(owner), q.queueKey(owner, true), q.queueKey(owner, false), q.channelKey(owner), q.readyKey(), q.waitingKey(), owner, timeNow().UnixMicro())
	if err != nil {
		return fmt.Errorf("error clearing queue for owner %s: %w", owner, err)
	}
	return nil
}

// Size returns the number of high priority, bulk priority and requeued items queued across all owners. It reads every
// owner and isn't atomic, so is meant for metrics.
func (q *MsgsV2) Size(ctx context.Context, vc valkey.Conn) (high, bulk, requeued int, err error) {
	var owners []OwnerID

	for _, set := range []string{q.readyKey(), q.waitingKey()} {
		members, err := valkey.Strings(valkey.DoContext(vc, ctx, "ZRANGE", set, 0, -1))
		if err != nil {
			return 0, 0, 0, err
		}
		for _, m := range members {
			owners = append(owners, OwnerID(m))
		}
	}

	// counted in a transaction so that a reply error, e.g. WRONGTYPE, doesn't leave other replies unread on vc
	vc.Send("MULTI")
	for _, owner := range owners {
		vc.Send("ZCARD", q.queueKey(owner, true))
		vc.Send("ZCARD", q.queueKey(owner, false))
		vc.Send("ZCARD", q.requeueKey(owner))
	}
	counts, err := valkey.Ints(valkey.DoContext(vc, ctx, "EXEC"))
	if err != nil {
		return 0, 0, 0, err
	}

	for i := 0; i < len(counts); i += 3 {
		high += counts[i]
		bulk += counts[i+1]
		requeued += counts[i+2]
	}
	return high, bulk, requeued, nil
}

func (q *MsgsV2) readyKey() string {
	return q.keyBase + ":ready"
}

func (q *MsgsV2) waitingKey() string {
	return q.keyBase + ":waiting"
}

func (q *MsgsV2) channelKey(owner OwnerID) string {
	return fmt.Sprintf("%s:ch:%s", q.keyBase, owner)
}

func (q *MsgsV2) queueKey(owner OwnerID, priority bool) string {
	if priority {
		return fmt.Sprintf("%s:q:%s/1", q.keyBase, owner)
	}
	return fmt.Sprintf("%s:q:%s/0", q.keyBase, owner)
}

func (q *MsgsV2) requeueKey(owner OwnerID) string {
	return fmt.Sprintf("%s:q:%s/2", q.keyBase, owner)
}
