package queues_test

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
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
	msgsV2Secs = secs
	queues.SetTimeNow(func() time.Time { return msgsV2Time(secs) })
}

var msgsV2Secs float64

// returns the time last set by setMsgsV2Time
func msgsV2Now() time.Time {
	return msgsV2Time(msgsV2Secs)
}

// returns the given number of seconds after t0
func msgsV2Time(secs float64) time.Time {
	return time.UnixMicro(int64((t0 + secs) * 1000000))
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

	require.NoError(t, q.Done(ctx, vc, "chan1", 1))
	require.NoError(t, q.Done(ctx, vc, "chan1", 1))
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
	assertMsgsV2Pop(t, q, vc, "", "") // parks chan1 as it's in debt

	assertvk.Keys(t, vc, "*", []string{"{msgs}:ch:chan1", "{msgs}:q:chan1/0", "{msgs}:waiting"})
	assertvk.HGetAll(t, vc, "{msgs}:ch:chan1", map[string]string{
		"tps": "1", "debt_until": "1767225601.5", "inflight": "1",
		"sec": "1767225600", "used": "1", "debt_sec": "1767225600", "debt_used": "1",
	})
	assertvk.ZGetAll(t, vc, "{msgs}:q:chan1/0", map[string]float64{`[{"id":2}]`: t0})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 1})

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
	assertvk.HGetAll(t, vc, "{msgs}:ch:chan2", map[string]string{
		"tps": "10", "debt_until": "1767225601.6", "inflight": "1",
		"sec": "1767225601", "used": "1", "debt_sec": "1767225601", "debt_used": "1",
	})

	require.NoError(t, q.Done(ctx, vc, "chan2", 1))
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

	for i := range 6 {
		assertMsgsV2Push(t, q, vc, "chan1", 2, false, fmt.Sprintf(`[{"id":%d}]`, i))
	}

	// each pop charges an estimated cost of 1, and an owner can be popped from while that leaves its debt no more than
	// a second's worth
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":0}`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "debt_until", "1767225601.5")

	// so now waits until another send fits
	assertMsgsV2Pop(t, q, vc, "", "")
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 0.5})

	// pushing while throttled doesn't make it ready
	assertMsgsV2Push(t, q, vc, "chan1", 2, false, `[{"id":6}]`)
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 0.5})

	// done doesn't either
	require.NoError(t, q.Done(ctx, vc, "chan1", 1))
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})

	setMsgsV2Time(0.4)
	assertMsgsV2Pop(t, q, vc, "", "")

	// debt drains at the TPS, with no extra allowance at the start of a second
	setMsgsV2Time(0.5)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":2}`)
	assertMsgsV2Pop(t, q, vc, "", "")
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 1})

	setMsgsV2Time(1)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":3}`)
	assertMsgsV2Pop(t, q, vc, "", "")
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 1.5})

	// a push with a new TPS applies to everything queued
	setMsgsV2Time(1.5)
	assertMsgsV2Push(t, q, vc, "chan1", 10, false, `[{"id":7}]`)
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan1": 3})
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":4}`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":5}`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":6}`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":7}`)
	assertMsgsV2Pop(t, q, vc, "", "")
}
func TestMsgsV2PushWhenThrottled(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	assertMsgsV2Push(t, q, vc, "chan1", 1, false, `[{"id":1}]`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)

	// a ready owner that's in debt is left for a pop to park
	assertMsgsV2Push(t, q, vc, "chan1", 1, false, `[{"id":2}]`)
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan1": 1})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{})

	assertMsgsV2Pop(t, q, vc, "", "")
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 1})

	setMsgsV2Time(1)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":2}`)

	// an owner that isn't ready but is in debt is made to wait
	assertMsgsV2Push(t, q, vc, "chan2", 10, false, `[{"id":3}]`)
	assertMsgsV2Pop(t, q, vc, "chan2", `{"id":3}`)
	require.NoError(t, q.Done(ctx, vc, "chan2", 20))
	assertMsgsV2Pop(t, q, vc, "", "") // drains chan2 and parks chan1
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})

	assertMsgsV2Push(t, q, vc, "chan2", 10, false, `[{"id":4}]`)
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})
	assertMsgsV2Waiting(t, vc, map[queues.OwnerID]float64{"chan1": 2, "chan2": 2.1})

	setMsgsV2Time(2.1)
	assertMsgsV2Pop(t, q, vc, "chan2", `{"id":4}`)
}
func TestMsgsV2Cost(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	for i := range 5 {
		assertMsgsV2Push(t, q, vc, "chan1", 2, false, fmt.Sprintf(`[{"id":%d}]`, i))
	}

	// done swaps the estimated cost of 1 charged by pop for the actual cost
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":0}`)
	assertMsgsV2DebtUntil(t, vc, "chan1", 0.5)
	require.NoError(t, q.Done(ctx, vc, "chan1", 3))
	assertMsgsV2DebtUntil(t, vc, "chan1", 1.5)
	assertMsgsV2Pop(t, q, vc, "", "")
	assertMsgsV2Waiting(t, vc, map[queues.OwnerID]float64{"chan1": 1})

	setMsgsV2Time(1)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)
	assertMsgsV2Pop(t, q, vc, "", "")
	assertMsgsV2Waiting(t, vc, map[queues.OwnerID]float64{"chan1": 1.5})

	// a send that cost less than estimated lets a waiting owner send sooner
	require.NoError(t, q.Done(ctx, vc, "chan1", 0))
	assertMsgsV2DebtUntil(t, vc, "chan1", 1.5)
	assertMsgsV2Waiting(t, vc, map[queues.OwnerID]float64{"chan1": 1})
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":2}`)

	// negative costs count as zero
	require.NoError(t, q.Done(ctx, vc, "chan1", -5))
	assertMsgsV2DebtUntil(t, vc, "chan1", 1.5)

	// debt is capped so that a very expensive send can't block the owner for long
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":3}`)
	require.NoError(t, q.Done(ctx, vc, "chan1", 1000))
	assertMsgsV2DebtUntil(t, vc, "chan1", 11)
	assertMsgsV2Pop(t, q, vc, "", "")
	assertMsgsV2Waiting(t, vc, map[queues.OwnerID]float64{"chan1": 10.5})

	setMsgsV2Time(10.5)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":4}`)

	// without a TPS nothing is charged
	assertMsgsV2Push(t, q, vc, "chan2", 0, false, `[{"id":5}]`)
	assertMsgsV2Pop(t, q, vc, "chan2", `{"id":5}`)
	require.NoError(t, q.Done(ctx, vc, "chan2", 10))
	assertvk.HGetAll(t, vc, "{msgs}:ch:chan2", map[string]string{"tps": "0", "inflight": "0"})
}
func TestMsgsV2LegacyWindow(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	assertMsgsV2Push(t, q, vc, "chan1", 5, false, `[{"id":1,"tps_cost":3}]`)
	assertMsgsV2Push(t, q, vc, "chan1", 5, false, `[{"id":2,"tps_cost":0.5}]`)
	assertMsgsV2Push(t, q, vc, "chan1", 5, false, `[{"id":3,"tps_cost":"2"}]`)
	assertMsgsV2Push(t, q, vc, "chan1", 5, false, `[{"id":4}]`)
	assertMsgsV2Push(t, q, vc, "chan1", 5, false, `[{"id":5}]`)
	assertMsgsV2Push(t, q, vc, "chan1", 5, false, `[{"id":6}]`)

	// older versions see what we send as usage of this second, charged as they would
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1,"tps_cost":3}`)
	assertvk.HGetAll(t, vc, "{msgs}:ch:chan1", map[string]string{
		"tps": "5", "debt_until": "1767225600.7", "inflight": "1",
		"sec": "1767225600", "used": "3", "debt_sec": "1767225600", "debt_used": "3",
	})

	// with costs below 1 or not numbers counting as 1
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":2,"tps_cost":0.5}`)
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "used", "4")
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":3,"tps_cost":"2"}`)
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "used", "5")

	// but that usage doesn't throttle us as we charge actual costs as debt
	require.NoError(t, q.Done(ctx, vc, "chan1", 1))
	require.NoError(t, q.Done(ctx, vc, "chan1", 1))
	require.NoError(t, q.Done(ctx, vc, "chan1", 1))
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":4}`)
	assertMsgsV2DebtUntil(t, vc, "chan1", 0.8)
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "used", "6")

	// whereas usage by older versions is charged as debt, so we don't send what they already have
	_, err := vc.Do("HINCRBY", "{msgs}:ch:chan1", "used", 2)
	require.NoError(t, err)
	assertMsgsV2Pop(t, q, vc, "", "")
	assertMsgsV2DebtUntil(t, vc, "chan1", 1.2)
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "debt_used", "8")
	assertMsgsV2Waiting(t, vc, map[queues.OwnerID]float64{"chan1": 0.4})

	// and an older version of a pushing app can make an owner ready that's in debt, which a pop parks
	_, err = vc.Do("ZADD", "{msgs}:ready", 0, "chan1")
	require.NoError(t, err)
	assertMsgsV2Pop(t, q, vc, "", "")
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})
	assertMsgsV2Waiting(t, vc, map[queues.OwnerID]float64{"chan1": 0.4})

	setMsgsV2Time(0.41)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":5}`)

	// usage by older versions is capped as debt like any other
	_, err = vc.Do("HINCRBY", "{msgs}:ch:chan1", "used", 1000)
	require.NoError(t, err)
	assertMsgsV2Pop(t, q, vc, "", "")
	assertMsgsV2DebtUntil(t, vc, "chan1", 10.41)
	assertMsgsV2Waiting(t, vc, map[queues.OwnerID]float64{"chan1": 9.61})

	// usage resets with each second
	setMsgsV2Time(9.62)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":6}`)
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "used", "1")
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "debt_used", "1")
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
	require.NoError(t, q.Done(ctx, vc, "chan1", 1))
	require.NoError(t, q.Done(ctx, vc, "chan1", 1))
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

	// every owner is now in debt so gets parked
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
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan100": t0 + 1})

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
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "inflight", "0")
	assertTTL(t, vc, "{msgs}:ch:chan1", 60)

	// sends that were still in flight complete to zero
	require.NoError(t, q.Done(ctx, vc, "chan1", 1))
	require.NoError(t, q.Done(ctx, vc, "chan1", 1))
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "inflight", "0")
	assertTTL(t, vc, "{msgs}:ch:chan1", 60)

	// debt survives the drain
	assertMsgsV2DebtUntil(t, vc, "chan1", 0.2)
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
	require.NoError(t, q.Done(ctx, vc, "chan1", 1))
	assertTTL(t, vc, "{msgs}:ch:chan1", 60)

	// but not that of a waiting owner
	require.NoError(t, q.Done(ctx, vc, "chan2", 1))
	assertvk.HGet(t, vc, "{msgs}:ch:chan2", "inflight", "0")
	assertTTL(t, vc, "{msgs}:ch:chan2", -1)

	// or a ready one
	setMsgsV2Time(3)
	assertMsgsV2Pop(t, q, vc, "chan2", `{"id":3}`)
	require.NoError(t, q.Done(ctx, vc, "chan2", 1))
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan2": 0})
	assertTTL(t, vc, "{msgs}:ch:chan2", -1)

	// done for an owner with no state creates state that expires
	require.NoError(t, q.Done(ctx, vc, "chan3", 1))
	assertvk.HGetAll(t, vc, "{msgs}:ch:chan3", map[string]string{"inflight": "0"})
	assertTTL(t, vc, "{msgs}:ch:chan3", 60)
}

