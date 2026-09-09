package queues

import (
	"time"
	"uuid"
)

// TaskID is the unique identifier for a task in the queue.
type TaskID string

// OwnerID is the identifier for an owner of tasks in the queue.
type OwnerID string

// newTaskID can be overridden in tests to generate predictable IDs
var newTaskID func() TaskID = defaultNewTaskID

func defaultNewTaskID() TaskID {
	return TaskID(uuid.NewV7().String())
}

// timeNow can be overridden in tests to control lease expiry
var timeNow func() time.Time = time.Now
