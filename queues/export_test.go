package queues

import "time"

// SetNewTaskID overrides the task ID generator in tests, or restores the default if fn is nil
func SetNewTaskID(fn func() TaskID) {
	if fn == nil {
		fn = defaultNewTaskID
	}
	newTaskID = fn
}

// SetTimeNow overrides the time source in tests, or restores the default if fn is nil
func SetTimeNow(fn func() time.Time) {
	if fn == nil {
		fn = time.Now
	}
	timeNow = fn
}
