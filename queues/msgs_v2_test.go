package queues_test

import (
	"fmt"
	"testing"
	"time"

	valkey "github.com/gomodule/redigo/redis"
	"github.com/nyaruka/gocommon/queues"
	"github.com/nyaruka/vkutil/assertvk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// t0 is the middle of a second, as a float so that tests can check scores
const t0 = 1767225600.5

// steps the time seen by the queue to the given number of seconds after t0
func setMsgsV2Time(secs float64) {
	queues.SetTimeNow(func() time.Time { return time.UnixMicro(int64((t0 + secs) * 1000000)) })
}

func TestMsgsV2(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	assertMsgsV2Empty(t, q, vc)

	for i := range 3 {
		assertMsgsV2Push(t, q, vc, "chan1", 0, false, fmt.Sprintf(`[{"id":%d}]`, i))
	}
	assertMsgsV2Push(t, q, vc, "chan1", 0, true, `[{"id":10}]`)

	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan1": 0})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{})
	assertvk.ZGetAll(t, vc, "{msgs}:q:chan1/1", map[string]float64{`[{"id":10}]`: t0})
	assertvk.ZCard(t, vc, "{msgs}:q:chan1/0", 3)
	assertvk.HGetAll(t, vc, "{msgs}:ch:chan1", map[string]string{"tps": "0"})
	assertMsgsV2Size(t, q, vc, 1, 3)

	// high priority first, then bulk in the order pushed
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":10}`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":0}`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)

	// ready score is the number of sends in flight
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan1": 3})
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "inflight", "3")
	assertMsgsV2Size(t, q, vc, 0, 1)

	// without a TPS limit no usage is tracked
	assertvk.HGetAll(t, vc, "{msgs}:ch:chan1", map[string]string{"tps": "0", "inflight": "3"})

	require.NoError(t, q.Done(ctx, vc, "chan1"))
	require.NoError(t, q.Done(ctx, vc, "chan1"))
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan1": 1})

	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":2}`)
	assertMsgsV2Size(t, q, vc, 0, 0)

	// still ready until a pop finds it has nothing queued
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan1": 2})
	assertMsgsV2Pop(t, q, vc, "", "")
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})
}

func TestMsgsV2Keys(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	// these keys are shared with other versions of the apps that push and pop, so mustn't change
	assertMsgsV2Push(t, q, vc, "chan1", 1, true, `[{"id":1}]`)
	assertMsgsV2Push(t, q, vc, "chan1", 1, false, `[{"id":2}]`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)
	assertMsgsV2Pop(t, q, vc, "", "") // parks chan1 as it's used its TPS

	assertvk.Keys(t, vc, "*", []string{"{msgs}:ch:chan1", "{msgs}:q:chan1/0", "{msgs}:waiting"})
	assertvk.HGetAll(t, vc, "{msgs}:ch:chan1", map[string]string{"tps": "1", "sec": "1767225600", "used": "1", "inflight": "1"})
	assertvk.ZGetAll(t, vc, "{msgs}:q:chan1/0", map[string]float64{`[{"id":2}]`: t0})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": 1767225601})

	setMsgsV2Time(1)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":2}`)
	assertvk.Keys(t, vc, "*", []string{"{msgs}:ch:chan1", "{msgs}:ready"})
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan1": 2})

	// and items written directly to those keys, as by another app, can be popped
	_, err := vc.Do("ZADD", "{msgs}:q:chan2/1", t0, `[{"id":3},{"id":4}]`)
	require.NoError(t, err)
	_, err = vc.Do("HSET", "{msgs}:ch:chan2", "tps", 10)
	require.NoError(t, err)
	_, err = vc.Do("ZADD", "{msgs}:ready", 0, "chan2")
	require.NoError(t, err)

	assertMsgsV2Pop(t, q, vc, "chan2", `{"id":3}`)
	assertvk.ZGetAll(t, vc, "{msgs}:q:chan2/1", map[string]float64{`[{"id":4}]`: t0 + 4})
	assertvk.HGetAll(t, vc, "{msgs}:ch:chan2", map[string]string{"tps": "10", "sec": "1767225601", "used": "1", "inflight": "1"})

	require.NoError(t, q.Done(ctx, vc, "chan2"))
	assertvk.HGet(t, vc, "{msgs}:ch:chan2", "inflight", "0")
}

