package queues

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"time"

	valkey "github.com/gomodule/redigo/redis"
)

// Fair implements a fair queue where tasks are distributed evenly across owners, and tasks are leased to consumers
// so that tasks whose consumers die can be redelivered.
//
// A queue with base key "foo" and owners "owner1" and "owner2" will have the following keys:
//   - {foo}:queued - zset of owners scored by number of queued tasks
//   - {foo}:active - zset of owners scored by number of in-flight tasks
//   - {foo}:paused - set of paused owners
//   - {foo}:inflight - hash of in-flight task IDs to records of those tasks
//   - {foo}:expires - zset of in-flight task IDs scored by lease expiry time
//   - {foo}:dead - list of tasks which exceeded the max number of delivery attempts
//   - {foo}:temp - used internally
//   - {foo}:o:owner1/0 - e.g. list of tasks for owner1 with priority 0 (low)
//   - {foo}:o:owner1/1 - e.g. list of tasks for owner1 with priority 1 (high)
//   - {foo}:o:owner2/0 - e.g. list of tasks for owner2 with priority 0 (low)
//   - {foo}:o:owner2/1 - e.g. list of tasks for owner2 with priority 1 (high)
//
// Every popped task is recorded as in-flight with a lease. Consumers must call Done when a task completes, and can
// call Extend if they need to hold a task for longer than the lease duration. If a consumer dies without calling
// Done, the task's lease eventually expires and the task is redelivered to a subsequent caller of Pop. Tasks which
// have been delivered the max number of attempts are instead moved to the dead list. Delivery is thus at-least-once:
// consumers can see the same task more than once if a previous consumer died holding it or ran past its lease.
//
// Note: it would be nice if owner queues could use distict hash tags and so live on different nodes in a cluster, but
// our push and pop scripts require atomic changes to the queued/active sets and the task lists.
type Fair struct {
	keyBase           string
	maxActivePerOwner int           // max number of active tasks per owner
	lease             time.Duration // how long a popped task remains in-flight before it can be redelivered
	maxAttempts       int           // max number of times a task can be delivered before it is moved to the dead list
}

// NewFair creates a new fair queue with the given key base.
func NewFair(keyBase string, maxActivePerOwner int, lease time.Duration, maxAttempts int) *Fair {
	return &Fair{keyBase: keyBase, maxActivePerOwner: maxActivePerOwner, lease: lease, maxAttempts: maxAttempts}
}

//go:embed lua/fair_push.lua
var luaFairPush string
var scriptFairPush = valkey.NewScript(4, luaFairPush)

// Push adds the passed in task to our queue for execution. Note that owner IDs must not contain '|'.
func (q *Fair) Push(ctx context.Context, vc valkey.Conn, owner OwnerID, priority bool, task []byte) (TaskID, error) {
	id := newTaskID()

	// prepend UUID to the task
	var payload bytes.Buffer
	payload.WriteString(string(id))
	payload.WriteByte('|')
	payload.Write(task)

	queueKeys := q.queueKeys(owner)

	_, err := scriptFairPush.DoContext(ctx, vc, q.queuedKey(), q.activeKey(), queueKeys[0], queueKeys[1], owner, priority, payload.Bytes())
	if err != nil {
		return "", fmt.Errorf("error pushing task for owner %s: %w", owner, err)
	}
	return id, nil
}

// PoppedTask is a task delivered to a consumer for processing.
type PoppedTask struct {
	ID       TaskID
	Owner    OwnerID
	Attempts int // number of times this task has been delivered, i.e. 1 for a first delivery
	Task     []byte
}

//go:embed lua/fair_pop.lua
var luaFairPop string
var scriptFairPop = valkey.NewScript(7, luaFairPop)

// Pop pops the next task off our queue, prioritizing redelivery of in-flight tasks whose leases have expired.
// Returns nil if there are no tasks to pop.
func (q *Fair) Pop(ctx context.Context, vc valkey.Conn) (*PoppedTask, error) {
	for {
		now := timeNow()
		reply, err := scriptFairPop.DoContext(ctx, vc,
			q.queuedKey(), q.activeKey(), q.pausedKey(), q.tempKey(), q.inflightKey(), q.expiresKey(), q.deadKey(),
			q.keyBase, q.maxActivePerOwner, now.UnixMilli(), now.Add(q.lease).UnixMilli(), q.maxAttempts,
		)
		if err != nil {
			return nil, fmt.Errorf("error popping task: %w", err)
		}
		if reply == nil { // no owners with queued tasks
			return nil, nil
		}

		vals, err := valkey.Values(reply, nil)
		if err != nil {
			return nil, fmt.Errorf("error reading pop result: %w", err)
		}
		if len(vals) == 0 {
			continue // selected owner turned out to have no queued tasks, try again
		}

		var id, owner string
		var attempts int
		var task []byte
		if _, err := valkey.Scan(vals, &id, &owner, &attempts, &task); err != nil {
			return nil, fmt.Errorf("error scanning pop result: %w", err)
		}

		return &PoppedTask{ID: TaskID(id), Owner: OwnerID(owner), Attempts: attempts, Task: task}, nil
	}
}

//go:embed lua/fair_done.lua
var luaFairDone string
var scriptFairDone = valkey.NewScript(3, luaFairDone)

