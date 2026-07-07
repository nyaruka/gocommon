package queues_test

import (
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	valkey "github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/nyaruka/gocommon/queues"
	"github.com/nyaruka/vkutil/assertvk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFair(t *testing.T) {
	ctx := t.Context()
	vp := assertvk.TestDB()
	vc := vp.Get()
	defer vc.Close()

	numIDs := 0
	queues.SetNewTaskID(func() queues.TaskID {
		numIDs++
		return queues.TaskID(fmt.Sprintf("01980000-0000-7000-8000-%012d", numIDs))
	})
	defer queues.SetNewTaskID(nil)

	defer assertvk.FlushDB()

	q := queues.NewFair("test", 3)

	assertQueued := func(expected map[queues.OwnerID]int) {
		actualStrings, err := valkey.StringMap(vc.Do("ZRANGE", "{test}:queued", 0, -1, "WITHSCORES"))
		require.NoError(t, err)

		actual := make(map[queues.OwnerID]int, len(actualStrings))
		for k, v := range actualStrings {
			actual[queues.OwnerID(k)], err = strconv.Atoi(v)
			require.NoError(t, err)
		}

		assert.Equal(t, expected, actual)

		// checked the .Queued method as well
		actualOwners, err := q.Queued(ctx, vc)
		assert.NoError(t, err)
		assert.ElementsMatch(t, slices.Collect(maps.Keys(expected)), actualOwners)
	}

	assertActive := func(expected map[queues.OwnerID]int) {
		actualStrings, err := valkey.StringMap(vc.Do("ZRANGE", "{test}:active", 0, -1, "WITHSCORES"))
		require.NoError(t, err)

		actual := make(map[queues.OwnerID]int, len(actualStrings))
		for k, v := range actualStrings {
			actual[queues.OwnerID(k)], err = strconv.Atoi(v)
			require.NoError(t, err)
		}

		assert.Equal(t, expected, actual)
	}

	assertTasks := func(owner queues.OwnerID, expected0, expected1 []string) {
		actual0, err := valkey.Strings(vc.Do("LRANGE", "{test}:o:"+owner+"/0", 0, -1))
		require.NoError(t, err)
		actual1, err := valkey.Strings(vc.Do("LRANGE", "{test}:o:"+owner+"/1", 0, -1))
		require.NoError(t, err)

		assert.Equal(t, expected0, actual0, "priority 0 tasks mismatch")
		assert.Equal(t, expected1, actual1, "priority 1 tasks mismatch")

		// checked .Size() method as well
		size, err := q.Size(ctx, vc, owner)
		assert.NoError(t, err)
		assert.Equal(t, len(expected0)+len(expected1), size)
	}

	assertDump := func(expected string) {
		dump, err := q.Dump(ctx, vc)
		require.NoError(t, err)
		assert.JSONEq(t, expected, string(dump), "dumped queue state does not match expected")
	}

	assertQueued(map[queues.OwnerID]int{})
	assertActive(map[queues.OwnerID]int{})
	assertTasks("owner1", []string{}, []string{})
	assertTasks("owner2", []string{}, []string{})
	assertDump(`{"queued": {}, "active": {}, "paused": {}}`)

	task1UUID := assertPush(t, q, vc, "owner1", false, []byte(`task1`))
	task2UUID := assertPush(t, q, vc, "owner1", true, []byte(`task2`))
	task3UUID := assertPush(t, q, vc, "owner2", false, []byte(`task3`))
	task4UUID := assertPush(t, q, vc, "owner1", false, []byte(`task4`))
	task5UUID := assertPush(t, q, vc, "owner2", true, []byte(`task5`))

	// nobody processing any tasks so no workers assigned in active set
	assertQueued(map[queues.OwnerID]int{"owner1": 3, "owner2": 2})
	assertActive(map[queues.OwnerID]int{})
	assertTasks("owner1", []string{"01980000-0000-7000-8000-000000000001|task1", "01980000-0000-7000-8000-000000000004|task4"}, []string{"01980000-0000-7000-8000-000000000002|task2"})
	assertTasks("owner2", []string{"01980000-0000-7000-8000-000000000003|task3"}, []string{"01980000-0000-7000-8000-000000000005|task5"})

	assertPop(t, q, vc, task2UUID, "owner1", "task2") // because it's highest priority for owner 1
	assertQueued(map[queues.OwnerID]int{"owner1": 2, "owner2": 2})
	assertActive(map[queues.OwnerID]int{"owner1": 1})

	assertPop(t, q, vc, task5UUID, "owner2", "task5") // because it's highest priority for owner 2
	assertQueued(map[queues.OwnerID]int{"owner1": 2, "owner2": 1})
	assertActive(map[queues.OwnerID]int{"owner1": 1, "owner2": 1})

	assertPop(t, q, vc, task1UUID, "owner1", "task1")
	assertQueued(map[queues.OwnerID]int{"owner1": 1, "owner2": 1})
	assertActive(map[queues.OwnerID]int{"owner1": 2, "owner2": 1})
	assertTasks("owner1", []string{"01980000-0000-7000-8000-000000000004|task4"}, []string{})
	assertTasks("owner2", []string{"01980000-0000-7000-8000-000000000003|task3"}, []string{})
	assertDump(`{"queued": {"owner1": 1, "owner2": 1}, "active": {"owner1": 2, "owner2": 1}, "paused": {}}`)

	// mark task2 and task1 (owner1) as complete
	q.Done(ctx, vc, "owner1")
	q.Done(ctx, vc, "owner1")

	assertQueued(map[queues.OwnerID]int{"owner1": 1, "owner2": 1})
	assertActive(map[queues.OwnerID]int{"owner2": 1})

	assertPop(t, q, vc, task4UUID, "owner1", "task4")
	assertPop(t, q, vc, task3UUID, "owner2", "task3")
	assertTasks("owner1", []string{}, []string{})
	assertTasks("owner2", []string{}, []string{})

	assertQueued(map[queues.OwnerID]int{})
	assertActive(map[queues.OwnerID]int{"owner1": 1, "owner2": 2})

	assertPop(t, q, vc, "", "", "") // no more tasks
	assertTasks("owner1", []string{}, []string{})
	assertTasks("owner2", []string{}, []string{})

	assertQueued(map[queues.OwnerID]int{})
	assertActive(map[queues.OwnerID]int{"owner1": 1, "owner2": 2})

	// mark remaining tasks as complete
	q.Done(ctx, vc, "owner1")
	q.Done(ctx, vc, "owner2")
	q.Done(ctx, vc, "owner2")

	assertQueued(map[queues.OwnerID]int{})
	assertActive(map[queues.OwnerID]int{})

	task6UUID := assertPush(t, q, vc, "owner1", false, []byte(`task6`))
	task7UUID := assertPush(t, q, vc, "owner1", false, []byte(`task7`))
	task8UUID := assertPush(t, q, vc, "owner2", false, []byte(`task8`))
	task9UUID := assertPush(t, q, vc, "owner2", false, []byte(`task9`))

	assertPop(t, q, vc, task6UUID, "owner1", "task6")

	q.Pause(ctx, vc, "owner1")
	q.Pause(ctx, vc, "owner1") // no-op if already paused

	assertQueued(map[queues.OwnerID]int{"owner1": 1, "owner2": 2})
	assertActive(map[queues.OwnerID]int{"owner1": 1})
	assertDump(`{"queued": {"owner1": 1, "owner2": 2}, "active": {"owner1": 1}, "paused": {"owner1": 1}}`)

	paused, err := q.Paused(ctx, vc)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []queues.OwnerID{"owner1"}, paused)

	assertPop(t, q, vc, task8UUID, "owner2", "task8")
	assertPop(t, q, vc, task9UUID, "owner2", "task9")
	assertPop(t, q, vc, "", "", "") // no more tasks

	q.Resume(ctx, vc, "owner1")
	q.Resume(ctx, vc, "owner1") // no-op if already active

	assertQueued(map[queues.OwnerID]int{"owner1": 1})
	assertActive(map[queues.OwnerID]int{"owner1": 1, "owner2": 2})

	paused, err = q.Paused(ctx, vc)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{}, paused)

	assertPop(t, q, vc, task7UUID, "owner1", "task7")

	q.Done(ctx, vc, "owner1")
	q.Done(ctx, vc, "owner1")
	q.Done(ctx, vc, "owner2")
	q.Done(ctx, vc, "owner2")

	assertQueued(map[queues.OwnerID]int{})
	assertActive(map[queues.OwnerID]int{})

	// if we somehow get into a state where an owner is in the queued set but doesn't have queued tasks, pop will retry
	assertPush(t, q, vc, "owner1", false, []byte("task10"))
	task11UUID := assertPush(t, q, vc, "owner2", false, []byte("task11"))

	assertQueued(map[queues.OwnerID]int{"owner1": 1, "owner2": 1})
	assertActive(map[queues.OwnerID]int{})

	assertvk.LLen(t, vc, "{test}:o:owner1/0", 1)
	_, err = vc.Do("DEL", "{test}:o:owner1/0") // task10 gone
	assert.NoError(t, err)

	assertPop(t, q, vc, task11UUID, "owner2", "task11")
	assertPop(t, q, vc, "", "", "")

	assertQueued(map[queues.OwnerID]int{})
	assertActive(map[queues.OwnerID]int{"owner2": 1})

	// if we somehow call done too many times, we never get negative workers
	q.Done(ctx, vc, "owner2")
	q.Done(ctx, vc, "owner2")

	assertActive(map[queues.OwnerID]int{})
}

