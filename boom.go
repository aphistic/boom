package boom

import (
	"reflect"
)

var nilValue = reflect.ValueOf(nil)

// TaskResult is the result expected from a boom task when it's finished executing
type TaskResult struct {
	Value interface{}
	Err   error
}

// NewResult is a convenience function for creating a TaskResult
func NewResult(val interface{}, err error) *TaskResult {
	return &TaskResult{
		Value: val,
		Err:   err,
	}
}
