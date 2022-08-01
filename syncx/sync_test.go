package syncx_test

import (
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/syncx"
	"github.com/stretchr/testify/assert"
)

func TestMutexMap(t *testing.T) {
	wg := sync.WaitGroup{}
	m := syncx.MutexMap{}

	counters := make(map[string]int)
	countersMutex := sync.Mutex{}

	job := func(index int, key string, millis int) {
		unlock := m.Lock(key)

		countersMutex.Lock()
		counters[key]++
		assert.Equal(t, 1, counters[key])
		countersMutex.Unlock()

		time.Sleep(time.Millisecond * time.Duration(millis))

		countersMutex.Lock()
		counters[key]--
		countersMutex.Unlock()

		unlock()
	}

	for i, tc := range []struct {
		key    string
		millis int
	}{
		{key: "abc", millis: 500},
		{key: "abc", millis: 500},
		{key: "def", millis: 200},
		{key: "ghi", millis: 300},
		{key: "def", millis: 200},
	} {
		wg.Add(1)
		index := i
		key := tc.key
		millis := tc.millis

		go func() {
			defer wg.Done()
			job(index, key, millis)
		}()
	}

	wg.Wait()
}

func TestHashedMutexMap(t *testing.T) {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	randString := func(n int) string {
		b := make([]rune, n)
		for i := range b {
			b[i] = letters[rand.Intn(len(letters))]
		}
		return string(b)
	}

	m := syncx.NewHashedMutexMap(4)

	for i := 0; i < 1000; i++ {
		unlock := m.Lock(randString(10))
		unlock()
	}

	// get all the keys from the map
	keys := make([]any, 0, 16)
	m.Range(func(key, value any) bool {
		keys = append(keys, key)
		return true
	})
	sort.Slice(keys, func(i, j int) bool { return keys[i].(uint32) < keys[j].(uint32) })

	assert.Equal(t, []any{uint32(0), uint32(1), uint32(2), uint32(3), uint32(4), uint32(5), uint32(6), uint32(7), uint32(8), uint32(9), uint32(10), uint32(11), uint32(12), uint32(13), uint32(14), uint32(15)}, keys)
}
