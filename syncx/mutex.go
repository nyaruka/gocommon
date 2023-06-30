package syncx

import (
	"crypto/md5"
	"encoding/binary"
	"sync"
)

// KeyMutex is a key based mutual exclusion lock. Different keys can hold the lock at the same time, same keys block.
//
// m := KeyMutex{}
// unlock := m.Lock("key1")
// defer unlock()
//
// Note that mutexes are not removed from the map when they are unlocked. Therefore the underlying map of mutexes can
// grow indefinitely.
//
type KeyMutex struct {
	mutexes sync.Map
}

// Lock locks on the given key and returns a function to unlock.
func (m *KeyMutex) Lock(key string) func() {
	return m.lock(key)
}

func (m *KeyMutex) lock(k any) func() {
	value, _ := m.mutexes.LoadOrStore(k, &sync.Mutex{})
	mtx := value.(*sync.Mutex)
	mtx.Lock()

	return func() { mtx.Unlock() }
}

// Range is the same as sync.Map.Range.
func (m *KeyMutex) Range(f func(key, value any) bool) {
	m.mutexes.Range(f)
}

// HashMutex is a KeyMutex which reduces keys to a fixed number of bits so that there will only ever be a fixed
// number of mutexes. This means that there's no guarantee that two disctinct keys will use separate locks, but it is
// guaranteed that different calls with the same key, will use the same lock.
type HashMutex struct {
	KeyMutex
	keybits int
}

// NewHashMutex creates a new hash mutex with keys reduced to the given number of bits, e.g. if bits is 4, then keys
// be reduced to 16 possible values.
func NewHashMutex(keybits int) *HashMutex {
	return &HashMutex{keybits: keybits}
}

func (m *HashMutex) Lock(key string) func() {
	return m.lock(hashToBits(key, m.keybits))
}

// hashes and reduces the given string to an integer with the given number of bits
func hashToBits(s string, bits int) uint32 {
	h := md5.New()
	h.Write([]byte(s))
	n := binary.BigEndian.Uint32(h.Sum(nil)[0:4])
	shift := uint32(32 - bits)
	return (n << shift) >> shift
}
