package boom

import (
	"context"
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
	cfg *taskConfig

	ctx       context.Context
	cancelCtx context.CancelFunc

	f    TaskFunc
	args []interface{}

	startedChan  chan struct{}
	runningChan  chan struct{}
	finishedChan chan struct{}

	resultLock sync.RWMutex
	resultChan chan TaskResult
	waitResult TaskResult

	discardOnce  sync.Once
	completeOnce sync.Once
}

// newTask creates a new task with the given function and arguments
func newTask(ctx context.Context, cfg *taskConfig, f TaskFunc, args ...interface{}) *Task {
	ctx, cancelCtx := context.WithCancel(ctx)

	task := &Task{
		cfg: cfg,

		ctx:       ctx,
		cancelCtx: cancelCtx,

		f:    f,
		args: args,

		startedChan:  make(chan struct{}),
		runningChan:  make(chan struct{}),
		finishedChan: make(chan struct{}),

		resultChan: make(chan TaskResult),
	}

	return task
}

// RunTask will create a new task and immediately call Start() to
// begin executing the task.
func runTask(ctx context.Context, cfg *taskConfig, f TaskFunc, args ...interface{}) *Task {
	task := newTask(ctx, cfg, f, args...)
	task.Start()
	return task
}

// Context returns the context for the task
func (t *Task) Context() context.Context {
	return t.ctx
}

// Started returns a channel that is closed if the task has been started.
func (t *Task) Started() <-chan struct{} {
	return t.startedChan
}

// Running returns a channel that is closed if the task has been set to the 'Running' state
// at some point. See SetRunning for how to set the value.
func (t *Task) Running() <-chan struct{} {
	return t.runningChan
}

// Stopping returns a channel that is closed if a task is stopping (set by
// the Stop method)
func (t *Task) Stopping() <-chan struct{} {
	return t.ctx.Done()
}

// Finished returns a channel that will be closed if the task has finished running.
func (t *Task) Finished() <-chan struct{} {
	return t.finishedChan
}

// Start will begin execution of the task in a separate goroutine.
func (t *Task) Start() error {
	// If the task has already finished, return an error
	select {
	case <-t.finishedChan:
		return ErrFinished
	default:
	}

	// If the task hasn't finished but has already started, return
	// a different error
	select {
	case <-t.startedChan:
		return ErrExecuting
	default:
	}

	close(t.startedChan)

	go func(task *Task) {
		res := task.f(t, task.args...)

		close(t.finishedChan)

		task.resultChan <- res
	}(t)

	return nil
}

// SetRunning is a utility provided to users to signal whether a task is
// actively running or not.  Typically this would be used within the task
// function itself to signal that it has completed any setup it needed to
// do and has started processing data.
func (t *Task) SetRunning(running bool) {
	if !running {
		t.runningChan = make(chan struct{})
	} else {
		select {
		case <-t.runningChan:
		default:
			close(t.runningChan)
		}
	}
}

// WaitForRunning will block until a task enters the 'Running' state or will return
// immediately if the task is already in the 'Running' state.
func (t *Task) WaitForRunning(timeout time.Duration) error {
	var timeoutChan <-chan time.Time = make(chan time.Time)
	if timeout > 0 {
		timeoutChan = t.cfg.clock.After(timeout)
	}
	select {
	case res := <-t.resultChan:
		t.completed(res)
		return nil
	case <-t.runningChan:
		return nil
	case <-timeoutChan:
		return ErrTimeout
	}
}

// Stop signals a started task to stop. It is up to the task
// itself to check Task.Stopping() to see if it should stop.
func (t *Task) Stop() error {
	select {
	case <-t.startedChan:
	default:
		return ErrNotExecuting
	}

	t.cancelCtx()

	return nil
}

// Discard will discard the result returned from the task.  Either Discard or Wait must be
// called or the task's goroutine will leak.
func (t *Task) Discard() {
	// Start a goroutine to listen to the task result channel and discard the result,
	// then exit.
	t.discardOnce.Do(func() {
		go func() {
			<-t.resultChan
			t.completed(nil)
		}()
	})
}

// Wait will wait for a task to end and return the TaskResult from
// the task over the returned channel. If a non-zero timeout is provided,
// Wait will wait until the timeout duration and close the channel. Either Discard or Wait must
// be called or the task's goroutine will leak.
func (t *Task) Wait(timeout time.Duration) (TaskResult, error) {
	select {
	case <-t.startedChan:
	default:
		return nil, ErrNotExecuting
	}

	t.resultLock.RLock()
	if t.waitResult != nil {
		t.resultLock.RUnlock()
		return t.waitResult, nil
	}
	t.resultLock.RUnlock()

	var timeoutChan <-chan time.Time = make(chan time.Time)
	if timeout > 0 {
		timeoutChan = t.cfg.clock.After(timeout)
	}

	select {
	case res := <-t.resultChan:
		t.completed(res)
		t.SetRunning(false)
		return res, nil
	case <-timeoutChan:
		return nil, ErrTimeout
	}
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

func (t *Task) completed(result TaskResult) {
	t.completeOnce.Do(func() {
		t.resultLock.Lock()
		defer t.resultLock.Unlock()

		close(t.resultChan)
		t.waitResult = result
	})
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

type ErrorResult struct {
	Error error
}

func NewErrorResult(err error) *ErrorResult {
	return &ErrorResult{
		Error: err,
	}
}

func (r *ErrorResult) Err() error {
	return r.Error
}
