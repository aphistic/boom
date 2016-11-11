package boom

import (
	"errors"
)

var (
	ErrTimeout = errors.New("Execution timed out")
)
