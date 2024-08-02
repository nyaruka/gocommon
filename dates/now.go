package dates

import (
	"sync"
	"time"
)

// Now returns the time now.. according to the current now function which can be switched out for testing.
func Now() time.Time {
	return currentNow()
}

// Since returns the time elapsed since t
func Since(t time.Time) time.Duration {
	return Now().Sub(t)
}

// NowFunc is a function that can provide a now time
type NowFunc func() time.Time

var currentNow = time.Now

// SetNowFunc sets the current now function
func SetNowFunc(source NowFunc) {
	currentNow = source
}

// NewFixedNow creates a new fixed now func
func NewFixedNow(now time.Time) NowFunc {
	return func() time.Time { return now }
}

type sequentialNow struct {
	start time.Time
	step  time.Duration
	mutex sync.Mutex
}

func (s *sequentialNow) now() time.Time {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := s.start
	s.start = s.start.Add(s.step)
	return now
}

// NewSequentialNow creates a new sequential time func
func NewSequentialNow(start time.Time, step time.Duration) NowFunc {
	return (&sequentialNow{start: start, step: step}).now
}
