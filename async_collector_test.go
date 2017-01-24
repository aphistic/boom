package boom

import (
	"fmt"
	"time"

	. "gopkg.in/check.v1"
)

type AsyncColSuite struct{}

var _ = Suite(&AsyncColSuite{})

func (s *AsyncColSuite) TestWaitTimeout(c *C) {
	col := NewAsyncCollector()

	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(15 * time.Millisecond)
		return NewValueResult(1, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(20 * time.Millisecond)
		return NewValueResult(2, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(30 * time.Millisecond)
		return NewValueResult(3, nil)
	})

	res, err := col.Wait(10 * time.Millisecond)
	c.Check(res, IsNil)
	c.Check(err, NotNil)
	c.Check(err, Equals, ErrTimeout)
}

func (s *AsyncColSuite) TestWaitCollect(c *C) {
	col := NewAsyncCollector()

	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(5 * time.Millisecond)
		return NewValueResult(1, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(1 * time.Millisecond)
		return NewValueResult(2, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		return NewValueResult(3, nil)
	})

	res, err := col.Wait(10 * time.Millisecond)
	c.Check(err, IsNil)
	c.Check(res, NotNil)
	c.Check(res[0], DeepEquals, &ValueResult{Value: 1, Error: nil})
	c.Check(res[1], DeepEquals, &ValueResult{Value: 2, Error: nil})
	c.Check(res[2], DeepEquals, &ValueResult{Value: 3, Error: nil})
}

func (s *AsyncColSuite) TestWaitCollectMultipleWaits(c *C) {
	col := NewAsyncCollector()

	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(5 * time.Millisecond)
		return NewValueResult(1, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(1 * time.Millisecond)
		return NewValueResult(2, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		return NewValueResult(3, nil)
	})

	res, err := col.Wait(10 * time.Millisecond)
	c.Check(err, IsNil)
	c.Check(res, NotNil)

	res, err = col.Wait(10 * time.Millisecond)
	c.Check(err, NotNil)
	c.Check(err, Equals, ErrFinished)
	c.Check(res, IsNil)
}

func (s *AsyncColSuite) TestWaitCloserTimeout(c *C) {
	col := NewAsyncCollector()

	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(15 * time.Millisecond)
		return NewValueResult(1, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(10 * time.Millisecond)
		return NewValueResult(2, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(5 * time.Millisecond)
		return NewValueResult(3, nil)
	})

	closeChan := make(chan int)
	res, err := col.WaitCloser(1*time.Millisecond, func(res TaskResult) {
		valResult := res.(*ValueResult)
		resNum := valResult.Value.(int)
		closeChan <- resNum
	})
	c.Check(res, IsNil)
	c.Check(err, NotNil)
	c.Check(err, Equals, ErrTimeout)

	num := <-closeChan
	c.Check(num, Equals, 3)
	num = <-closeChan
	c.Check(num, Equals, 2)
	num = <-closeChan
	c.Check(num, Equals, 1)
}

func (s *AsyncColSuite) TestWaitCloserTimeoutWithResults(c *C) {
	col := NewAsyncCollector()

	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(15 * time.Millisecond)
		return NewValueResult(1, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		return NewValueResult(2, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(5 * time.Millisecond)
		return NewValueResult(3, nil)
	})

	closeChan := make(chan int)
	res, err := col.WaitCloser(1*time.Millisecond, func(res TaskResult) {
		valResult := res.(*ValueResult)
		resNum := valResult.Value.(int)
		closeChan <- resNum
	})
	c.Check(res, IsNil)
	c.Check(err, NotNil)
	c.Check(err, Equals, ErrTimeout)

	num := <-closeChan
	c.Check(num, Equals, 2)
	num = <-closeChan
	c.Check(num, Equals, 3)
	num = <-closeChan
	c.Check(num, Equals, 1)
}

func (s *AsyncColSuite) TestWaitCloserCollect(c *C) {
	col := NewAsyncCollector()

	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(5 * time.Millisecond)
		return NewValueResult(1, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(1 * time.Millisecond)
		return NewValueResult(2, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		return NewValueResult(3, nil)
	})

	res, err := col.WaitCloser(10*time.Millisecond, func(res TaskResult) {})
	c.Check(err, IsNil)
	c.Check(res, NotNil)
	c.Check(res[0], DeepEquals, &ValueResult{Value: 1, Error: nil})
	c.Check(res[1], DeepEquals, &ValueResult{Value: 2, Error: nil})
	c.Check(res[2], DeepEquals, &ValueResult{Value: 3, Error: nil})
}

func (s *AsyncColSuite) TestWaitCloserCollectNoTimeout(c *C) {
	col := NewAsyncCollector()

	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(5 * time.Millisecond)
		return NewValueResult(1, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(1 * time.Millisecond)
		return NewValueResult(2, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		return NewValueResult(3, nil)
	})

	res, err := col.WaitCloser(0, func(res TaskResult) {})
	c.Check(err, IsNil)
	c.Check(res, NotNil)
	c.Check(res[0], DeepEquals, &ValueResult{Value: 1, Error: nil})
	c.Check(res[1], DeepEquals, &ValueResult{Value: 2, Error: nil})
	c.Check(res[2], DeepEquals, &ValueResult{Value: 3, Error: nil})
}

func (s *AsyncColSuite) TestArgs(c *C) {
	col := NewAsyncCollector()

	col.Run(func(data ...interface{}) TaskResult {
		return NewValueResult(fmt.Sprintf("r %d %d", data[0], data[1]), nil)
	}, 1, 2)
	col.Run(func(data ...interface{}) TaskResult {
		return NewValueResult(fmt.Sprintf("r %d %d", data[0], data[1]), nil)
	}, 3, 4)
	col.Run(func(data ...interface{}) TaskResult {
		return NewValueResult(fmt.Sprintf("r %d %d", data[0], data[1]), nil)
	}, 5, 6)

	res, err := col.Wait(10 * time.Millisecond)
	c.Check(err, IsNil)
	c.Check(res, NotNil)
	c.Check(res[0], DeepEquals, &ValueResult{Value: "r 1 2", Error: nil})
	c.Check(res[1], DeepEquals, &ValueResult{Value: "r 3 4", Error: nil})
	c.Check(res[2], DeepEquals, &ValueResult{Value: "r 5 6", Error: nil})
}
