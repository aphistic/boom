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

	statusLock     sync.RWMutex
	started        bool
	finished       bool
	resultReadChan chan struct{}

	runLock sync.RWMutex
	runChan chan struct{}
	running bool

	resultLock  sync.RWMutex
	resultChan  chan TaskResult
	waitResult  TaskResult
	discardOnce sync.Once
}

// newTask creates a new task with the given function and arguments
func newTask(ctx context.Context, cfg *taskConfig, f TaskFunc, args ...interface{}) *Task {
	ctx, cancelCtx := context.WithCancel(ctx)

	return &Task{
		cfg: cfg,

		ctx:       ctx,
		cancelCtx: cancelCtx,

		f:       f,
		args:    args,
		runChan: make(chan struct{}),

		resultChan:     make(chan TaskResult),
		resultReadChan: make(chan struct{}),
	}
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

// Start will begin execution of the task in a separate goroutine.
func (t *Task) Start() error {
	t.statusLock.Lock()
	if t.started {
		t.statusLock.Unlock()
		return ErrExecuting
	}
	if t.finished {
		t.statusLock.Unlock()
		return ErrFinished
	}
	t.started = true
	t.statusLock.Unlock()

	go func(task *Task) {
		res := task.f(t, task.args...)

		task.statusLock.Lock()
		task.started = false
		task.finished = true
		task.statusLock.Unlock()

		task.resultChan <- res

		close(task.resultReadChan)
	}(t)

	return nil
}

// Started returns whether Start() has been called or not.
func (t *Task) Started() bool {
	t.statusLock.RLock()
	defer t.statusLock.RUnlock()

	return t.started
}

// SetRunning is a utility provided to users to signal whether a task is
// actively running or not.  Typically this would be used within the task
// function itself to signal that it has completed any setup it needed to
// do and has started processing data.
func (t *Task) SetRunning(running bool) {
	t.runLock.Lock()
	defer t.runLock.Unlock()

	if !running {
		t.runChan = make(chan struct{})
	} else {
		// If we're not already running close the run channel
		// so anything waiting will be triggered.
		if !t.running {
			close(t.runChan)
		}
	}
	t.running = running
}

// Running returns whether this task has been set in the 'Running' state or
// not. See SetRunning for more information
func (t *Task) Running() bool {
	t.runLock.RLock()
	defer t.runLock.RUnlock()

	return t.running
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
	case <-t.runChan:
		return nil
	case <-timeoutChan:
		return ErrTimeout
	}
}

// Stop signals a started task to stop. It is up to the task
// itself to check Task.Stopping() to see if it should stop.
func (t *Task) Stop() error {
	t.statusLock.Lock()
	if !t.started || t.IsStopping() {
		t.statusLock.Unlock()
		return ErrNotExecuting
	}

	t.cancelCtx()
	t.statusLock.Unlock()

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
		}()
	})
}

// Wait will wait for a task to end and return the TaskResult from
// the task over the returned channel. If a non-zero timeout is provided,
// Wait will wait until the timeout duration and close the channel. Either Discard or Wait must
// be called or the task's goroutine will leak.
func (t *Task) Wait(timeout time.Duration) (TaskResult, error) {
	t.statusLock.RLock()
	if !t.started && !t.finished {
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
		timeoutChan = t.cfg.clock.After(timeout)
	}

	select {
	case res := <-t.resultChan:
		t.completed(res)
		t.SetRunning(false)
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

// Stopping returns a channel that is closed if a task is stopping (set by
// the Stop method)
func (t *Task) Stopping() <-chan struct{} {
	return t.ctx.Done()
}

// IsStopping is a convenience method to check if the Stopping() channel is
// closed
func (t *Task) IsStopping() bool {
	select {
	case <-t.ctx.Done():
		return true
	default:
		return false
	}
}

func (t *Task) completed(result TaskResult) {
	t.resultLock.Lock()
	defer t.resultLock.Unlock()

	close(t.resultChan)
	t.resultChan = nil
	t.waitResult = result
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
