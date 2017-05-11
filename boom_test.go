package boom

import (
	"errors"
	"testing"

	"time"

	"fmt"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type TaskSuite struct{}

var _ = Suite(&TaskSuite{})

const waitTimeout = 10 * time.Millisecond

func ExampleTask() {
	// Create a new task but don't start execution right away
	t := NewTask(func(task *Task, args ...interface{}) TaskResult {
		// Run the task until something requests that we stop
		<-task.Stopping()
		return NewValueResult(args, nil)
	}, "first", "second", 3)

	// Start the Task. Another way to do this in a single command is to use
	// RunTask() to create a new task and start it right away
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

func (s *TaskSuite) TestStartSync(c *C) {
	t := NewTask(func(task *Task, args ...interface{}) TaskResult {
		return NewValueResult(args[0], nil)
	}, 1)

	res, err := t.StartSync()
	c.Assert(err, IsNil)
	c.Check(res, DeepEquals, &ValueResult{Value: 1, Error: nil})
}

func (s *TaskSuite) TestStart(c *C) {
	t := RunTask(func(task *Task, args ...interface{}) TaskResult {
		return NewValueResult(args[0], nil)
	}, 1)

	res, err := t.Wait(waitTimeout)
	c.Assert(err, IsNil)
	c.Check(res, DeepEquals, &ValueResult{Value: 1, Error: nil})
}

func (s *TaskSuite) TestStartStop(c *C) {
	t := RunTask(func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()

		return NewValueResult(args[0], nil)
	}, 1)

	t.Stop()

	res, err := t.Wait(waitTimeout)
	c.Assert(err, IsNil)
	c.Check(res, DeepEquals, &ValueResult{Value: 1, Error: nil})
}

func (s *TaskSuite) TestStartFinished(c *C) {
	t := RunTask(func(task *Task, args ...interface{}) TaskResult {
		return NewValueResult(args[0], nil)
	}, 1)

	res, err := t.Wait(waitTimeout)
	c.Assert(err, IsNil)
	c.Check(res, DeepEquals, &ValueResult{Value: 1, Error: nil})

	err = t.Start()
	c.Check(err, Equals, ErrFinished)
}

func (s *TaskSuite) TestFinished(c *C) {
	t := RunTask(func(task *Task, args ...interface{}) TaskResult {
		return NewValueResult(args[0], nil)
	}, 1)

	res, err := t.Wait(waitTimeout)
	c.Assert(err, IsNil)
	c.Check(res, DeepEquals, &ValueResult{Value: 1, Error: nil})

	c.Check(t.Finished(), Equals, true)
}

func (s *TaskSuite) TestStarted(c *C) {
	t := RunTask(func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()
		return NewValueResult(args[0], nil)
	}, 1)

	c.Check(t.Started(), Equals, true)

	t.Stop()
	t.Wait(waitTimeout)
}

func (s *TaskSuite) TestRunning(c *C) {
	advance := make(chan int)
	advanced := make(chan int)
	t := RunTask(func(task *Task, args ...interface{}) TaskResult {
		c.Log("Started task")
		<-advance
		c.Log("Setting running to true")
		task.SetRunning(true)
		advanced <- 1
		<-advance
		c.Log("Setting running to false")
		task.SetRunning(false)
		advanced <- 1
		<-task.Stopping()

		c.Log("Task returning")
		return NewValueResult(nil, nil)
	})

	c.Check(t.Running(), Equals, false)
	c.Log("Advancing to running")
	advance <- 1
	<-advanced
	c.Check(t.Running(), Equals, true)
	c.Log("Advancing to not running")
	advance <- 1
	<-advanced
	c.Check(t.Running(), Equals, false)

	c.Log("Stopping")
	t.Stop()
}

func (s *TaskSuite) TestWaitForRunning(c *C) {
	t := RunTask(func(task *Task, args ...interface{}) TaskResult {
		task.SetRunning(true)
		<-task.Stopping()

		return NewValueResult(nil, nil)
	})

	err := t.WaitForRunning(1 * time.Second)
	c.Check(err, IsNil)

	_, err = t.StopAndWait(1 * time.Second)
	c.Check(err, IsNil)
}

func (s *TaskSuite) TestWaitForRunningTimeout(c *C) {
	t := RunTask(func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()
		return NewValueResult(nil, nil)
	})

	err := t.WaitForRunning(100 * time.Millisecond)
	c.Check(err, Equals, ErrTimeout)
	_, err = t.StopAndWait(1 * time.Second)
	c.Check(err, IsNil)
}

func (s *TaskSuite) TestWaitForRunningTaskFinished(c *C) {
	t := RunTask(func(task *Task, args ...interface{}) TaskResult {
		return NewErrorResult(errors.New("I'm an error! - Ralph"))
	})

	err := t.WaitForRunning(100 * time.Millisecond)
	c.Check(err, Equals, ErrFinished)
	res, err := t.Wait(100 * time.Millisecond)
	c.Check(err, IsNil)
	c.Check(res, DeepEquals, NewErrorResult(errors.New("I'm an error! - Ralph")))
}