func TestMsgsV2Pause(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	for i := range 3 {
		assertMsgsV2Push(t, q, vc, "chan1", 0, false, fmt.Sprintf(`[{"id":%d}]`, i))
	}
	assertMsgsV2Push(t, q, vc, "chan2", 0, false, `[{"id":10}]`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":0}`)

	// pausing a ready owner parks it until the pause ends
	require.NoError(t, q.Pause(ctx, vc, "chan1", msgsV2Time(10)))
	assertvk.HGetAll(t, vc, "{msgs}:ch:chan1", map[string]string{"tps": "0", "inflight": "1", "paused_until": "1767225610.5"})
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{"chan2": 0})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 10})

	// other owners are unaffected
	assertMsgsV2Pop(t, q, vc, "chan2", `{"id":10}`)
	assertMsgsV2Pop(t, q, vc, "", "")

	// pushing doesn't make a paused owner ready
	assertMsgsV2Push(t, q, vc, "chan1", 0, true, `[{"id":3}]`)
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 10})

	// a pause can be extended but not shortened, and pauses in the past are ignored
	require.NoError(t, q.Pause(ctx, vc, "chan1", msgsV2Time(5)))
	require.NoError(t, q.Pause(ctx, vc, "chan2", msgsV2Time(-1)))
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "paused_until", "1767225610.5")
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 10})

	require.NoError(t, q.Pause(ctx, vc, "chan1", msgsV2Time(20)))
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "paused_until", "1767225620.5")
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 20})

	setMsgsV2Time(19.9)
	assertMsgsV2Pop(t, q, vc, "", "")

	setMsgsV2Time(20)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":3}`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":2}`)
	assertMsgsV2Pop(t, q, vc, "", "")
}

func TestMsgsV2PauseWithOtherApps(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	assertMsgsV2Push(t, q, vc, "chan1", 0, false, `[{"id":1},{"id":2}]`)
	assertMsgsV2Push(t, q, vc, "chan1", 0, false, `[{"id":3}]`)
	require.NoError(t, q.Pause(ctx, vc, "chan1", msgsV2Time(1)))

	// a paused owner made ready by an app that doesn't know about pauses is parked by a pop
	_, err := vc.Do("ZADD", "{msgs}:ready", 0, "chan1")
	require.NoError(t, err)
	assertMsgsV2Pop(t, q, vc, "", "")
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 1})

	setMsgsV2Time(1)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":3}`)
	assertMsgsV2Pop(t, q, vc, "", "")
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 4})

	// a pause that ends before what the owner is waiting on doesn't wake it early
	require.NoError(t, q.Pause(ctx, vc, "chan1", msgsV2Time(2)))
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 4})

	// but a push for a paused owner only waits on the pause
	assertMsgsV2Push(t, q, vc, "chan1", 0, false, `[{"id":4}]`)
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 2})

	setMsgsV2Time(2)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":4}`)
}

