package queues

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/google/uuid"
)

// TaskID is the unique identifier for a task in the queue.
type TaskID string

// OwnerID is the identifier for an owner of tasks in the queue.
type OwnerID string

var idRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-7][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// newTaskID can be overridden in tests to generate predictable IDs
var newTaskID func() TaskID = defaultNewTaskID

func defaultNewTaskID() TaskID {
	return TaskID(uuid.Must(uuid.NewV7()).String())
}

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
