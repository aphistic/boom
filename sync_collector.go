package boom

import "time"

// SyncCollector can take a number of tasks, execute them in parallel, and then collect
// the results of those tasks once all have been completed.
type SyncCollector struct{}

// NewSyncCollector creates a new SyncCollector instance
func NewSyncCollector() *SyncCollector {
	return &SyncCollector{}
}

// Run takes a CollectorFunc to execute and zero or more parameters to pass to that
// function and immediately starts executing the function.
func (c *SyncCollector) Run(f CollectorFunc, args ...interface{}) {

}

// Wait will wait until all tasks associated with the Collector have finished and then
// will return the results of those functions.  Wait will wait for results until it has
// not received a task result in 'timeout' amount of time.  If timeout is 0, Wait will
// wait indefinitely for tasks to finish.
func (c *SyncCollector) Wait(timeout time.Duration) ([]TaskResult, error) {
	return nil, nil
}

// WaitCloser will wait similar to Wait except if an error occurs while waiting
// for tasks to finish, closer will be called on each task result as they finish.
func (c *SyncCollector) WaitCloser(timeout time.Duration, closer CollectorCloser) ([]TaskResult, error) {
	return nil, nil
}