func TestMsgsV2PauseStateExpiry(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	// pausing an owner with nothing queued keeps its state for the pause
	assertMsgsV2Push(t, q, vc, "chan1", 0, false, `[{"id":1}]`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)
	assertMsgsV2Pop(t, q, vc, "", "")
	assertTTL(t, vc, "{msgs}:ch:chan1", 60)

	require.NoError(t, q.Pause(ctx, vc, "chan1", msgsV2Time(90)))
	require.NoError(t, q.Pause(ctx, vc, "chan2", msgsV2Time(90)))
	assertvk.HGetAll(t, vc, "{msgs}:ch:chan2", map[string]string{"paused_until": "1767225690.5"})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 90, "chan2": t0 + 90})

	require.NoError(t, q.Done(ctx, vc, "chan1", 1))
	assertTTL(t, vc, "{msgs}:ch:chan1", -1)
	assertTTL(t, vc, "{msgs}:ch:chan2", -1)

	// so that pushes during the pause wait on it
	setMsgsV2Time(80)
	assertMsgsV2Push(t, q, vc, "chan2", 0, false, `[{"id":2}]`)
	assertMsgsV2Pop(t, q, vc, "", "")

	// and once the pause ends, state expires as soon as there's nothing queued
	setMsgsV2Time(90)
	assertMsgsV2Pop(t, q, vc, "chan2", `{"id":2}`)
	assertMsgsV2Pop(t, q, vc, "", "")
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{})
	assertTTL(t, vc, "{msgs}:ch:chan1", 60)
	assertTTL(t, vc, "{msgs}:ch:chan2", 60)
}

