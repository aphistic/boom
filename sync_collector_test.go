package boom

import (
	"fmt"
	"time"

	. "gopkg.in/check.v1"
)

type SyncColSuite struct{}

var _ = Suite(&SyncColSuite{})

func (s *SyncColSuite) TestWaitTimeout(c *C) {
	col := NewSyncCollector()

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

func (s *SyncColSuite) TestWaitCollect(c *C) {
	col := NewSyncCollector()

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

func (s *SyncColSuite) TestWaitCollectMultipleWaits(c *C) {
	col := NewSyncCollector()

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

func (s *SyncColSuite) TestWaitCloserTimeout(c *C) {
	col := NewSyncCollector()

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

func (s *SyncColSuite) TestWaitCloserTimeoutWithResults(c *C) {
	col := NewSyncCollector()

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

	c.Check(<-closeChan, Equals, 2)
	c.Check(<-closeChan, Equals, 3)
	c.Check(<-closeChan, Equals, 1)
}

func (s *SyncColSuite) TestWaitCloserCollect(c *C) {
	col := NewSyncCollector()

	finishChan := make(chan int, 3)
	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(25 * time.Millisecond)
		finishChan <- 1
		return NewValueResult(1, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(10 * time.Millisecond)
		finishChan <- 2
		return NewValueResult(2, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		finishChan <- 3
		return NewValueResult(3, nil)
	})

	res, err := col.WaitCloser(50*time.Millisecond, func(res TaskResult) {})
	c.Check(err, IsNil)
	c.Check(res, NotNil)
	c.Check(res[0], DeepEquals, &ValueResult{Value: 1, Error: nil})
	c.Check(res[1], DeepEquals, &ValueResult{Value: 2, Error: nil})
	c.Check(res[2], DeepEquals, &ValueResult{Value: 3, Error: nil})
	c.Check(<-finishChan, Equals, 1)
	c.Check(<-finishChan, Equals, 2)
	c.Check(<-finishChan, Equals, 3)
}

func (s *SyncColSuite) TestWaitCloserCollectNoTimeout(c *C) {
	col := NewSyncCollector()

	finishChan := make(chan int, 3)
	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(10 * time.Millisecond)
		finishChan <- 1
		return NewValueResult(1, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		time.Sleep(30 * time.Millisecond)
		finishChan <- 2
		return NewValueResult(2, nil)
	})
	col.Run(func(data ...interface{}) TaskResult {
		finishChan <- 3
		return NewValueResult(3, nil)
	})

	res, err := col.WaitCloser(0, func(res TaskResult) {})
	c.Check(err, IsNil)
	c.Check(res, NotNil)
	c.Check(res[0], DeepEquals, &ValueResult{Value: 1, Error: nil})
	c.Check(res[1], DeepEquals, &ValueResult{Value: 2, Error: nil})
	c.Check(res[2], DeepEquals, &ValueResult{Value: 3, Error: nil})
	c.Check(<-finishChan, Equals, 1)
	c.Check(<-finishChan, Equals, 2)
	c.Check(<-finishChan, Equals, 3)
}

func (s *SyncColSuite) TestArgs(c *C) {
	col := NewSyncCollector()

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