func TestMsgsV2Batches(t *testing.T) {
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	assertMsgsV2Push(t, q, vc, "chan1", 0, false, `[{"id":1},{"id":2},{"id":3}]`)
	assertMsgsV2Push(t, q, vc, "chan1", 0, false, `[{"id":4}]`)

	// first of the batch is popped and the rest rescheduled 3s out as high priority so they're sent in order
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)
	assertvk.ZGetAll(t, vc, "{msgs}:q:chan1/1", map[string]float64{`[{"id":2},{"id":3}]`: t0 + 3})
	assertMsgsV2Size(t, q, vc, 1, 1)

	// other items can be popped in the meantime
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":4}`)

	// with only the rescheduled batch left, the owner waits until it's due
	assertMsgsV2Pop(t, q, vc, "", "")
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 3})

	setMsgsV2Time(2.9)
	assertMsgsV2Pop(t, q, vc, "", "")

	setMsgsV2Time(3)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":2}`)
	assertvk.ZGetAll(t, vc, "{msgs}:q:chan1/1", map[string]float64{`[{"id":3}]`: t0 + 6})

	setMsgsV2Time(6)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":3}`)
	assertvk.ZCard(t, vc, "{msgs}:q:chan1/1", 0)
	assertMsgsV2Pop(t, q, vc, "", "")
}

func TestMsgsV2PushWakesWaiting(t *testing.T) {
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	assertMsgsV2Push(t, q, vc, "chan1", 0, false, `[{"id":1},{"id":2}]`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)
	assertMsgsV2Pop(t, q, vc, "", "")
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 3})

	// a new push doesn't wait on the rest of the batch
	assertMsgsV2Push(t, q, vc, "chan1", 0, false, `[{"id":3}]`)
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{})
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan1": 1})

	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":3}`)
}

func TestMsgsV2Throttle(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	for i := range 5 {
		assertMsgsV2Push(t, q, vc, "chan1", 2, false, fmt.Sprintf(`[{"id":%d}]`, i))
	}

	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":0}`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)
	assertvk.HGetAll(t, vc, "{msgs}:ch:chan1", map[string]string{"tps": "2", "sec": "1767225600", "used": "2", "inflight": "2"})

	// TPS used up so the owner waits until the next second
	assertMsgsV2Pop(t, q, vc, "", "")
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": 1767225601})

	// pushing while throttled doesn't make it ready
	assertMsgsV2Push(t, q, vc, "chan1", 2, false, `[{"id":5}]`)
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": 1767225601})

	// done doesn't either
	require.NoError(t, q.Done(ctx, vc, "chan1"))
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})

	setMsgsV2Time(0.4)
	assertMsgsV2Pop(t, q, vc, "", "")

	// usage resets with the next second
	setMsgsV2Time(0.5)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":2}`)
	assertvk.HGetAll(t, vc, "{msgs}:ch:chan1", map[string]string{"tps": "2", "sec": "1767225601", "used": "1", "inflight": "2"})
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":3}`)
	assertMsgsV2Pop(t, q, vc, "", "")

	// a push with a new TPS applies to everything queued
	setMsgsV2Time(1.5)
	assertMsgsV2Push(t, q, vc, "chan1", 3, false, `[{"id":6}]`)
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan1": 3})
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":4}`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":5}`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":6}`)
	assertMsgsV2Pop(t, q, vc, "", "")
}

func TestMsgsV2PushWhenThrottled(t *testing.T) {
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	assertMsgsV2Push(t, q, vc, "chan1", 1, false, `[{"id":1}]`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)

	// a ready owner that's used up its TPS is left for a pop to park
	assertMsgsV2Push(t, q, vc, "chan1", 1, false, `[{"id":2}]`)
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan1": 1})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{})

	assertMsgsV2Pop(t, q, vc, "", "")
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": 1767225601})

	setMsgsV2Time(0.5)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":2}`)

	// an owner that isn't ready but has used up its TPS, e.g. because the TPS was lowered, is made to wait
	assertMsgsV2Push(t, q, vc, "chan2", 5, false, `[{"id":3}]`)
	assertMsgsV2Pop(t, q, vc, "chan2", `{"id":3}`)
	assertMsgsV2Pop(t, q, vc, "", "") // drains chan2 and parks chan1
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})

	assertMsgsV2Push(t, q, vc, "chan2", 1, false, `[{"id":4}]`)
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": 1767225602, "chan2": 1767225602})

	setMsgsV2Time(1.5)
	assertMsgsV2Pop(t, q, vc, "chan2", `{"id":4}`)
}