func TestMsgsV2ClearWhenPaused(t *testing.T) {
	ctx := t.Context()
	vc := assertvk.ClaimDB(t).Pool().Get()
	defer vc.Close()

	setMsgsV2Time(0)
	defer queues.SetTimeNow(nil)

	q := queues.NewMsgsV2("msgs")

	assertMsgsV2Push(t, q, vc, "chan1", 0, false, `[{"id":1},{"id":2}]`)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":1}`)
	require.NoError(t, q.Pause(ctx, vc, "chan1", msgsV2Time(90)))

	// clearing a paused owner keeps it parked, and its state, until the pause ends
	require.NoError(t, q.Clear(ctx, vc, "chan1"))
	assertMsgsV2Size(t, q, vc, 0, 0)
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{"chan1": t0 + 90})
	assertTTL(t, vc, "{msgs}:ch:chan1", -1)

	setMsgsV2Time(80)
	assertMsgsV2Push(t, q, vc, "chan1", 0, false, `[{"id":3}]`)
	assertMsgsV2Pop(t, q, vc, "", "")

	setMsgsV2Time(90)
	assertMsgsV2Pop(t, q, vc, "chan1", `{"id":3}`)
	assertMsgsV2Pop(t, q, vc, "", "")
	assertTTL(t, vc, "{msgs}:ch:chan1", 60)

	// once the pause is over, clearing lets the state expire as usual
	assertMsgsV2Push(t, q, vc, "chan1", 0, false, `[{"id":4}]`)
	require.NoError(t, q.Clear(ctx, vc, "chan1"))
	assertvk.ZGetAll(t, vc, "{msgs}:ready", map[string]float64{})
	assertvk.ZGetAll(t, vc, "{msgs}:waiting", map[string]float64{})
	assertTTL(t, vc, "{msgs}:ch:chan1", 60)
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
	assertMsgsV2Waiting(t, vc, map[queues.OwnerID]float64{"chan1": 1, "chan2": 1, "chan3": 1})
	assertMsgsV2Size(t, q, vc, 0, 2)

	require.NoError(t, q.Clear(ctx, vc, "chan1"))

	assertvk.NotExists(t, vc, "{msgs}:q:chan1/1")
	assertvk.NotExists(t, vc, "{msgs}:q:chan1/0")
	assertMsgsV2Waiting(t, vc, map[queues.OwnerID]float64{"chan2": 1, "chan3": 1})
	assertMsgsV2Size(t, q, vc, 0, 1)

	// state is kept for the send in flight, but expires
	assertvk.HGet(t, vc, "{msgs}:ch:chan1", "inflight", "1")
	assertTTL(t, vc, "{msgs}:ch:chan1", 60)
	require.NoError(t, q.Done(ctx, vc, "chan1", 1))
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

