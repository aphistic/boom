package boom

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aphistic/sweet"
	junit "github.com/aphistic/sweet-junit"
	. "github.com/onsi/gomega"
)

func TestMain(m *testing.M) {
	RegisterFailHandler(sweet.GomegaFail)

	sweet.Run(m, func(s *sweet.S) {
		s.RegisterPlugin(junit.NewPlugin())

		s.AddSuite(&TaskSuite{})
		s.AddSuite(&RunnerSuite{})
		s.AddSuite(&AsyncColSuite{})
	})
}

type TaskSuite struct{}

const waitTimeout = 10 * time.Millisecond

func ExampleTask() {
	// Create a new task but don't start execution right away
	t := newTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		// Run the task until something requests that we stop
		<-task.Stopping()
		return NewValueResult(args, nil)
	}, "first", "second", 3)

	// Start the Task. Another way to do this in a single command is to use
	// runTask(newTaskConfig(), ) to create a new task and start it right away
	t.Start()

	// Let the task run a little bit
	time.Sleep(100 * time.Millisecond)

	// Ask the task to stop
	t.Stop()

	// Wait forever for the task to finish running and get
	// the result from it
	res, _ := t.Wait(0)

	valRes := res.(*ValueResult)

	fmt.Printf("Value: %+v\n", valRes.Value)
	fmt.Printf("Error: %+v\n", valRes.Error)

	// Output:
	// Value: [first second 3]
	// Error: <nil>
}

func (s *TaskSuite) TestStartSync(t sweet.T) {
	task := newTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		return NewValueResult(args[0], nil)
	}, 1)

	res, err := task.StartSync()
	Expect(err).To(BeNil())
	Expect(res).To(Equal(&ValueResult{Value: 1, Error: nil}))
}

func (s *TaskSuite) TestContext(t sweet.T) {
	var taskCtx context.Context

	ctx := context.Background()
	task := runTask(ctx, newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		taskCtx = task.Context()
		return nil
	})
	task.Wait(0)

	Expect(taskCtx).ToNot(BeNil())
}

func (s *TaskSuite) TestContextCancelStopsTask(t sweet.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	task := newTask(ctx, newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		return nil
	})

	Expect(ctx.Done()).ToNot(BeClosed())
	Expect(task.Stopping()).ToNot(BeClosed())

	cancelCtx()

	Eventually(ctx.Done()).Should(BeClosed())
	Eventually(task.Stopping()).Should(BeClosed())
}

func (s *TaskSuite) TestStart(t sweet.T) {
	task := runTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		return NewValueResult(args[0], nil)
	}, 1)

	res, err := task.Wait(waitTimeout)
	Expect(err).To(BeNil())
	Expect(res).To(Equal(&ValueResult{Value: 1, Error: nil}))
}

func (s *TaskSuite) TestStartStop(t sweet.T) {
	task := runTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()

		return NewValueResult(args[0], nil)
	}, 1)

	task.Stop()

	res, err := task.Wait(waitTimeout)
	Expect(err).To(BeNil())
	Expect(res).To(Equal(&ValueResult{Value: 1, Error: nil}))
}

func (s *TaskSuite) TestStartFinished(t sweet.T) {
	task := runTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		return NewValueResult(args[0], nil)
	}, 1)

	res, err := task.Wait(waitTimeout)
	Expect(err).To(BeNil())
	Expect(res).To(Equal(&ValueResult{Value: 1, Error: nil}))

	err = task.Start()
	Expect(err).To(Equal(ErrFinished))
}

func (s *TaskSuite) TestFinished(t sweet.T) {
	task := runTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		return NewValueResult(args[0], nil)
	}, 1)

	res, err := task.Wait(waitTimeout)
	Expect(err).To(BeNil())
	Expect(res).To(Equal(&ValueResult{Value: 1, Error: nil}))

	Eventually(task.Finished()).Should(BeClosed())
	Expect(task.resultChan).To(BeClosed())
}