func TestMsgsV2TPSCost(t *testing.T) {
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	assertMsgsV2Push(t, q, vc, "chan1", 5, false, `[{"id":1,"tps_cost":3}]`)
	assertMsgsV2Push(t, q, vc, "chan1", 5, false, `[{"id":2,"tps_cost":0.5}]`)
	assertMsgsV2Push(t, q, vc, "chan1", 5, false, `[{"id":3,"tps_cost":"2"}]`)
	assertMsgsV2Push(t, q, vc, "chan1", 5, false, `[{"id":4}]`)

	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1,"tps_cost":3}`)
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "used", "3")

	// costs below 1 or not numbers count as 1
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":2,"tps_cost":0.5}`)
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "used", "4")
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":3,"tps_cost":"2"}`)
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "used", "5")

	assertMsgsV2Pop(t, q, vc, "", "")

	// a cost can take usage over the TPS, but only once the TPS isn't used up already
	setMsgsV2Time(1)
	assertMsgsV2Push(t, q, vc, "chan1", 5, false, `["foo"]`)
	assertMsgsV2Push(t, q, vc, "chan1", 5, false, `[{"id":5,"tps_cost":10}]`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":4}`)
	assertMsgsV2Pop(t, q, vc, "chan1", `"foo"`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":5,"tps_cost":10}`)
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "used", "12")
	assertMsgsV2Pop(t, q, vc, "", "")
}

func TestMsgsV2Fairness(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	for i := range 3 {
		assertMsgsV2Push(t, q, vc, "chan1", 0, false, fmt.Sprintf(`[{"id":%d}]`, i))
		assertMsgsV2Push(t, q, vc, "chan2", 0, false, fmt.Sprintf(`[{"id":%d}]`, 10+i))
	}

	// owners take turns as each pop puts a send in flight
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":0}`)
	assertMsgsV2Pop(t, q, vc, "chan2", `{"id":10}`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan1": 2, "chan2": 1})

	// until one completes its sends
	require.NoError(t, q.Done(ctx, vc, "chan1"))
	require.NoError(t, q.Done(ctx, vc, "chan1"))
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan1": 0, "chan2": 1})

	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":2}`)
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan1": 1, "chan2": 1})

	// a newly pushed owner has nothing in flight so goes first
	assertMsgsV2Push(t, q, vc, "chan3", 0, false, `[{"id":20}]`)
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan1": 1, "chan2": 1, "chan3": 0})
	assertMsgsV2Pop(t, q, vc, "chan3", `{"id":20}`)
}

func TestMsgsV2WakeLimit(t *testing.T) {
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	for i := range 101 {
		owner := queues.OwnerID(fmt.Sprintf("chan%03d", i))
		assertMsgsV2Push(t, q, vc, owner, 1, false, `[{"id":1}]`)
		assertMsgsV2Push(t, q, vc, owner, 1, false, `[{"id":2}]`)
	}
	for range 101 {
		_, msg, err := q.Pop(t.Context(), vc)
		require.NoError(t, err)
		assert.Equal(t, `{"id":1}`, string(msg))
	}

	// every owner has used its TPS so gets parked
	assertMsgsV2Pop(t, q, vc, "", "")
	assertvk.ZCard(t, vc, "{msgs}:ready", 0)
	assertvk.ZCard(t, vc, "{msgs}:waiting", 101)

	// a single pop only wakes so many
	setMsgsV2Time(1)
	owner, msg, err := q.Pop(t.Context(), vc)
	require.NoError(t, err)
	assert.Equal(t, queues.OwnerID("chan000"), owner)
	assert.Equal(t, `{"id":2}`, string(msg))
	assertvk.ZCard(t, vc, "{msgs}:ready", 100)
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan100": 1767225601})

	// woken owners are ready with their in-flight counts
	assertvk.ZScore(t, vc, "{msgs}:ready", "chan000", 2)
	assertvk.ZScore(t, vc, "{msgs}:ready", "chan001", 1)

	// and the next pop wakes the rest
	_, _, err = q.Pop(t.Context(), vc)
	require.NoError(t, err)
	assertvk.ZCard(t, vc, "{msgs}:waiting", 0)
}

func TestMsgsV2ForgetsLostSends(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	assertMsgsV2Push(t, q, vc, "chan1", 10, false, `[{"id":1}]`)
	assertMsgsV2Push(t, q, vc, "chan1", 10, false, `[{"id":2}]`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":2}`)
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "inflight", "2")
	assertTTL(t, vc, "{msgs}:ch:chan1", -1)

	// a pop that finds nothing queued forgets sends in flight and lets the state expire
	assertMsgsV2Pop(t, q, vc, "", "")
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})
	assertvk.HGetAll(t, vc, "{msgs}:ch:chan1", map[string]string{"tps": "10", "sec": "1767225600", "used": "2", "inflight": "0"})
	assertTTL(t, vc, "{msgs}:ch:chan1", 60)

	// sends that were still in flight complete to zero
	require.NoError(t, q.Done(ctx, vc, "chan1"))
	require.NoError(t, q.Done(ctx, vc, "chan1"))
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "inflight", "0")
	assertTTL(t, vc, "{msgs}:ch:chan1", 60)

	// TPS usage for this second survives the drain
	for i := range 9 {
		assertMsgsV2Push(t, q, vc, "chan1", 10, false, fmt.Sprintf(`[{"id":%d}]`, 10+i))
	}
	assertTTL(t, vc, "{msgs}:ch:chan1", -1)
	for i := range 8 {
		assertMsgsV2Pop(t, q, vc, "chan1", fmt.Sprintf(`{"id":%d}`, 10+i))
	}
	assertMsgsV2Pop(t, q, vc, "", "")
	assertMsgsV2Size(t, q, vc, 0, 1)
}