func TestTaskPayloads(t *testing.T) {
	vp := assertvk.TestDB()
	vc := vp.Get()
	defer vc.Close()

	defer assertvk.FlushDB()

	q := queues.NewFair("test", 2)

	task1UUID := assertPush(t, q, vc, "owner1", true, []byte(`{"foo": "|"}`))
	task2UUID := assertPush(t, q, vc, "owner1", true, []byte(`task2`))

	assertPop(t, q, vc, task1UUID, "owner1", `{"foo": "|"}`)
	assertPop(t, q, vc, task2UUID, "owner1", "task2")
}

func TestFairMaxActivePerOwner(t *testing.T) {
	ctx := t.Context()
	vp := assertvk.TestDB()
	vc := vp.Get()
	defer vc.Close()

	defer assertvk.FlushDB()

	q := queues.NewFair("test", 2)

	task1UUID := assertPush(t, q, vc, "owner1", false, []byte(`task1`))
	task2UUID := assertPush(t, q, vc, "owner1", true, []byte(`task2`))
	task3UUID := assertPush(t, q, vc, "owner1", false, []byte(`task3`))

	assertPop(t, q, vc, task2UUID, "owner1", "task2")
	assertPop(t, q, vc, task1UUID, "owner1", "task1")
	assertPop(t, q, vc, "", "", "") // owner1 has reached max active tasks

	q.Done(ctx, vc, "owner1")

	assertPop(t, q, vc, task3UUID, "owner1", "task3") // now we can pop task3
}