// Done marks the passed in task as complete, releasing its lease. Callers must call this for every task they pop in
// order to maintain fair distribution across owners. Calling it for a task whose lease already expired is a no-op.
func (q *Fair) Done(ctx context.Context, vc valkey.Conn, id TaskID) error {
	_, err := scriptFairDone.DoContext(ctx, vc, q.activeKey(), q.inflightKey(), q.expiresKey(), string(id))
	if err != nil {
		return fmt.Errorf("error marking task %s done: %w", id, err)
	}
	return nil
}

//go:embed lua/fair_extend.lua
var luaFairExtend string
var scriptFairExtend = valkey.NewScript(1, luaFairExtend)

// Extend renews the lease on the given in-flight task for the given duration from now, returning whether the task
// was still leased. Consumers holding tasks for longer than the queue's lease duration should call this periodically
// to prevent redelivery.
func (q *Fair) Extend(ctx context.Context, vc valkey.Conn, id TaskID, dur time.Duration) (bool, error) {
	extended, err := valkey.Int(scriptFairExtend.DoContext(ctx, vc, q.expiresKey(), timeNow().Add(dur).UnixMilli(), string(id)))
	if err != nil {
		return false, fmt.Errorf("error extending lease for task %s: %w", id, err)
	}
	return extended == 1, nil
}

//go:embed lua/fair_reconcile.lua
var luaFairReconcile string
var scriptFairReconcile = valkey.NewScript(2, luaFairReconcile)

// Reconcile rebuilds the active counts from the in-flight records, healing any drift such as counts left behind by
// consumer processes which died. Consumers should call this periodically.
func (q *Fair) Reconcile(ctx context.Context, vc valkey.Conn) error {
	_, err := scriptFairReconcile.DoContext(ctx, vc, q.activeKey(), q.inflightKey())
	if err != nil {
		return fmt.Errorf("error reconciling active counts: %w", err)
	}
	return nil
}

// Pause marks the given owner as paused, disabling processing of their tasks
func (q *Fair) Pause(ctx context.Context, vc valkey.Conn, owner OwnerID) error {
	_, err := valkey.DoContext(vc, ctx, "SADD", q.pausedKey(), owner)
	return err
}

// Resume unmarks the given owner as paused, re-enabling processing of their tasks
func (q *Fair) Resume(ctx context.Context, vc valkey.Conn, owner OwnerID) error {
	_, err := valkey.DoContext(vc, ctx, "SREM", q.pausedKey(), owner)
	return err
}

// Paused returns the list of owners marked as paused
func (q *Fair) Paused(ctx context.Context, vc valkey.Conn) ([]OwnerID, error) {
	strs, err := valkey.Strings(valkey.DoContext(vc, ctx, "SMEMBERS", q.pausedKey()))
	if err != nil {
		return nil, err
	}

	owners := make([]OwnerID, len(strs))
	for i, str := range strs {
		owners[i] = OwnerID(str)
	}

	return owners, nil
}

// Queued returns the list of owners with queued tasks
func (q *Fair) Queued(ctx context.Context, vc valkey.Conn) ([]OwnerID, error) {
	strs, err := valkey.Strings(valkey.DoContext(vc, ctx, "ZRANGE", q.queuedKey(), 0, -1))
	if err != nil {
		return nil, err
	}

	owners := make([]OwnerID, len(strs))
	for i, str := range strs {
		owners[i] = OwnerID(str)
	}

	return owners, nil
}

// Size returns the number of queued tasks for the given owner
func (q *Fair) Size(ctx context.Context, vc valkey.Conn, owner OwnerID) (int, error) {
	queueKeys := q.queueKeys(owner)

	vc.Send("MULTI")
	vc.Send("LLEN", queueKeys[0])
	vc.Send("LLEN", queueKeys[1])
	counts, err := valkey.Ints(valkey.DoContext(vc, ctx, "EXEC"))
	if err != nil {
		return 0, err
	}

	return counts[0] + counts[1], nil
}

//go:embed lua/fair_dump.lua
var luaFairDump string
var scriptFairDump = valkey.NewScript(5, luaFairDump)

func (q *Fair) Dump(ctx context.Context, vc valkey.Conn) ([]byte, error) {
	dump, err := valkey.Bytes(scriptFairDump.DoContext(ctx, vc, q.queuedKey(), q.activeKey(), q.pausedKey(), q.inflightKey(), q.deadKey()))
	if err != nil {
		return nil, fmt.Errorf("error dumping queue state: %w", err)
	}

	return dump, nil
}

func (q *Fair) queuedKey() string {
	return fmt.Sprintf("{%s}:queued", q.keyBase)
}

func (q *Fair) activeKey() string {
	return fmt.Sprintf("{%s}:active", q.keyBase)
}

func (q *Fair) pausedKey() string {
	return fmt.Sprintf("{%s}:paused", q.keyBase)
}

func (q *Fair) inflightKey() string {
	return fmt.Sprintf("{%s}:inflight", q.keyBase)
}

func (q *Fair) expiresKey() string {
	return fmt.Sprintf("{%s}:expires", q.keyBase)
}

func (q *Fair) deadKey() string {
	return fmt.Sprintf("{%s}:dead", q.keyBase)
}

func (q *Fair) tempKey() string {
	return fmt.Sprintf("{%s}:temp", q.keyBase)
}

func (q *Fair) queueKeys(owner OwnerID) [2]string {
	return [2]string{
		fmt.Sprintf("{%s}:o:%s/0", q.keyBase, owner),
		fmt.Sprintf("{%s}:o:%s/1", q.keyBase, owner),
	}
}
