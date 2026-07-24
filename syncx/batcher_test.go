package syncx_test

import (
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/syncx"
	"github.com/stretchr/testify/assert"
)

func TestBatcher(t *testing.T) {
	// process callback runs in the batcher goroutine so access to recorded batches needs to be synchronized
	var mu sync.Mutex
	all := make([][]int, 0)

	b := syncx.NewBatcher(func(batch []int) {
		mu.Lock()
		defer mu.Unlock()
		all = append(all, batch)
	}, 2, time.Second, 3)

	batches := func() [][]int {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(all)
	}

	wg := &sync.WaitGroup{}
	b.Start(wg)

	assert.Equal(t, 4, b.Queue(1)) // won't trigger a batch

	time.Sleep(time.Millisecond * 100)
	assert.Equal(t, [][]int{}, batches())

	b.Queue(2) // 2 items triggers a batch

	time.Sleep(time.Millisecond * 100)
	assert.Equal(t, [][]int{{1, 2}}, batches())

	b.Queue(3)
	b.Queue(4)

	time.Sleep(time.Millisecond * 100)
	assert.Equal(t, [][]int{{1, 2}, {3, 4}}, batches())

	b.Queue(5)

	time.Sleep(time.Millisecond * 100) // won't trigger a batch
	assert.Equal(t, [][]int{{1, 2}, {3, 4}}, batches())

	time.Sleep(time.Millisecond * 1100) // batch forced because of age
	assert.Equal(t, [][]int{{1, 2}, {3, 4}, {5}}, batches())

	time.Sleep(time.Millisecond * 1100) // empty batches never triggered
	assert.Equal(t, [][]int{{1, 2}, {3, 4}, {5}}, batches())

	b.Queue(6)
	b.Queue(7)
	b.Queue(8)
	b.Flush()

	assert.Equal(t, [][]int{{1, 2}, {3, 4}, {5}, {6, 7}, {8}}, batches())

	b.Queue(9)
	b.Queue(10)
	b.Queue(11)

	b.Stop()
	wg.Wait()

	assert.Equal(t, [][]int{{1, 2}, {3, 4}, {5}, {6, 7}, {8}, {9, 10}, {11}}, batches())

	// panic if you try to queue to a stopped batcher
	assert.Panics(t, func() {
		b.Queue(100)
	})
}
