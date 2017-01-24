package boom

import "time"

type Collector interface {
	Run(CollectorFunc, ...interface{})
	Wait(time.Duration) []TaskResult
	WaitCloser(time.Duration, CollectorCloser) ([]TaskResult, error)
}

// CollectorFunc is the signature for the function a Collector runs to perform a task
type CollectorFunc func(data ...interface{}) TaskResult

// CollectorCloser is the signature for the function a Collector runs when
// WaitCloser is called and an error occurs in the Collector.
type CollectorCloser func(result TaskResult)