func (s *TaskSuite) TestStarted(t sweet.T) {
	task := runTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()
		return NewValueResult(args[0], nil)
	}, 1)

	Expect(task.Started()).To(BeClosed())

	task.Stop()
	task.Wait(waitTimeout)
}

func (s *TaskSuite) TestRunning(t sweet.T) {
	advance := make(chan int)
	advanced := make(chan int)
	task := runTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		//c.Log("Started task")
		<-advance
		//c.Log("Setting running to true")
		task.SetRunning(true)
		advanced <- 1
		<-advance
		//c.Log("Setting running to false")
		task.SetRunning(false)
		advanced <- 1
		<-task.Stopping()

		//c.Log("Task returning")
		return NewValueResult(nil, nil)
	})

	Expect(task.Running()).ToNot(BeClosed())
	//c.Log("Advancing to running")
	advance <- 1
	<-advanced
	Expect(task.Running()).To(BeClosed())
	//c.Log("Advancing to not running")
	advance <- 1
	<-advanced
	Expect(task.Running()).ToNot(BeClosed())

	//c.Log("Stopping")
	task.Stop()
}

func (s *TaskSuite) TestWaitForRunning(t sweet.T) {
	task := runTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		task.SetRunning(true)
		<-task.Stopping()

		return NewValueResult(nil, nil)
	})

	err := task.WaitForRunning(1 * time.Second)
	Expect(err).To(BeNil())

	_, err = task.StopAndWait(1 * time.Second)
	Expect(err).To(BeNil())
}

func (s *TaskSuite) TestWaitForRunningTimeout(t sweet.T) {
	task := runTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()
		return NewValueResult(nil, nil)
	})

	err := task.WaitForRunning(100 * time.Millisecond)
	Expect(err).To(Equal(ErrTimeout))
	_, err = task.StopAndWait(1 * time.Second)
	Expect(err).To(BeNil())
}

func (s *TaskSuite) TestWaitForRunningTaskFinished(t sweet.T) {
	task := runTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		return NewErrorResult(errors.New("I'm an error! - Ralph"))
	})

	err := task.WaitForRunning(100 * time.Millisecond)
	Expect(err).To(BeNil())
	res, err := task.Wait(100 * time.Millisecond)
	Expect(err).To(BeNil())
	Expect(res).To(Equal(NewErrorResult(errors.New("I'm an error! - Ralph"))))
}

func (s *TaskSuite) TestRunningSetFalseWhenFinished(t sweet.T) {
	task := runTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		task.SetRunning(true)
		<-task.Stopping()
		return NewValueResult(nil, nil)
	})

	err := task.WaitForRunning(100 * time.Millisecond)
	Expect(err).To(BeNil())
	_, err = task.StopAndWait(1 * time.Second)
	Expect(err).To(BeNil())
	Expect(task.Running()).ToNot(BeClosed())
}

func (s *TaskSuite) TestWaitTwice(t sweet.T) {
	task := newTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		task.SetRunning(true)
		<-task.Stopping()
		return NewValueResult(args[0], nil)
	}, 1)

	task.Start()

	task.WaitForRunning(0)

	task.Stop()

	res, err := task.Wait(waitTimeout)
	Expect(err).To(BeNil())
	Expect(res).To(Equal(&ValueResult{Value: 1, Error: nil}))
	Expect(task.resultChan).To(BeClosed())

	res, err = task.Wait(waitTimeout)
	Expect(err).To(BeNil())
	Expect(res).To(Equal(&ValueResult{Value: 1, Error: nil}))
	Expect(task.resultChan).To(BeClosed())
}

