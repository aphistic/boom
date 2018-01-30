package boom

import (
	"context"
)

// TaskRunner is a way to start one-off tasks where the collection of results
// does not matter, only each individual task.
type TaskRunner struct {
	cfg *taskConfig
}

// NewTaskRunner creates a new TaskRunner instance.
func NewTaskRunner(configs ...TaskConfig) *TaskRunner {
	cfg := newTaskConfig()
	cfg.ApplyConfigs(configs)

	return &TaskRunner{
		cfg: cfg,
	}
}

// New creates a new task with the given function and arguments
func (tr *TaskRunner) New(f TaskFunc, args ...interface{}) *Task {
	return newTask(context.Background(), tr.cfg, f, args...)
}

// NewWithContext creates a new task with the given context, function and arguments
func (tr *TaskRunner) NewWithContext(ctx context.Context, f TaskFunc, args ...interface{}) *Task {
	return newTask(ctx, tr.cfg, f, args...)
}

// Run will create a new task and immediately call Start to begin
// execution of the task.
func (tr *TaskRunner) Run(f TaskFunc, args ...interface{}) *Task {
	return runTask(context.Background(), tr.cfg, f, args...)
}

// RunWithContext calls Run using the provided context.Context for the task
func (tr *TaskRunner) RunWithContext(ctx context.Context, f TaskFunc, args ...interface{}) *Task {
	return runTask(ctx, tr.cfg, f, args...)
}
