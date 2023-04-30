package common

import (
	"context"
	"fmt"
	"math"
	"os"
	"testing"
	"time"
)

func _TestExit(t *testing.T) {
	go func() {
		time.Sleep(3 * time.Second)
		p, _ := os.FindProcess(os.Getpid())
		p.Signal(os.Interrupt)
	}()
	now := time.Now()
	<-ProgramDone()
	now2 := time.Now()
	seconds := now2.Sub(now).Seconds()
	t.Logf(`TestExit seconds = %f`, seconds)
}

func TestWaitAll(t *testing.T) {
	t1 := NewTask(func() error {
		time.Sleep(3 * time.Second)
		return nil
	})
	t2 := NewTask(func() error {
		time.Sleep(5 * time.Second)
		return nil
	})
	now := time.Now()
	WaitAllTasks(t1, t2)
	now2 := time.Now()
	seconds := now2.Sub(now).Seconds()
	if seconds < 5 || seconds >= 6 {
		t.Errorf(`WaitAll used %v seconds, expected [5,6)`, seconds)
	}

	println(`Checking if WaitAllTasks([]...) dead locks...`)
	WaitAllTasks(nil...)
}

func TestWaitAny(t *testing.T) {
	t1 := NewTask(func() error {
		time.Sleep(3 * time.Second)
		return nil
	})
	t2 := NewTask(func() error {
		time.Sleep(5 * time.Second)
		return nil
	})
	now := time.Now()
	WaitAnyTask(t1, t2)
	now2 := time.Now()
	seconds := now2.Sub(now).Seconds()
	fmt.Printf("TestWaitAny seconds = %f\n", seconds)
}

func TestResult(t *testing.T) {
	t1 := NewTaskWithResult(func() (interface{}, error) {
		return 123, nil
	})
	<-t1.Done()
	result, err := t1.GetResult()
	fmt.Printf("TestResult result = %v, err = %v\n", result, err)

	t2 := NewTaskWithResult(func() (interface{}, error) {
		time.Sleep(5 * time.Second)
		return 456, fmt.Errorf(`test error`)
	})
	now := time.Now()
	<-t2.Done()
	now2 := time.Now()
	result, err = t2.GetResult()
	seconds := now2.Sub(now).Seconds()
	fmt.Printf("TestResult seconds = %f, result = %v, err = %v\n", seconds, result, err)
}

func TestResultT(t *testing.T) {
	t1 := NewTaskWithResultT(func() (int, error) {
		return 123, nil
	})
	<-t1.Done()
	result, err := t1.GetResult()
	fmt.Printf("TestResult result = %v, err = %v\n", result, err)

	t2 := NewTaskWithResultT(func() (int, error) {
		time.Sleep(5 * time.Second)
		return 456, fmt.Errorf(`test error`)
	})
	now := time.Now()
	<-t2.Done()
	now2 := time.Now()
	result, err = t2.GetResult()
	seconds := now2.Sub(now).Seconds()
	fmt.Printf("TestResult seconds = %f, result = %v, err = %v\n", seconds, result, err)
}

func concurrentTestSet(id int, c context.Context, v *WaitableValue) {
	for i := 0; ; i++ {
		//fmt.Printf("%d: set\n", id)
		v.Set(id)
		//time.Sleep(time.Nanosecond)
		//fmt.Printf("%d %d: check done\n", id, i)
		if c.Err() != nil {
			//fmt.Printf("%d %d: done\n", id, i)
			return
		}
		//fmt.Printf("%d %d: next\n", id, i)
	}
}

