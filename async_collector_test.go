package boom

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/efritz/glock"
	. "github.com/onsi/gomega"
)

type AsyncColSuite struct{}

func (s *AsyncColSuite) TestFitsCollector(t *testing.T) {
	// Somehow I missed that AsyncCollector didn't match the Collector interface
	// so this is here to make sure I can't accidentally do that again.

	col := NewAsyncCollector()

	func(c Collector) {

	}(col)
}

func (s *AsyncColSuite) TestWaitTimeout(t *testing.T) {
	clock := glock.NewMockClock()

	col := NewAsyncCollector(WithClock(clock))

	var running sync.WaitGroup
	running.Add(3)
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		running.Done()
		clock.Sleep(15 * time.Millisecond)
		return NewValueResult(1, nil)
	})
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		running.Done()
		clock.Sleep(20 * time.Millisecond)
		return NewValueResult(2, nil)
	})
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		running.Done()
		clock.Sleep(30 * time.Millisecond)
		return NewValueResult(3, nil)
	})

	// Wait for goroutines to start
	running.Wait()

	go clock.BlockingAdvance(10 * time.Millisecond)

	res, err := col.Wait(10 * time.Millisecond)
	Expect(res).To(BeNil())
	Expect(err).ToNot(BeNil())
	Expect(err).To(Equal(ErrTimeout))
}

func (s *AsyncColSuite) TestWaitCollect(t *testing.T) {
	clock := glock.NewMockClock()

	col := NewAsyncCollector(WithClock(clock))

	var running sync.WaitGroup
	running.Add(3)
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		running.Done()
		clock.Sleep(5 * time.Millisecond)
		return NewValueResult(1, nil)
	})
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		running.Done()
		clock.Sleep(1 * time.Millisecond)
		return NewValueResult(2, nil)
	})
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		running.Done()
		return NewValueResult(3, nil)
	})

	// Wait until all the goroutines have started
	running.Wait()

	// Advance time so blocking tasks can finish
	go clock.BlockingAdvance(5 * time.Millisecond)

	res, err := col.Wait(10 * time.Millisecond)
	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res[0]).To(Equal(&ValueResult{Value: 1, Error: nil}))
	Expect(res[1]).To(Equal(&ValueResult{Value: 2, Error: nil}))
	Expect(res[2]).To(Equal(&ValueResult{Value: 3, Error: nil}))
}

func (s *AsyncColSuite) TestWaitCollectMultipleWaits(t *testing.T) {
	clock := glock.NewMockClock()

	col := NewAsyncCollector(WithClock(clock))

	var running sync.WaitGroup
	running.Add(3)
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		running.Done()
		clock.Sleep(5 * time.Millisecond)
		return NewValueResult(1, nil)
	})
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		running.Done()
		clock.Sleep(1 * time.Millisecond)
		return NewValueResult(2, nil)
	})
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		running.Done()
		return NewValueResult(3, nil)
	})

	running.Wait()

	go clock.Advance(5 * time.Millisecond)

	res, err := col.Wait(10 * time.Millisecond)
	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())

	res, err = col.Wait(10 * time.Millisecond)
	Expect(err).ToNot(BeNil())
	Expect(err).To(Equal(ErrFinished))
	Expect(res).To(BeNil())
}

func (s *AsyncColSuite) TestWaitCloserTimeout(t *testing.T) {
	clock := glock.NewMockClock()

	col := NewAsyncCollector(WithClock(clock))

	var running sync.WaitGroup
	running.Add(3)
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		running.Done()
		clock.Sleep(15 * time.Millisecond)
		return NewValueResult(1, nil)
	})
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		running.Done()
		clock.Sleep(10 * time.Millisecond)
		return NewValueResult(2, nil)
	})
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		running.Done()
		clock.Sleep(5 * time.Millisecond)
		return NewValueResult(3, nil)
	})

	running.Wait()

	var closed sync.WaitGroup
	closed.Add(3)
	go clock.BlockingAdvance(20 * time.Millisecond)
	res, err := col.WaitCloser(1*time.Millisecond, func(res TaskResult) {
		closed.Done()
	})
	Expect(res).To(BeNil())
	Expect(err).ToNot(BeNil())
	Expect(err).To(Equal(ErrTimeout))

	// If this returns we know it's successful because all 3 tasks called
	// done on the wait group
	closed.Wait()
}

func (s *AsyncColSuite) TestWaitCloserCollect(t *testing.T) {
	col := NewAsyncCollector()

	col.Run(func(task *Task, data ...interface{}) TaskResult {
		time.Sleep(5 * time.Millisecond)
		return NewValueResult(1, nil)
	})
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		time.Sleep(1 * time.Millisecond)
		return NewValueResult(2, nil)
	})
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		return NewValueResult(3, nil)
	})

	res, err := col.WaitCloser(10*time.Millisecond, func(res TaskResult) {})
	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res[0]).To(Equal(&ValueResult{Value: 1, Error: nil}))
	Expect(res[1]).To(Equal(&ValueResult{Value: 2, Error: nil}))
	Expect(res[2]).To(Equal(&ValueResult{Value: 3, Error: nil}))
}

func (s *AsyncColSuite) TestWaitCloserCollectNoTimeout(t *testing.T) {
	col := NewAsyncCollector()

	col.Run(func(task *Task, data ...interface{}) TaskResult {
		time.Sleep(5 * time.Millisecond)
		return NewValueResult(1, nil)
	})
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		time.Sleep(1 * time.Millisecond)
		return NewValueResult(2, nil)
	})
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		return NewValueResult(3, nil)
	})

	res, err := col.WaitCloser(0, func(res TaskResult) {})
	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res[0]).To(Equal(&ValueResult{Value: 1, Error: nil}))
	Expect(res[1]).To(Equal(&ValueResult{Value: 2, Error: nil}))
	Expect(res[2]).To(Equal(&ValueResult{Value: 3, Error: nil}))
}

func (s *AsyncColSuite) TestArgs(t *testing.T) {
	col := NewAsyncCollector()

	col.Run(func(task *Task, data ...interface{}) TaskResult {
		return NewValueResult(fmt.Sprintf("r %d %d", data[0], data[1]), nil)
	}, 1, 2)
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		return NewValueResult(fmt.Sprintf("r %d %d", data[0], data[1]), nil)
	}, 3, 4)
	col.Run(func(task *Task, data ...interface{}) TaskResult {
		return NewValueResult(fmt.Sprintf("r %d %d", data[0], data[1]), nil)
	}, 5, 6)

	res, err := col.Wait(10 * time.Millisecond)
	Expect(err).To(BeNil())
	Expect(res).ToNot(BeNil())
	Expect(res[0]).To(Equal(&ValueResult{Value: "r 1 2", Error: nil}))
	Expect(res[1]).To(Equal(&ValueResult{Value: "r 3 4", Error: nil}))
	Expect(res[2]).To(Equal(&ValueResult{Value: "r 5 6", Error: nil}))
}