func TestMsgsV2StateExpiry(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	assertMsgsV2Push(t, q, vc, "chan1", 0, false, `[{"id":1}]`)
	assertMsgsV2Push(t, q, vc, "chan2", 0, false, `[{"id":2},{"id":3}]`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)
	assertMsgsV2Pop(t, q, vc, "chan2", `{"id":2}`)
	assertMsgsV2Pop(t, q, vc, "", "") // drains chan1, parks chan2 till rest of its batch is due
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan2": t0 + 3})

	// state of an owner with nothing queued expires once nothing is in flight
	_, err := vc.Do("PERSIST", "{msgs}:ch:chan1")
	require.NoError(t, err)
	require.NoError(t, q.Done(ctx, vc, "chan1"))
	assertTTL(t, vc, "{msgs}:ch:chan1", 60)

	// but not that of a waiting owner
	require.NoError(t, q.Done(ctx, vc, "chan2"))
	assertvk.HGet(t, vc, "{msgs}:ch:chan2", "inflight", "0")
	assertTTL(t, vc, "{msgs}:ch:chan2", -1)

	// or a ready one
	setMsgsV2Time(3)
	assertMsgsV2Pop(t, q, vc, "chan2", `{"id":3}`)
	require.NoError(t, q.Done(ctx, vc, "chan2"))
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan2": 0})
	assertTTL(t, vc, "{msgs}:ch:chan2", -1)

	// done for an owner with no state creates state that expires
	require.NoError(t, q.Done(ctx, vc, "chan3"))
	assertvk.HGetAll(t, vc, "{msgs}:ch:chan3", map[string]string{"inflight": "0"})
	assertTTL(t, vc, "{msgs}:ch:chan3", 60)
}

func TestMsgsV2Clear(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	assertMsgsV2Push(t, q, vc, "chan1", 1, true, `[{"id":1}]`)
	assertMsgsV2Push(t, q, vc, "chan1", 1, false, `[{"id":2}]`)
	assertMsgsV2Push(t, q, vc, "chan2", 1, false, `[{"id":3}]`)
	assertMsgsV2Push(t, q, vc, "chan3", 1, false, `[{"id":4}]`)
	assertMsgsV2Push(t, q, vc, "chan3", 1, false, `[{"id":5}]`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)
	assertMsgsV2Pop(t, q, vc, "chan2", `{"id":3}`)
	assertMsgsV2Pop(t, q, vc, "chan3", `{"id":4}`)
	assertMsgsV2Pop(t, q, vc, "", "")
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": 1767225601, "chan2": 1767225601, "chan3": 1767225601})
	assertMsgsV2Size(t, q, vc, 0, 2)

	require.NoError(t, q.Clear(ctx, vc, "chan1"))

	assertvk.NotExists(t, vc, "{msgs}:q:chan1/1")
	assertvk.NotExists(t, vc, "{msgs}:q:chan1/0")
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan2": 1767225601, "chan3": 1767225601})
	assertMsgsV2Size(t, q, vc, 0, 1)

	// state is kept for the send in flight, but expires
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "inflight", "1")
	assertTTL(t, vc, "{msgs}:ch:chan1", 60)
	require.NoError(t, q.Done(ctx, vc, "chan1"))
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "inflight", "0")

	// clearing a ready owner
	setMsgsV2Time(1)
	assertMsgsV2Push(t, q, vc, "chan2", 1, false, `[{"id":6}]`)
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan2": 1})
	require.NoError(t, q.Clear(ctx, vc, "chan2"))
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})

	// clearing an owner with nothing queued is a noop
	require.NoError(t, q.Clear(ctx, vc, "chan4"))

	// other owners unaffected
	assertMsgsV2Pop(t, q, vc, "chan3", `{"id":5}`)
	assertMsgsV2Pop(t, q, vc, "", "")
	assertMsgsV2Size(t, q, vc, 0, 0)
}