func TestConcurrent(t *testing.T) {
	var account int
	accountEvent := NewWaitableValue()
	var updateTask *Task
	cancel := NewCancelCtx(context.Background())

	waitCount := 0
	NewUpdateTask := func() {
		updateTask = NewTask(
			func() error {
				waitCount++
				fmt.Printf("updateTask %d: WaitAndReset\n", waitCount)
				account = accountEvent.WaitAndReset().(int)
				fmt.Printf("updateTask %d: awaited %d\n", waitCount, account)
				return nil
			})
	}
	NewUpdateTask()

	go concurrentTestSet(1, cancel, accountEvent)
	go concurrentTestSet(2, cancel, accountEvent)
	go concurrentTestSet(3, cancel, accountEvent)
	ticker := time.NewTicker(time.Second)

	count := 100000
	success := 0
	run := true
	last := 0
	checks := 0
	for run {
		select {
		case <-updateTask.Done():
			_ = account
			success++
			if success >= count {
				run = false
				break
			}
			NewUpdateTask()
		case <-ticker.C:
			now := success
			checks++
			if now == last {
				cancel.Cancel()
				t.Errorf(`deadlock after %d waits, %d checks`, success, checks)
				run = false
				break
				//time.Now()
			}
			last = now
		}
	}

	cancel.Cancel()

	if success != count {
		t.Fail()
	}
}

func TestSetIntervalFuncLong(t *testing.T) {
	count := 0
	start := time.Now()
	context := NewCancelCtx(context.Background())
	SetIntervalFunc(func() time.Duration { return 500 * time.Millisecond },
		func() {
			fmt.Printf("%v\n", time.Now())
			time.Sleep(time.Millisecond * 1000)
			count++
			if count >= 10 {
				context.Cancel()
			}
		}).WithContext(context, nil).Run()
	<-context.Done()

	duration := time.Since(start)
	if count != 10 {
		t.Fail()
	}
	if duration < time.Millisecond*10500 || duration > time.Second*11 {
		t.Fail()
	}
}

func TestSetIntervalFuncShort(t *testing.T) {
	count := 0
	start := time.Now()
	context := NewCancelCtx(context.Background())
	SetIntervalFunc(func() time.Duration { return 1000 * time.Millisecond },
		func() {
			fmt.Printf("%v\n", time.Now())
			time.Sleep(time.Millisecond * 500)
			count++
			if count >= 10 {
				context.Cancel()
			}
		}).WithContext(context, nil).Run()
	<-context.Done()

	duration := time.Since(start)
	if count != 10 {
		t.Fail()
	}
	if duration < time.Millisecond*10500 || duration > time.Second*11 {
		t.Fail()
	}
}

func TestSetIntervalFuncVar(t *testing.T) {
	i := 0
	times := []time.Time{time.Now()}
	cancel := NewCancelCtx(context.Background())
	SetIntervalFunc(func() time.Duration {
		i++
		return time.Duration(i) * time.Second
	}, func() {
		times = append(times, time.Now())
		if i > 3 {
			cancel.Cancel()
		}
	}).WithContext(cancel, nil).RunInCurrentGoProc()

	if math.Round(times[1].Sub(times[0]).Seconds()) != 1 {
		t.Error(1, math.Round(times[1].Sub(times[0]).Seconds()))
	}
	if math.Round(times[2].Sub(times[1]).Seconds()) != 2 {
		t.Error(2, math.Round(times[2].Sub(times[1]).Seconds()))
	}
	if math.Round(times[3].Sub(times[2]).Seconds()) != 3 {
		t.Error(3, math.Round(times[3].Sub(times[2]).Seconds()))
	}
}

func TestTaskCompletionSource(t *testing.T) {
	c := NewTaskCompletionSource()
	c.SetResult(123)
	c.Wait()
	r, err := c.GetResult()
	if err != nil {
		t.Fail()
	}
	if r.(int) != 123 {
		t.Fail()
	}
}

func TestTaskCompletionSourceT(t *testing.T) {
	c := NewTaskCompletionSourceT[int]()
	c.SetResult(123)
	c.Wait()
	r, err := c.GetResult()
	if err != nil {
		t.Fail()
	}
	if r != 123 {
		t.Fail()
	}
}