func (s *TaskSuite) TestRunningSetFalseWhenFinished(c *C) {
	t := RunTask(func(task *Task, args ...interface{}) TaskResult {
		task.SetRunning(true)
		<-task.Stopping()
		return NewValueResult(nil, nil)
	})

	err := t.WaitForRunning(100 * time.Millisecond)
	c.Check(err, IsNil)
	_, err = t.StopAndWait(1 * time.Second)
	c.Check(err, IsNil)
	c.Check(t.Running(), Equals, false)
}

func (s *TaskSuite) TestWaitTwice(c *C) {
	t := NewTask(func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()
		return NewValueResult(args[0], nil)
	}, 1)

	t.Start()
	t.Stop()

	res, err := t.Wait(waitTimeout)
	c.Assert(err, IsNil)
	c.Check(res, DeepEquals, &ValueResult{Value: 1, Error: nil})

	res, err = t.Wait(waitTimeout)
	c.Assert(err, IsNil)
	c.Check(res, DeepEquals, &ValueResult{Value: 1, Error: nil})
}

func (s *TaskSuite) TestWaitTimeout(c *C) {
	t := NewTask(func(task *Task, args ...interface{}) TaskResult {
		time.Sleep(100 * time.Millisecond)
		return NewValueResult(args[0], nil)
	}, 1)

	t.Start()

	res, err := t.Wait(10 * time.Millisecond)
	c.Assert(err, Equals, ErrTimeout)
	c.Check(res, IsNil)

	res, err = t.Wait(200 * time.Millisecond)
	c.Assert(err, IsNil)
	c.Check(res, DeepEquals, &ValueResult{Value: 1, Error: nil})
}

func (s *TaskSuite) TestStartStarted(c *C) {
	t := NewTask(func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()
		return NewValueResult(nil, nil)
	})

	err := t.Start()
	c.Assert(err, IsNil)
	err = t.Start()
	c.Assert(err, NotNil)
	c.Check(err, Equals, ErrExecuting)

	t.Stop()
	t.Wait(waitTimeout)
}

func (s *TaskSuite) TestStopStopped(c *C) {
	t := NewTask(func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()
		return NewValueResult(nil, nil)
	})

	err := t.Stop()
	c.Assert(err, Equals, ErrNotExecuting)
}

func (s *TaskSuite) TestStopFlagStopped(c *C) {
	t := NewTask(func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()
		return NewValueResult(nil, nil)
	})

	err := t.Start()
	c.Assert(err, IsNil)

	c.Check(t.stopFlag(), Equals, false)

	_, err = t.StopAndWait(1 * time.Second)
	c.Assert(err, IsNil)

	c.Check(t.stopFlag(), Equals, true)
}

func (s *TaskSuite) TestWaitStopped(c *C) {
	t := NewTask(func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()
		return NewValueResult(nil, nil)
	})

	res, err := t.Wait(waitTimeout)
	c.Assert(err, Equals, ErrNotExecuting)
	c.Check(res, IsNil)
}

func (s *TaskSuite) TestStopAndWait(c *C) {
	t := RunTask(func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()
		return NewValueResult(1, nil)
	})

	c.Assert(t.Started(), Equals, true)

	res, err := t.StopAndWait(waitTimeout)
	c.Assert(err, IsNil)
	c.Check(res, DeepEquals, &ValueResult{Value: 1, Error: nil})
}

func (s *TaskSuite) TestStopAndWaitWhileStopped(c *C) {
	t := NewTask(func(task *Task, args ...interface{}) TaskResult {
		<-task.Stopping()
		return NewValueResult(1, nil)
	})

	res, err := t.StopAndWait(waitTimeout)
	c.Check(err, Equals, ErrNotExecuting)
	c.Check(res, IsNil)
}

func (s *TaskSuite) TestStopAndWaitTimeout(c *C) {
	t := RunTask(func(task *Task, args ...interface{}) TaskResult {
		time.Sleep(25 * time.Millisecond)
		<-task.Stopping()
		return NewValueResult(1, nil)
	})

	res, err := t.StopAndWait(10 * time.Millisecond)
	c.Check(err, Equals, ErrTimeout)
	c.Check(res, IsNil)
}

func (s *TaskSuite) TestNilResult(c *C) {
	t := RunTask(func(task *Task, args ...interface{}) TaskResult {
		return nil
	})

	res, err := t.Wait(waitTimeout)
	c.Assert(err, IsNil)
	c.Check(res, IsNil)
}

func (s *TaskSuite) TestValueResultErr(c *C) {
	res := NewValueResult(1234, errors.New("I'm an error - Ralph"))
	c.Check(res.Err(), DeepEquals, errors.New("I'm an error - Ralph"))
}

func (s *TaskSuite) TestErrorResultErr(c *C) {
	res := NewErrorResult(errors.New("I'm an error - Ralph"))
	c.Check(res.Err(), DeepEquals, errors.New("I'm an error - Ralph"))
}
