package boom

import (
	"reflect"
	"sync"
	"time"
)

// CollectorFunc is the signature for function a Collector runs to perform a task
type CollectorFunc func(data ...interface{}) *TaskResult

// Collector can take a number of tasks, execute them in parallel, and then collect
// the results of those tasks once all have been completed.
type Collector struct {
	lock      sync.Mutex
	waitCount int
	tasks     []*collectorTask
	results   []*TaskResult
}

// NewCollector creates a new Collector instance
func NewCollector() *Collector {
	return &Collector{
		waitCount: 0,
		tasks:     make([]*collectorTask, 0),
		results:   make([]*TaskResult, 0),
	}
}

// Run takes a CollectorFunc to execute and zero or more parameters to pass to that
// function and immediately starts executing the function.
func (c *Collector) Run(f CollectorFunc, args ...interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()

	task := newCollectorTask(f, args)
	c.tasks = append(c.tasks, task)
	c.results = append(c.results, nil)
	c.waitCount++
	task.Start()
}

// Wait will wait until all tasks associated with the Collector have finished and then
// will return the results of those functions.  Wait will wait for results until it has
// not received a task result in 'timeout' amount of time.  If timeout is 0, Wait will
// wait indefinitely for tasks to finish.
func (c *Collector) Wait(timeout time.Duration) ([]*TaskResult, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	chans := make([]reflect.SelectCase, len(c.tasks)+1)
	chans[0].Dir = reflect.SelectRecv

	for idx, task := range c.tasks {
		chans[idx+1].Dir = reflect.SelectRecv
		chans[idx+1].Chan = reflect.ValueOf(task.resChan)
	}

	if timeout > 0 {
		ticker := time.NewTicker(timeout)
		defer ticker.Stop()
		chans[0].Chan = reflect.ValueOf(ticker.C)
	}

	for {
		choice, val, _ := reflect.Select(chans)
		if choice == 0 {
			return nil, ErrTimeout
		}

		res := val.Interface().(*TaskResult)
		c.results[choice-1] = res
		c.waitCount--

		if c.waitCount == 0 {
			break
		}
	}

	return c.results, nil
}

type collectorTask struct {
	f       CollectorFunc
	args    []interface{}
	resChan chan interface{}
}

func newCollectorTask(f CollectorFunc, args []interface{}) *collectorTask {
	return &collectorTask{
		f:       f,
		args:    args,
		resChan: make(chan interface{}),
	}
}

func (ct *collectorTask) Start() {
	go ct.worker()
}

func (ct *collectorTask) worker() {
	res := ct.f(ct.args...)
	ct.resChan <- res
}