// asserts the owner's debt_until, as seconds after t0
func assertMsgsV2DebtUntil(t *testing.T, vc valkey.Conn, owner queues.OwnerID, expected float64) {
	t.Helper()

	actual, err := valkey.Float64(vc.Do("HGET", "{msgs}:ch:"+owner, "debt_until"))
	require.NoError(t, err)
	assert.InDelta(t, expected, actual-t0, 0.000001, "debt_until mismatch for %s", owner)
}

// asserts the waiting owners and when they wake, as seconds after t0
func assertMsgsV2Waiting(t *testing.T, vc valkey.Conn, expected map[queues.OwnerID]float64) {
	t.Helper()

	vals, err := valkey.StringMap(vc.Do("ZRANGE", "{msgs}:waiting", 0, -1, "WITHSCORES"))
	require.NoError(t, err)
	actual := make(map[queues.OwnerID]float64, len(vals))
	for owner, score := range vals {
		f, err := strconv.ParseFloat(score, 64)
		require.NoError(t, err)
		actual[queues.OwnerID(owner)] = f - t0
	}
	require.ElementsMatch(t, slices.Collect(maps.Keys(expected)), slices.Collect(maps.Keys(actual)), "waiting mismatch")
	for owner, e := range expected {
		assert.InDelta(t, e, actual[owner], 0.000001, "wake time mismatch for %s", owner)
	}
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
	assertvk.HGetAll(t, vc, "{msgs}:ch:chan1", map[string]string{
		"tps": "1", "debt_until": "1767225601.6", "inflight": "1",
		"sec": "1767225600", "used": "1", "debt_sec": "1767225600", "debt_used": "1",
	})
}

