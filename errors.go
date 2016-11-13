package boom

import (
	"errors"
)

var (
	// ErrTimeout is returned when a function's execution times out
	ErrTimeout = errors.New("Execution timed out")

	// ErrFinished is returned when execution for tasks has already finished
	ErrFinished = errors.New("Execution has already finished")
)
