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

type collectorResult struct {
	Choice int
	Result TaskResult
}

type collectorTask struct {
	f       CollectorFunc
	closer  CollectorCloser
	args    []interface{}
	resChan chan<- *collectorResult
}

func newCollectorTask(f CollectorFunc, args []interface{}, resChan chan<- *collectorResult) *collectorTask {
	return &collectorTask{
		f:       f,
		args:    args,
		resChan: resChan,
	}
}

func (ct *collectorTask) Start(choice int) {
	go ct.worker(choice)
}

func (ct *collectorTask) worker(choice int) {
	res := ct.f(ct.args...)
	ct.resChan <- &collectorResult{
		Choice: choice,
		Result: res,
	}
}