func TestMsgsV2Rates(t *testing.T) {
	mixed := []int{1, 3, 1, 0, 2}

	tcs := []struct {
		name    string
		owners  []msgsV2SimOwner
		latency float64
		window  float64
	}{
		{
			// low TPS owners measured over a long window as a single send can be several seconds' worth
			name:    "low tps, fast sends",
			owners:  []msgsV2SimOwner{{"tps1", 1, mixed}, {"tps1x3", 1, []int{3}}, {"tps2", 2, mixed}},
			latency: 0.05,
			window:  300,
		},
		{
			// a send that costs nothing, i.e. reached no provider, still holds its estimated cost while in flight, which
			// an owner with a low TPS can't make up when sends are slow, so these all cost something
			name:    "low tps, slow sends",
			owners:  []msgsV2SimOwner{{"tps1", 1, []int{1, 3, 1, 2}}, {"tps1x3", 1, []int{3}}, {"tps2", 2, []int{1, 3, 1, 2}}},
			latency: 2,
			window:  300,
		},
		{
			name:    "mixed tps, fast sends",
			owners:  []msgsV2SimOwner{{"tps1", 1, mixed}, {"tps5", 5, mixed}, {"tps50", 50, mixed}, {"tps500", 500, mixed}},
			latency: 0.05,
			window:  10,
		},
		{
			name:    "mixed tps, slow sends",
			owners:  []msgsV2SimOwner{{"tps1", 1, mixed}, {"tps5", 5, mixed}, {"tps50", 50, mixed}, {"tps500", 500, mixed}},
			latency: 2,
			window:  10,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			vc := assertvk.ClaimDB(t).Pool().Get()
			defer vc.Close()

			sim := simulateMsgsV2(t, vc, []msgsV2SimNode{newMsgsV2SimNode(vc)}, tc.owners, tc.latency, tc.window)
			sim.assertRates(t, tc.owners, 0.01)
		})
	}
}

