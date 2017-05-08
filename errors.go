package boom

import (
	"errors"
)

var (
	// ErrTimeout is returned when a function's execution times out
	ErrTimeout = errors.New("Execution timed out")

	// ErrExecuting is returned when a task is requested to start but is
	// already started
	ErrExecuting = errors.New("Execution has already started")

	// ErrNotExecuting is returned when a task has not started executing yet
	ErrNotExecuting = errors.New("Task has not started executing yet")

	// ErrFinished is returned when execution for tasks has already finished
	ErrFinished = errors.New("Execution has already finished")
)
