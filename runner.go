package boom

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

// NewTask creates a new task with the given function and arguments
func (tr *TaskRunner) NewTask(f TaskFunc, args ...interface{}) *Task {
	return newTask(tr.cfg, f, args...)
}

// RunTask will create a new task and immediately call Start to begin
// execution of the task.
func (tr *TaskRunner) RunTask(f TaskFunc, args ...interface{}) *Task {
	return runTask(tr.cfg, f, args...)
}
