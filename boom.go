package boom

import (
	"reflect"
)

var nilValue = reflect.ValueOf(nil)

type TaskResult struct {
	Value interface{}
	Err   error
}

func NewResult(val interface{}, err error) *TaskResult {
	return &TaskResult{
		Value: val,
		Err:   err,
	}
}