func TestFairConcurrency(t *testing.T) {
	ctx := t.Context()
	vp := assertvk.TestDB()
	vc := vp.Get()
	defer vc.Close()

	defer assertvk.FlushDB()

	q := queues.NewFair("test", 5) // one owner can only occupy 5 of the 10 consumers at a time

	type ownerAndTask struct {
		owner queues.OwnerID
		task  string
	}

	numTasks := 10000
	pushedTasks := make([]*ownerAndTask, 0, numTasks)
	poppedTasks := make([]*ownerAndTask, 0, numTasks)

	var wg sync.WaitGroup
	var mutex sync.Mutex

	recordTaskPushed := func(owner queues.OwnerID, task string) {
		mutex.Lock()
		defer mutex.Unlock()

		pushedTasks = append(pushedTasks, &ownerAndTask{owner: owner, task: task})
	}

	recordTaskProcessed := func(owner queues.OwnerID, task string) {
		mutex.Lock()
		defer mutex.Unlock()

		poppedTasks = append(poppedTasks, &ownerAndTask{owner: owner, task: task})
	}

	// Start 5 producers to push tasks each concurrently
	for i := range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			vc := vp.Get()
			defer vc.Close()

			for range numTasks / 5 {
				owner := queues.OwnerID(fmt.Sprintf("owner%d", rand.IntN(5)+1)) // five possible owners (1...5)
				task := []byte(uuid.Must(uuid.NewV7()).String())
				_, err := q.Push(ctx, vc, owner, false, task)
				assert.NoError(t, err, "Producer %d failed to push task for owner %s", i, owner)

				recordTaskPushed(owner, string(task))

				time.Sleep(time.Duration(rand.IntN(5)) * time.Millisecond)
			}
		}()
	}

	// Start 10 consumers to pop tasks concurrently
	for i := range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			vc := vp.Get()
			defer vc.Close()

			for {
				_, owner, task, err := q.Pop(ctx, vc)
				assert.NoError(t, err, "Consumer %d failed to pop task", i)

				if task != nil {
					time.Sleep(time.Duration(rand.IntN(5)) * time.Millisecond)

					err = q.Done(ctx, vc, owner)
					assert.NoError(t, err, "Consumer %d failed to mark task done", i)

					recordTaskProcessed(owner, string(task))

					fmt.Printf("Consumer %d processed task %s for owner %s\n", i, string(task), owner)
				} else {
					fmt.Printf("Consumer %d got no task when popping\n", i)
				}
				// Check if all tasks have been processed
				mutex.Lock()
				allDone := len(poppedTasks) >= numTasks
				mutex.Unlock()

				if allDone {
					return
				} else {
					time.Sleep(time.Millisecond)
				}
			}
		}()
	}

	wg.Wait() // Wait for all producers and consumers to complete

	// can't guarantee order of processed tasks, but we can check that all expected tasks were processed
	assert.ElementsMatch(t, pushedTasks, poppedTasks)

	assertvk.ZGetAll(t, vc, "{test}:queued", map[string]float64{})
	assertvk.ZGetAll(t, vc, "{test}:active", map[string]float64{})

	for i := range 5 {
		assertvk.LGetAll(t, vc, fmt.Sprintf("{test}:o:owner%d/0", i+1), []string{})
		assertvk.LGetAll(t, vc, fmt.Sprintf("{test}:o:owner%d/1", i+1), []string{})
	}
}

// assertPush is a helper function that asserts the result of a Push operation
func assertPush(t *testing.T, q *queues.Fair, vc valkey.Conn, owner queues.OwnerID, priority bool, task []byte) queues.TaskID {
	ctx := t.Context()

	id, err := q.Push(ctx, vc, owner, priority, task)
	assert.NoError(t, err)
	return id
}

// assertPop is a helper function that asserts the result of a Pop operation
func assertPop(t *testing.T, q *queues.Fair, vc valkey.Conn, expectedID queues.TaskID, expectedOwner queues.OwnerID, expectedTask string) {
	ctx := t.Context()

	uuid, owner, task, err := q.Pop(ctx, vc)
	require.NoError(t, err)
	if expectedTask != "" {
		assert.Equal(t, expectedID, uuid)
		assert.Equal(t, expectedOwner, owner)
		assert.Equal(t, expectedTask, string(task))
	} else {
		assert.Nil(t, task)
	}
}
