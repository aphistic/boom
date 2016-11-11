package boom

import (
	"errors"
)

var (
	// ErrTimeout is returned when a function's execution times out
	ErrTimeout = errors.New("Execution timed out")
)