func TestMsgsV2Size(t *testing.T) {
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	assertMsgsV2Size(t, q, vc, 0, 0)

	assertMsgsV2Push(t, q, vc, "chan1", 1, true, `[{"id":1}]`)
	assertMsgsV2Push(t, q, vc, "chan1", 1, true, `[{"id":2}]`)
	assertMsgsV2Push(t, q, vc, "chan1", 1, false, `[{"id":3}]`)
	assertMsgsV2Push(t, q, vc, "chan2", 0, false, `[{"id":4},{"id":5}]`)
	assertMsgsV2Push(t, q, vc, "chan2", 0, false, `[{"id":6}]`)
	assertMsgsV2Size(t, q, vc, 2, 3)

	// items of waiting owners are counted too, and rescheduled batches count as high priority
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)
	assertMsgsV2Pop(t, q, vc, "chan2", `{"id":4}`)
	assertMsgsV2Pop(t, q, vc, "chan2", `{"id":6}`)
	assertMsgsV2Pop(t, q, vc, "", "")
	assertvk.ZCard(t, vc, "{msgs}:waiting", 2)
	assertMsgsV2Size(t, q, vc, 2, 1)
}

func assertMsgsV2Push(t *testing.T, q *queues.MsgsV2, vc valkey.Conn, owner queues.OwnerID, tps int, priority bool, batch string) {
	t.Helper()

	require.NoError(t, q.Push(t.Context(), vc, owner, tps, priority, []byte(batch)))
}

func assertMsgsV2Pop(t *testing.T, q *queues.MsgsV2, vc valkey.Conn, expectedOwner queues.OwnerID, expectedMsg string) {
	t.Helper()

	owner, msg, err := q.Pop(t.Context(), vc)
	require.NoError(t, err)
	assert.Equal(t, expectedOwner, owner)
	if expectedMsg == "" {
		assert.Nil(t, msg)
	} else {
		assert.Equal(t, expectedMsg, string(msg))
	}
}

func assertMsgsV2Size(t *testing.T, q *queues.MsgsV2, vc valkey.Conn, expectedHigh, expectedBulk int) {
	t.Helper()

	high, bulk, err := q.Size(t.Context(), vc)
	require.NoError(t, err)
	assert.Equal(t, expectedHigh, high, "high priority size mismatch")
	assert.Equal(t, expectedBulk, bulk, "bulk priority size mismatch")
}

func assertMsgsV2Empty(t *testing.T, q *queues.MsgsV2, vc valkey.Conn) {
	t.Helper()

	assertMsgsV2Pop(t, q, vc, "", "")
	assertMsgsV2Size(t, q, vc, 0, 0)
	assertvk.Keys(t, vc, "*", []string{})
}

func assertTTL(t *testing.T, vc valkey.Conn, key string, expected int) {
	t.Helper()

	ttl, err := valkey.Int(vc.Do("TTL", key))
	require.NoError(t, err)
	assert.Equal(t, expected, ttl, "TTL mismatch for %s", key)
}

func TestMsgsV2InvalidItems(t *testing.T) {
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	// items that aren't non-empty arrays are dropped without counting against the owner
	assertMsgsV2Push(t, q, vc, "chan1", 1, false, `[]`)
	assertMsgsV2Push(t, q, vc, "chan1", 1, false, `{"id":1}`)
	assertMsgsV2Push(t, q, vc, "chan1", 1, false, `"foo"`)
	assertMsgsV2Push(t, q, vc, "chan1", 1, false, `not json`)

	setMsgsV2Time(0.1)
	assertMsgsV2Push(t, q, vc, "chan1", 1, false, `[{"id":2}]`)

	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":2}`)
	assertMsgsV2Size(t, q, vc, 0, 0)
	assertvk.HGetAll(t, vc, "{msgs}:ch:chan1", map[string]string{"tps": "1", "sec": "1767225600", "used": "1", "inflight": "1"})
}