func TestMsgsV2MixedVersions(t *testing.T) {
	// older versions charge the tps_cost estimate at pop, so make that the actual cost
	owners := []msgsV2SimOwner{{"tps1", 1, []int{1, 3, 1, 2}}, {"tps10", 10, []int{1, 3, 1, 2}}, {"tps50", 50, []int{1}}}

	simulate := func(t *testing.T, versions ...string) *msgsV2SimResult {
		vc := assertvk.ClaimDB(t).Pool().Get()
		defer vc.Close()

		var nodes []msgsV2SimNode
		for _, v := range versions {
			if v == "legacy" {
				nodes = append(nodes, newMsgsV2LegacySimNode(t, vc))
			} else {
				nodes = append(nodes, newMsgsV2SimNode(vc))
			}
		}
		return simulateMsgsV2(t, vc, nodes, owners, 0.05, 30)
	}

	// older versions let a single send take an owner over its TPS each second, so overshoot at low TPS
	legacy := simulate(t, "legacy")
	legacy.logRates(t, owners)

	for _, versions := range [][]string{{"legacy", "new"}, {"legacy", "new", "new", "new"}} {
		t.Run(strings.Join(versions, ","), func(t *testing.T) {
			sim := simulate(t, versions...)
			sim.logRates(t, owners)

			// a mix of versions does no worse than older versions alone, and doesn't starve owners, though may send a
			// little under the TPS as both versions' charges apply
			for _, o := range owners {
				expected := float64(o.tps) * sim.window
				slack := float64(slices.Max(o.costs))
				actual, legacyActual := float64(sim.charged[o.id]), float64(legacy.charged[o.id])
				assert.LessOrEqual(t, actual, max(legacyActual, 1.02*expected)+slack, "overshoot for %s", o.id)
				assert.GreaterOrEqual(t, actual, 0.95*expected-slack, "undershoot for %s", o.id)
			}
		})
	}
}

type msgsV2SimOwner struct {
	id    queues.OwnerID
	tps   int
	costs []int // costs of successive sends, cycled
}

// an app that pushes and pops, which may be running an older version
type msgsV2SimNode struct {
	push func(queues.OwnerID, int, string)
	pop  func() (queues.OwnerID, []byte)
	done func(queues.OwnerID, int)
}

func newMsgsV2SimNode(vc valkey.Conn) msgsV2SimNode {
	q := queues.NewMsgsV2("msgs")
	ctx := context.Background()

	return msgsV2SimNode{
		push: func(owner queues.OwnerID, tps int, batch string) {
			must(q.Push(ctx, vc, owner, tps, false, []byte(batch)))
		},
		pop: func() (queues.OwnerID, []byte) {
			owner, msg, err := q.Pop(ctx, vc)
			must(err)
			return owner, msg
		},
		done: func(owner queues.OwnerID, cost int) { must(q.Done(ctx, vc, owner, cost)) },
	}
}

// runs copies of the scripts of the version that charged the tps_cost estimate at pop against a fixed window
func newMsgsV2LegacySimNode(t *testing.T, vc valkey.Conn) msgsV2SimNode {
	load := func(name string, keys int) *valkey.Script {
		src, err := os.ReadFile("testdata/msgs2_legacy/" + name + ".lua")
		require.NoError(t, err)
		return valkey.NewScript(keys, string(src))
	}
	push, pop, done := load("push", 4), load("pop", 2), load("done", 3)
	nowMicros := func() int64 { return msgsV2Now().UnixMicro() }

	return msgsV2SimNode{
		push: func(owner queues.OwnerID, tps int, batch string) {
			_, err := push.Do(vc, "{msgs}:q:"+owner+"/0", "{msgs}:ch:"+owner, "{msgs}:ready", "{msgs}:waiting", owner, tps,
				batch, nowMicros())
			must(err)
		},
		pop: func() (queues.OwnerID, []byte) {
			for {
				vals, err := valkey.ByteSlices(pop.Do(vc, "{msgs}:ready", "{msgs}:waiting", "{msgs}", 100, nowMicros()))
				must(err)
				switch string(vals[0]) {
				case "empty":
					return "", nil
				case "retry":
					continue
				}
				return queues.OwnerID(vals[0]), vals[1]
			}
		},
		done: func(owner queues.OwnerID, cost int) {
			_, err := done.Do(vc, "{msgs}:ch:"+owner, "{msgs}:ready", "{msgs}:waiting", owner)
			must(err)
		},
	}
}

type msgsV2SimResult struct {
	window  float64
	charged map[queues.OwnerID]int // total cost of sends popped during the window
	maxSec  map[queues.OwnerID]int // most cost of sends popped in any second of the window
}

