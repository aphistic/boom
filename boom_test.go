package boom

import (
	"errors"
	"testing"

	"time"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type TaskSuite struct{}

var _ = Suite(&TaskSuite{})

const waitTimeout = 10 * time.Millisecond

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
		for !task.Stopping() {
			time.Sleep(1 * time.Millisecond)
		}

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

func (s *TaskSuite) TestRunning(c *C) {
	t := RunTask(func(task *Task, args ...interface{}) TaskResult {
		for !task.Stopping() {
			time.Sleep(10 * time.Millisecond)
		}
		return NewValueResult(args[0], nil)
	}, 1)

	c.Check(t.Running(), Equals, true)

	t.Stop()
	t.Wait(waitTimeout)
}

func (s *TaskSuite) TestWaitTwice(c *C) {
	t := NewTask(func(task *Task, args ...interface{}) TaskResult {
		for !task.Stopping() {
			time.Sleep(1 * time.Millisecond)
		}

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
		for !task.Stopping() {
			time.Sleep(1 * time.Millisecond)
		}

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
		for !task.Stopping() {
			time.Sleep(1 * time.Millisecond)
		}

		return NewValueResult(nil, nil)
	})

	err := t.Stop()
	c.Assert(err, Equals, ErrNotExecuting)
}

func (s *TaskSuite) TestWaitStopped(c *C) {
	t := NewTask(func(task *Task, args ...interface{}) TaskResult {
		for !task.Stopping() {
			time.Sleep(1 * time.Millisecond)
		}

		return NewValueResult(nil, nil)
	})

	res, err := t.Wait(waitTimeout)
	c.Assert(err, Equals, ErrNotExecuting)
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
	res := NewValueResult(1234, errors.New("I'm an error"))
	c.Check(res.Err(), DeepEquals, errors.New("I'm an error"))
}
