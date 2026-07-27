package queues

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"regexp"

	valkey "github.com/gomodule/redigo/redis"
)

// FairV2 implements a fair queue where tasks are distributed evenly across owners. Unlike FairV3 it doesn't lease
// tasks to consumers: a popped task only exists as an active count against its owner, which consumers decrement by
// calling Done, so a consumer dying mid-task loses the task. It is retained until consumers are ready for FairV3's
// at-least-once delivery semantics.
//
// A queue with base key "foo" and owners "owner1" and "owner2" will have the following keys:
//   - {foo}:queued - set of owners scored by number of queued tasks
//   - {foo}:active - set of owners scored by number of active tasks
//   - {foo}:paused - set of paused owners
//   - {foo}:temp - used internally
//   - {foo}:o:owner1/0 - e.g. list of tasks for owner1 with priority 0 (low)
//   - {foo}:o:owner1/1 - e.g. list of tasks for owner1 with priority 1 (high)
//   - {foo}:o:owner2/0 - e.g. list of tasks for owner2 with priority 0 (low)
//   - {foo}:o:owner2/1 - e.g. list of tasks for owner2 with priority 1 (high)
//
// Note: it would be nice if owner queues could use distict hash tags and so live on different nodes in a cluster, but
// our push and pop scripts require atomic changes to the queued/active sets and the task lists.
type FairV2 struct {
	keyBase           string
	maxActivePerOwner int // max number of active tasks per owner
}

// NewFairV2 creates a new fair queue with the given key base.
func NewFairV2(keyBase string, maxActivePerOwner int) *FairV2 {
	return &FairV2{keyBase: keyBase, maxActivePerOwner: maxActivePerOwner}
}

//go:embed lua/fair2_push.lua
var luaFair2Push string
var scriptFair2Push = valkey.NewScript(4, luaFair2Push)

// Push adds the passed in task to our queue for execution
func (q *FairV2) Push(ctx context.Context, vc valkey.Conn, owner OwnerID, priority bool, task []byte) (TaskID, error) {
	id := newTaskID()

	// prepend UUID to the task
	var payload bytes.Buffer
	payload.WriteString(string(id))
	payload.WriteByte('|')
	payload.Write(task)

	queueKeys := q.queueKeys(owner)

	_, err := scriptFair2Push.Do(vc, q.queuedKey(), q.activeKey(), queueKeys[0], queueKeys[1], owner, priority, payload.Bytes())
	if err != nil {
		return "", fmt.Errorf("error pushing task for owner %s: %w", owner, err)
	}
	return id, nil
}

//go:embed lua/fair2_pop_owner.lua
var luaFair2PopOwner string
var scriptFair2PopOwner = valkey.NewScript(4, luaFair2PopOwner)

//go:embed lua/fair2_pop_task.lua
var luaFair2PopTask string
var scriptFair2PopTask = valkey.NewScript(3, luaFair2PopTask)

// Pop pops the next task off our queue
func (q *FairV2) Pop(ctx context.Context, vc valkey.Conn) (TaskID, OwnerID, []byte, error) {
	for {
		// Select an owner with queued tasks
		owner, err := valkey.String(scriptFair2PopOwner.DoContext(ctx, vc, q.queuedKey(), q.activeKey(), q.pausedKey(), q.tempKey(), q.maxActivePerOwner))
		if err != nil {
			return "", "", nil, fmt.Errorf("error selecting task owner: %w", err)
		}
		if owner == "" { // None found so no tasks to pop
			return "", "", nil, nil
		}

		// Pop a task for the owner
		queueKeys := q.queueKeys(OwnerID(owner))
		payload, err := valkey.String(scriptFair2PopTask.DoContext(ctx, vc, q.activeKey(), queueKeys[0], queueKeys[1], owner))
		if err != nil {
			return "", "", nil, fmt.Errorf("error popping task for owner %s: %w", owner, err)
		}
		if payload != "" {
			id, task, err := parsePayload([]byte(payload))
			if err != nil {
				return "", "", nil, fmt.Errorf("error parsing task payload for owner %s: %w", owner, err)
			}

			return id, OwnerID(owner), task, nil
		}

		// It's possible that we selected an owner with no tasks, so go back around again
	}
}

//go:embed lua/fair2_done.lua
var luaFair2Done string
var scriptFair2Done = valkey.NewScript(1, luaFair2Done)

// Done marks the passed in task as complete. Callers must call this in order
// to maintain fair workers across orgs
func (q *FairV2) Done(ctx context.Context, vc valkey.Conn, owner OwnerID) error {
	_, err := scriptFair2Done.Do(vc, q.activeKey(), owner)
	if err != nil {
		return fmt.Errorf("error marking task done for owner %s: %w", owner, err)
	}
	return nil
}

// Pause marks the given owner as paused, disabling processing of their tasks
func (q *FairV2) Pause(ctx context.Context, vc valkey.Conn, owner OwnerID) error {
	_, err := valkey.DoContext(vc, ctx, "SADD", q.pausedKey(), owner)
	return err
}

// Resume unmarks the given owner as paused, re-enabling processing of their tasks
func (q *FairV2) Resume(ctx context.Context, vc valkey.Conn, owner OwnerID) error {
	_, err := valkey.DoContext(vc, ctx, "SREM", q.pausedKey(), owner)
	return err
}

// Paused returns the list of owners marked as paused
func (q *FairV2) Paused(ctx context.Context, vc valkey.Conn) ([]OwnerID, error) {
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
func (q *FairV2) Queued(ctx context.Context, vc valkey.Conn) ([]OwnerID, error) {
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
func (q *FairV2) Size(ctx context.Context, vc valkey.Conn, owner OwnerID) (int, error) {
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

//go:embed lua/fair2_dump.lua
var luaFair2Dump string
var scriptFair2Dump = valkey.NewScript(3, luaFair2Dump)

func (q *FairV2) Dump(ctx context.Context, vc valkey.Conn) ([]byte, error) {
	dump, err := valkey.Bytes(scriptFair2Dump.Do(vc, q.queuedKey(), q.activeKey(), q.pausedKey()))
	if err != nil {
		return nil, fmt.Errorf("error dumping queue state: %w", err)
	}

	return dump, nil
}

func (q *FairV2) queuedKey() string {
	return fmt.Sprintf("{%s}:queued", q.keyBase)
}

func (q *FairV2) activeKey() string {
	return fmt.Sprintf("{%s}:active", q.keyBase)
}

func (q *FairV2) pausedKey() string {
	return fmt.Sprintf("{%s}:paused", q.keyBase)
}

func (q *FairV2) tempKey() string {
	return fmt.Sprintf("{%s}:temp", q.keyBase)
}

func (q *FairV2) queueKeys(owner OwnerID) [2]string {
	return [2]string{
		fmt.Sprintf("{%s}:o:%s/0", q.keyBase, owner),
		fmt.Sprintf("{%s}:o:%s/1", q.keyBase, owner),
	}
}

var idRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-7][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func parsePayload(raw []byte) (TaskID, []byte, error) {
	if len(raw) == 0 {
		return "", nil, fmt.Errorf("empty task payload")
	}

	parts := bytes.SplitN(raw, []byte{'|'}, 2)
	if len(parts) != 2 || !idRegex.Match(parts[0]) {
		return "", nil, fmt.Errorf("invalid task payload: %s", raw)
	}

	id := TaskID(parts[0])
	task := parts[1]

	return id, task, nil
}