func (r *msgsV2SimResult) logRates(t *testing.T, owners []msgsV2SimOwner) {
	t.Helper()

	for _, o := range owners {
		expected := float64(o.tps) * r.window
		actual := float64(r.charged[o.id])
		t.Logf("%-8s tps=%-3d expected=%-6.0f charged=%-6.0f (%+.2f%%) max/sec=%d", o.id, o.tps, expected, actual,
			100*(actual-expected)/expected, r.maxSec[o.id])
	}
}

func (r *msgsV2SimResult) assertRates(t *testing.T, owners []msgsV2SimOwner, tolerance float64) {
	t.Helper()

	r.logRates(t, owners)

	for _, o := range owners {
		// what's charged for a send popped near the end of the window is only paid off after it
		expected := float64(o.tps) * r.window
		delta := tolerance*expected + float64(slices.Max(o.costs))
		assert.InDelta(t, expected, float64(r.charged[o.id]), delta, "rate mismatch for %s", o.id)
	}
}

// simulates owners with full queues being pushed to and popped from by the given nodes every few milliseconds, with
// sends taking the given latency, and measures what's popped during the window after a warmup.
func simulateMsgsV2(
	t *testing.T, vc valkey.Conn, nodes []msgsV2SimNode, owners []msgsV2SimOwner, latency, window float64,
) *msgsV2SimResult {
	defer queues.SetTimeNow(nil)

	const tick = 0.01
	warmup := latency + 3

	type send struct {
		node  msgsV2SimNode
		owner queues.OwnerID
		cost  int
		done  float64
	}
	var inflight []send
	pushed, popped := map[queues.OwnerID]int{}, map[queues.OwnerID]int{}
	perSec := map[queues.OwnerID]map[int]int{}
	result := &msgsV2SimResult{window: window, charged: map[queues.OwnerID]int{}, maxSec: map[queues.OwnerID]int{}}

	for i, now := 0, 0.0; now < warmup+window; i++ {
		setMsgsV2Time(now)

		// complete sends that are done
		remaining := inflight[:0]
		for _, s := range inflight {
			if s.done <= now {
				s.node.done(s.owner, s.cost)
			} else {
				remaining = append(remaining, s)
			}
		}
		inflight = remaining

		// keep every owner's queue topped up, with nodes taking turns to push
		for _, o := range owners {
			for pushed[o.id]-popped[o.id] < 2*o.tps+10 {
				n := pushed[o.id]
				c := o.costs[n%len(o.costs)]
				nodes[n%len(nodes)].push(o.id, o.tps, fmt.Sprintf(`[{"n":%d,"cost":%d,"tps_cost":%d}]`, n, c, c))
				pushed[o.id]++
			}
		}

		// nodes take turns to pop until none can, starting with a different one each tick
		for active, j := len(nodes), i; active > 0; j++ {
			node := nodes[j%len(nodes)]
			owner, msg := node.pop()
			if owner == "" {
				active--
				continue
			}
			active = len(nodes)

			var m struct{ Cost int }
			require.NoError(t, json.Unmarshal(msg, &m))
			popped[owner]++
			if now >= warmup {
				result.charged[owner] += m.Cost
				if perSec[owner] == nil {
					perSec[owner] = map[int]int{}
				}
				sec := int(math.Floor(t0 + now))
				perSec[owner][sec] += m.Cost
				result.maxSec[owner] = max(result.maxSec[owner], perSec[owner][sec])
			}
			inflight = append(inflight, send{node, owner, m.Cost, now + latency})
		}

		// step to the next tick at which a send completes or an owner wakes
		next := math.Inf(1)
		for _, s := range inflight {
			next = min(next, s.done)
		}
		waiting, err := valkey.Strings(vc.Do("ZRANGE", "{msgs}:waiting", 0, 0, "WITHSCORES"))
		require.NoError(t, err)
		if len(waiting) == 2 {
			wakeAt, _ := strconv.ParseFloat(waiting[1], 64)
			next = min(next, wakeAt-t0)
		}
		now = max(now+tick, math.Ceil(next/tick)*tick)
	}
	return result
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
