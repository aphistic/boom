package boom

import (
	"context"
	"time"

	"github.com/aphistic/sweet"
	"github.com/efritz/glock"
	. "github.com/onsi/gomega"
)

type RunnerSuite struct{}

func (s *RunnerSuite) TestNew(t sweet.T) {
	clock := glock.NewMockClockAt(time.Unix(100, 0))

	tr := NewTaskRunner(WithClock(clock))
	task := tr.New(func(task *Task, args ...interface{}) TaskResult {
		Expect(args[0]).To(Equal(1))
		Expect(args[1]).To(Equal(2))
		Expect(args[2]).To(Equal(3))

		return NewValueResult(5, nil)
	}, 1, 2, 3)

	Expect(task.cfg.clock).To(Equal(clock))

	res, err := task.StartSync()
	Expect(err).To(BeNil())
	Expect(res).To(Equal(NewValueResult(5, nil)))
}

func (s *RunnerSuite) TestNewWithContext(t sweet.T) {
	clock := glock.NewMockClockAt(time.Unix(100, 0))
	ctx := context.WithValue(context.Background(), "test", 1234)

	var taskCtx context.Context

	tr := NewTaskRunner(WithClock(clock))
	task := tr.NewWithContext(ctx, func(task *Task, args ...interface{}) TaskResult {
		Expect(args[0]).To(Equal(1))
		Expect(args[1]).To(Equal(2))
		Expect(args[2]).To(Equal(3))

		taskCtx = task.Context()

		return NewValueResult(5, nil)
	}, 1, 2, 3)

	Expect(task.cfg.clock).To(Equal(clock))

	res, err := task.StartSync()
	Expect(err).To(BeNil())
	Expect(res).To(Equal(NewValueResult(5, nil)))

	Expect(taskCtx.Value("test")).To(Equal(1234))
}

func (s *RunnerSuite) TestRun(t sweet.T) {
	clock := glock.NewRealClock()

	tr := NewTaskRunner(WithClock(clock))
	task := tr.Run(func(task *Task, args ...interface{}) TaskResult {
		Expect(args[0]).To(Equal(1))
		Expect(args[1]).To(Equal(2))
		Expect(args[2]).To(Equal(3))

		return NewValueResult(5, nil)
	}, 1, 2, 3)

	Expect(task.cfg.clock).To(Equal(clock))

	res, err := task.Wait(1 * time.Second)
	Expect(err).To(BeNil())
	Expect(res).To(Equal(NewValueResult(5, nil)))
}

func (s *RunnerSuite) TestRunWithContext(t sweet.T) {
	clock := glock.NewRealClock()

	ctx := context.WithValue(context.Background(), "test", 1234)

	var taskCtx context.Context

	tr := NewTaskRunner(WithClock(clock))
	task := tr.RunWithContext(ctx, func(task *Task, args ...interface{}) TaskResult {
		Expect(args[0]).To(Equal(1))
		Expect(args[1]).To(Equal(2))
		Expect(args[2]).To(Equal(3))

		taskCtx = task.Context()

		return NewValueResult(5, nil)
	}, 1, 2, 3)

	Expect(task.cfg.clock).To(Equal(clock))

	res, err := task.Wait(1 * time.Second)
	Expect(err).To(BeNil())
	Expect(res).To(Equal(NewValueResult(5, nil)))

	Expect(taskCtx.Value("test")).To(Equal(1234))
}
