package queues

// SetNewTaskID overrides the task ID generator in tests, or restores the default if fn is nil
func SetNewTaskID(fn func() TaskID) {
	if fn == nil {
		fn = defaultNewTaskID
	}
	newTaskID = fn
}
