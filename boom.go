package boom

import (
	"reflect"
)

var nilValue = reflect.ValueOf(nil)

// TaskResult is the result expected from a boom task when it's finished executing
type TaskResult interface {
	Err() error
}

type ValueResult struct {
	Value interface{}
	Error error
}

// NewValueResult is a convenience function for creating a ValueResult
func NewValueResult(val interface{}, err error) *ValueResult {
	return &ValueResult{
		Value: val,
		Error: err,
	}
}

func (r *ValueResult) Err() error {
	return r.Error
}