func (s *TaskSuite) TestWaitTimeout(t sweet.T) {
	task := newTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		time.Sleep(100 * time.Millisecond)
		return NewValueResult(args[0], nil)
	}, 1)

	task.Start()

	res, err := task.Wait(10 * time.Millisecond)
	Expect(err).To(Equal(ErrTimeout))
	Expect(res).To(BeNil())
	Expect(task.resultChan).ToNot(BeClosed())

	res, err = task.Wait(200 * time.Millisecond)
	Expect(err).To(BeNil())
	Expect(res).To(Equal(&ValueResult{Value: 1, Error: nil}))
	Expect(task.resultChan).To(BeClosed())
}

func (s *TaskSuite) TestDiscardBeforeRunning(t sweet.T) {
	task := newTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		return NewValueResult(nil, nil)
	})

	task.Discard()
	task.Start()

	Eventually(task.resultChan).Should(BeClosed())
}

func (s *TaskSuite) TestDiscardAfterRunning(t sweet.T) {
	task := newTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		return NewValueResult(nil, nil)
	})

	task.Start()

	// Yield to allow the runtime to start goroutines
	time.Sleep(time.Millisecond)

	task.Discard()

	// Yield to allow the runtime to start goroutines
	time.Sleep(time.Millisecond)

	Eventually(task.resultChan).Should(BeClosed())
}

func (s *TaskSuite) TestStartStarted(t sweet.T) {
	task := newTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()
		return NewValueResult(nil, nil)
	})

	err := task.Start()
	Expect(err).To(BeNil())
	err = task.Start()
	Expect(err).ToNot(BeNil())
	Expect(err).To(Equal(ErrExecuting))

	task.Stop()
	task.Wait(waitTimeout)
}

func (s *TaskSuite) TestStopStopped(t sweet.T) {
	task := newTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()
		return NewValueResult(nil, nil)
	})

	err := task.Stop()
	Expect(err).To(Equal(ErrNotExecuting))
}

func (s *TaskSuite) TestWaitStopped(t sweet.T) {
	task := newTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()
		return NewValueResult(nil, nil)
	})

	res, err := task.Wait(waitTimeout)
	Expect(err).To(Equal(ErrNotExecuting))
	Expect(res).To(BeNil())
}

func (s *TaskSuite) TestStopAndWait(t sweet.T) {
	task := runTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()
		return NewValueResult(1, nil)
	})

	Eventually(task.Started()).Should(BeClosed())

	res, err := task.StopAndWait(waitTimeout)
	Expect(err).To(BeNil())
	Expect(res).To(Equal(&ValueResult{Value: 1, Error: nil}))
}

func (s *TaskSuite) TestStopAndWaitWhileStopped(t sweet.T) {
	task := newTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()
		return NewValueResult(1, nil)
	})

	res, err := task.StopAndWait(waitTimeout)
	Expect(err).To(Equal(ErrNotExecuting))
	Expect(res).To(BeNil())
}

func (s *TaskSuite) TestStopAndWaitTimeout(t sweet.T) {
	task := runTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		time.Sleep(25 * time.Millisecond)
		<-task.Stopping()
		return NewValueResult(1, nil)
	})

	res, err := task.StopAndWait(10 * time.Millisecond)
	Expect(err).To(Equal(ErrTimeout))
	Expect(res).To(BeNil())
}

func (s *TaskSuite) TestNilResult(t sweet.T) {
	task := runTask(context.Background(), newTaskConfig(), func(task *Task, args ...interface{}) TaskResult {
		return nil
	})

	res, err := task.Wait(waitTimeout)
	Expect(err).To(BeNil())
	Expect(res).To(BeNil())
}

func (s *TaskSuite) TestValueResultErr(t sweet.T) {
	res := NewValueResult(1234, errors.New("I'm an error - Ralph"))
	Expect(res.Err()).To(Equal(errors.New("I'm an error - Ralph")))
}

func (s *TaskSuite) TestErrorResultErr(t sweet.T) {
	res := NewErrorResult(errors.New("I'm an error - Ralph"))
	Expect(res.Err()).To(Equal(errors.New("I'm an error - Ralph")))
}
