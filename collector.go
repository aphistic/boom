package boom

import (
	"context"
	"time"
)

type Collector interface {
	Run(TaskFunc, ...interface{})
	RunWithContext(context.Context, TaskFunc, ...interface{})
	Wait(time.Duration) ([]TaskResult, error)
	WaitCloser(time.Duration, CollectorCloser) ([]TaskResult, error)
}

// CollectorCloser is the signature for the function a Collector runs when
// WaitCloser is called and an error occurs in the Collector.
type CollectorCloser func(result TaskResult)
