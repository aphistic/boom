package boom

import (
	"context"
	"sync"
	"time"
)

// AsyncCollector can take a number of tasks, execute them in parallel, and then collect
// the results of those tasks once all have been completed.
type AsyncCollector struct {
	cfg       *taskConfig
	lock      sync.Mutex
	waitCount int
	tasks     []*collectorTask
	results   []TaskResult
	resChan   chan *collectorResult
}

// NewAsyncCollector creates a new AsyncCollector instance
func NewAsyncCollector(configs ...TaskConfig) *AsyncCollector {
	cfg := newTaskConfig()
	cfg.ApplyConfigs(configs)

	return &AsyncCollector{
		cfg:       cfg,
		waitCount: 0,
		tasks:     make([]*collectorTask, 0),
		results:   make([]TaskResult, 0),
		resChan:   make(chan *collectorResult),
	}
}

func (c *AsyncCollector) cleanup(closer CollectorCloser) {
	for _, res := range c.results {
		if res != nil {
			closer(res)
		}
	}

	for {
		res := <-c.resChan
		closer(res.Result)

		c.lock.Lock()
		c.waitCount--
		if c.waitCount == 0 {
			return
		}
		c.lock.Unlock()
	}

}

// Run takes a TaskFunc to execute and zero or more parameters to pass to that
// function and immediately starts executing the function.
func (c *AsyncCollector) Run(f TaskFunc, args ...interface{}) {
	c.RunWithContext(context.Background(), f, args...)
}

// RunWithContext calls Run and uses the provided context.Context to run the task.
func (c *AsyncCollector) RunWithContext(ctx context.Context, f TaskFunc, args ...interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()

	task := newTask(ctx, c.cfg, f, args...)

	colTask := newCollectorTask(task, c.resChan)
	c.tasks = append(c.tasks, colTask)
	c.results = append(c.results, nil)
	c.waitCount++
	colTask.Start(len(c.results) - 1)
}

// Wait will wait until all tasks associated with the Collector have finished and then
// will return the results of those functions.  Wait will wait for results until it has
// not received a task result in 'timeout' amount of time.  If timeout is 0, Wait will
// wait indefinitely for tasks to finish.
func (c *AsyncCollector) Wait(timeout time.Duration) ([]TaskResult, error) {
	return c.WaitCloser(timeout, nil)
}

// WaitCloser will wait similar to Wait except if an error occurs while waiting
// for tasks to finish, closer will be called on each task result as they finish.
func (c *AsyncCollector) WaitCloser(timeout time.Duration, closer CollectorCloser) ([]TaskResult, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.waitCount == 0 {
		return c.results, nil
	}

	var timeoutChan <-chan time.Time
	if timeout > 0 {
		timeoutChan = c.cfg.clock.After(timeout)
	} else {
		timeoutChan = make(chan time.Time)
	}

	for {
		select {
		case res := <-c.resChan:
			c.results[res.Choice] = res.Result
			c.waitCount--

			if c.waitCount == 0 {
				return c.results, nil
			}
		case <-timeoutChan:
			if closer != nil {
				go c.cleanup(closer)
			}
			return nil, ErrTimeout
		}
	}
}

type collectorResult struct {
	Choice int
	Result TaskResult
}

type collectorTask struct {
	task    *Task
	closer  CollectorCloser
	resChan chan<- *collectorResult
}

func newCollectorTask(task *Task, resChan chan<- *collectorResult) *collectorTask {
	return &collectorTask{
		task:    task,
		resChan: resChan,
	}
}

func (ct *collectorTask) Start(choice int) {
	go ct.worker(choice)
}

func (ct *collectorTask) worker(choice int) {
	res, _ := ct.task.StartSync()
	ct.resChan <- &collectorResult{
		Choice: choice,
		Result: res,
	}
}
