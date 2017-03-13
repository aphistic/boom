package boom

import (
	"reflect"
	"sync"
	"time"
)

var nilValue = reflect.ValueOf(nil)

// TaskFunc is the signature for the function executed to perform a Task
type TaskFunc func(task *Task, data ...interface{}) TaskResult

// TaskResult is the result expected from a boom task when it's finished executing
type TaskResult interface {
	Err() error
}

// Task represents a task being executed within boom
type Task struct {
	f    TaskFunc
	args []interface{}

	statusLock sync.RWMutex
	running    bool
	stopChan   chan struct{}
	finished   bool

	resultLock sync.RWMutex
	resultChan chan TaskResult
	waitResult TaskResult
}

// NewTask creates a new task with the given function and arguments
func NewTask(f TaskFunc, args ...interface{}) *Task {
	return &Task{
		f:        f,
		args:     args,
		stopChan: make(chan struct{}),
	}
}

// RunTask will create a new task and immediately call Start() to
// begin executing the task.
func RunTask(f TaskFunc, args ...interface{}) *Task {
	task := NewTask(f, args...)
	task.Start()
	return task
}

// Start will begin execution of the task in a separate goroutine.
func (t *Task) Start() error {
	t.statusLock.Lock()
	if t.running {
		t.statusLock.Unlock()
		return ErrExecuting
	}
	if t.finished {
		t.statusLock.Unlock()
		return ErrFinished
	}
	t.running = true
	t.statusLock.Unlock()

	t.resultLock.Lock()
	t.resultChan = make(chan TaskResult)
	t.resultLock.Unlock()

	go func(task *Task) {
		res := task.f(t, task.args...)
		task.statusLock.Lock()
		task.running = false
		task.finished = true
		task.statusLock.Unlock()
		task.resultChan <- res
	}(t)

	return nil
}

func (t *Task) stopFlag() bool {
	select {
	case <-t.stopChan:
		return true
	default:
		return false
	}
}

// Stop signals a running task to stop running. It is up to the task
// itself to check Task.Stopping() to see if it should stop.
func (t *Task) Stop() error {
	t.statusLock.Lock()
	if !t.running || t.stopFlag() {
		t.statusLock.Unlock()
		return ErrNotExecuting
	}

	close(t.stopChan)
	t.statusLock.Unlock()

	return nil
}

// Wait will wait for a task to end and return the TaskResult from
// the task over the returned channel. If a non-zero timeout is provided,
// Wait will wait until the timeout duration and close the channel
func (t *Task) Wait(timeout time.Duration) (TaskResult, error) {
	t.statusLock.RLock()
	if !t.running && !t.finished {
		t.statusLock.RUnlock()
		return nil, ErrNotExecuting
	}
	t.statusLock.RUnlock()
	t.resultLock.RLock()
	if t.waitResult != nil {
		t.resultLock.RUnlock()
		return t.waitResult, nil
	}
	t.resultLock.RUnlock()

	var timeoutChan <-chan time.Time = make(chan time.Time)
	if timeout > 0 {
		timeoutChan = time.After(timeout)
	}

	select {
	case res := <-t.resultChan:
		t.resultLock.Lock()
		close(t.resultChan)
		t.resultChan = nil
		t.waitResult = res
		t.resultLock.Unlock()
		return res, nil
	case <-timeoutChan:
	}
	return nil, ErrTimeout
}

// StopAndWait is a convenience function for calling both Stop() and Wait() in
// a single call.
func (t *Task) StopAndWait(timeout time.Duration) (TaskResult, error) {
	err := t.Stop()
	if err != nil {
		return nil, err
	}
	res, err := t.Wait(timeout)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// StartSync will run a task synchronously
func (t *Task) StartSync() (TaskResult, error) {
	t.Start()
	res, err := t.Wait(0)
	return res, err
}

// Finished returns whether the task has finished executing or not.
func (t *Task) Finished() bool {
	t.statusLock.RLock()
	defer t.statusLock.RUnlock()

	return t.finished
}

// Running returns whether the task is currently executing or not.
func (t *Task) Running() bool {
	t.statusLock.RLock()
	defer t.statusLock.RUnlock()

	return t.running
}

// Stopping returns whether a task is stopping (set by the Stop method)
func (t *Task) Stopping() <-chan struct{} {
	return t.stopChan
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
