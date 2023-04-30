package common

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

/*
 *	处理程序退出
 */

var (
	//exitSignal     chan struct{} = make(chan struct{})
	programDone      bool
	programExitEvent *Event = NewEvent()
	once             sync.Once
)

func InitExitHandler() {
	once.Do(func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
		go func() {
			for {
				sig := <-c
				if sig != syscall.SIGHUP {
					programDone = true
					//close(exitSignal) // close会让所有等待channel的协程返回
					programExitEvent.Set()
					return
				}
			}
		}()
	})
}

// 等待直到收到程序退出信号
func ProgramDone() <-chan struct{} {
	return programExitEvent.Done()
}

func SetProgramDone() {
	programExitEvent.Set()
	programDone = true
}

func IsProgramDone() bool {
	return programDone
}

/*
 *	定时器
 */

// TimerCallback 回调函数定义
type TimerCallback func()

type GetIntervalFunc func() time.Duration

type SetIntervalTask struct {
	interval    time.Duration
	callback    TimerCallback
	ctx         context.Context
	cleanupFunc func()

	skipFirstInterval  bool
	runInCurrentGoProc bool
}

type SetIntervalFuncTask struct {
	intervalFunc GetIntervalFunc
	callback     TimerCallback
	ctx          context.Context
	cleanupFunc  func()

	skipFirstInterval  bool
	runInCurrentGoProc bool
}

// Run/RunInCurrentGoProc的时候，先马上*在当前go proc*执行一次，然后定时执行。
func (me *SetIntervalTask) SkipFirstInterval() *SetIntervalTask {
	me.skipFirstInterval = true
	return me
}

func (me *SetIntervalFuncTask) SkipFirstInterval() *SetIntervalFuncTask {
	me.skipFirstInterval = true
	return me
}

func (me *SetIntervalTask) RunInCurrentGoProc() {
	me.runInCurrentGoProc = true
	me.Run()
}

func (me *SetIntervalTask) WithContext(ctx context.Context, cleanupFunc func()) *SetIntervalTask {
	me.ctx = ctx
	me.cleanupFunc = cleanupFunc
	return me
}

func (me *SetIntervalFuncTask) WithContext(ctx context.Context, cleanupFunc func()) *SetIntervalFuncTask {
	me.ctx = ctx
	me.cleanupFunc = cleanupFunc
	return me
}

func (me *SetIntervalTask) Run() {
	if me.skipFirstInterval {
		me.callback()
	}

	fun := func() {
		timer := time.NewTicker(me.interval)
		defer func() {
			timer.Stop()
			if me.cleanupFunc != nil {
				me.cleanupFunc()
			}
		}()

		for {
			select {
			case <-timer.C:
				me.callback()
			case <-ProgramDone():
				return
			case <-me.ctx.Done():
				return
			}
		}
	}

	if !me.runInCurrentGoProc {
		go fun()
	} else {
		fun()
	}
}

func (me *SetIntervalFuncTask) Run() {
	if me.skipFirstInterval {
		me.callback()
	}

	closedChan := make(chan time.Time, 1)
	close(closedChan)

	fun := func() {
		var interval time.Duration
		var timer *time.Timer
		var c <-chan time.Time
		end := time.Now().Add(me.intervalFunc())

		startCounter := func() {
			interval = me.intervalFunc()
			now := time.Now()
			if now.Before(end) { // 剩余时间还可以休眠
				end0 := end
				end = end.Add(interval)
				timer = time.NewTimer(time.Until(end0))
				c = timer.C
			} else {
				end = now.Add(interval)
				c = closedChan
			}
		}
		startCounter()

		defer func() {
			if timer != nil {
				timer.Stop()
			}
			if me.cleanupFunc != nil {
				me.cleanupFunc()
			}
		}()
		for {
			select {
			case <-c:
				me.callback()
				startCounter()
			case <-me.ctx.Done():
				return
			case <-ProgramDone():
				return
			}
		}
	}

	if !me.runInCurrentGoProc {
		go fun()
	} else {
		fun()
	}
}

func (me *SetIntervalFuncTask) RunInCurrentGoProc() {
	me.runInCurrentGoProc = true
	me.Run()
}

// 定时器，如果程序收到退出信号，定时器会自动取消
func SetInterval(interval time.Duration, callback TimerCallback) *SetIntervalTask {
	return &SetIntervalTask{
		interval: interval,
		callback: callback,
		ctx:      context.Background(),
	}
}

func SetIntervalMS(intervalMs int64, callback TimerCallback) *SetIntervalTask {
	return &SetIntervalTask{
		interval: time.Duration(intervalMs) * time.Millisecond,
		callback: callback,
		ctx:      context.Background(),
	}
}

func SetIntervalFunc(intervalFunc GetIntervalFunc, callback TimerCallback) *SetIntervalFuncTask {
	return &SetIntervalFuncTask{
		intervalFunc: intervalFunc,
		callback:     callback,
		ctx:          context.Background(),
	}
}

func SetTimeoutMS(intervalMs int64, callback TimerCallback) {
	if err := SleepMS(intervalMs); err == nil {
		callback()
	}
}

func SetTimeout(duration time.Duration, callback TimerCallback) {
	if err := Sleep(duration); err == nil {
		callback()
	}
}

// func SetIntervalRandomMS(intervalMinMs, intervalMaxMs int64, callback TimerCallback) {
// 	Range := [2]time.Duration{time.Duration(intervalMinMs) * time.Millisecond, time.Duration(intervalMaxMs) * time.Millisecond}
// 	SetIntervalRandom(Range, callback)
// }

// func SetIntervalRandom(_range [2]time.Duration, callback TimerCallback) {
// 	timer := helper.NextTimer(_range)
// 	for {
// 		select {
// 		case <-timer.C:
// 			timer = helper.NextTimer(_range)
// 			callback()
// 		case <-ProgramDone():
// 			timer.Stop()
// 			return
// 		}
// 	}
// }

// func SetIntervalRandomFunc(intervalFunc func() [2]time.Duration, callback TimerCallback) {
// 	timer := helper.NextTimer(intervalFunc())
// 	for {
// 		select {
// 		case <-timer.C:
// 			timer = helper.NextTimer(intervalFunc())
// 			callback()
// 		case <-ProgramDone():
// 			timer.Stop()
// 			return
// 		}
// 	}
// }

// 等待，如果程序收到退出信号，则马上返回err
func SleepMS(intervalMs int64) error {
	timer := time.NewTimer(time.Duration(intervalMs) * time.Millisecond)
	select {
	case <-timer.C:
		return nil
	case <-ProgramDone():
		return ProgramExitingError
	}
}

func Sleep(duration time.Duration) error {
	timer := time.NewTimer(duration)
	select {
	case <-timer.C:
		return nil
	case <-ProgramDone():
		return ProgramExitingError
	}
}

/*
 *	任务处理
 */

type (
	TaskT[T any] struct {
		result      T
		err         error
		isDone      atomic.Bool
		completion  *Event
		cancelEvent *Event
	}

	// deprecated 已过时，新项目请使用 TaskT[T]，Task 后续将会移除
	Task = TaskT[any]
)

func (me *TaskT[T]) Done() <-chan struct{} {
	return me.completion.Done()
}

func (me *TaskT[T]) Wait() {
	<-me.Done()
}

func (me *TaskT[T]) WaitWithContext(ctx context.Context) error {
	select {
	case <-me.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (me *TaskT[T]) WaitTimeout(timeout time.Duration) error {
	timeoutContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case <-me.Done():
		return nil
	case <-timeoutContext.Done():
		return timeoutContext.Err()
	}
}

func (me *TaskT[T]) IsDone() bool {
	return me.completion.IsSet()
}

func (me *TaskT[T]) GetResult() (T, error) {
	<-me.Done()
	return me.result, me.err
}

func (me *TaskT[T]) done() {
	me.completion.Set()
}

func (me *TaskT[T]) Cancel() error {
	if me.cancelEvent != nil {
		me.cancelEvent.Set()
		return nil
	} else {
		return NewError(`This task is not cancellable`, nil)
	}
}

func NewTaskWithResultT[T any](fun func() (T, error)) *TaskT[T] {
	result := &TaskT[T]{
		completion: NewEvent(),
	}
	go func() {
		defer func() {
			if err := recover(); err != nil {
				var ok bool
				if result.err, ok = err.(error); !ok {
					result.err = fmt.Errorf(`%v`, err)
				}
			}
			result.done()
		}()
		result.result, result.err = fun()
	}()
	return result
}

func NewTaskWithResult(fun func() (any, error)) *Task {
	return NewTaskWithResultT(fun)
}

func NewTask(fun func() error) *Task {
	result := &Task{
		completion: NewEvent(),
	}
	go func() {
		defer func() {
			if err := recover(); err != nil {
				result.err = NewErrorf(err, `NewTask.loop`)
			}
			result.done()
		}()
		result.err = fun()
	}()
	return result
}

func NewCancellableTask(fun func(cancelEvent *Event) error) *Task {
	result := &Task{
		completion:  NewEvent(),
		cancelEvent: NewEvent(),
	}
	go func() {
		defer func() {
			if err := recover(); err != nil {
				var ok bool
				if result.err, ok = err.(error); !ok {
					result.err = fmt.Errorf(`%v`, err)
				}
			}
			result.done()
		}()
		result.err = fun(result.cancelEvent)
	}()
	return result
}

type TaskCompletionSource = TaskCompletionSourceT[any]
type TaskCompletionSourceT[T any] struct {
	TaskT[T]
}

func NewTaskCompletionSource() *TaskCompletionSource {
	return NewTaskCompletionSourceT[any]()
}

func NewTaskCompletionSourceT[T any]() *TaskCompletionSourceT[T] {
	return &TaskCompletionSourceT[T]{TaskT[T]{
		completion:  NewEvent(),
		cancelEvent: NewEvent(),
	}}
}

func (me *TaskCompletionSourceT[T]) SetResult(r T) bool {
	if me.isDone.CompareAndSwap(false, true) {
		me.result = r
		me.completion.Set()
		return true
	}
	return false
}

func (me *TaskCompletionSourceT[T]) SetError(e error) bool {
	if me.isDone.CompareAndSwap(false, true) {
		me.err = e
		me.completion.Set()
		return true
	}
	return false
}

// 等待任一任务完成，如果收到程序退出信号，则马上返回且error不为nil
func WaitAnyTask(tasks ...*Task) (*Task, error) {
	if len(tasks) == 0 {
		return nil, nil
	}

	set := []reflect.SelectCase{}
	for _, ch := range tasks {
		set = append(set, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch.Done()),
		})
	}
	set = append(set, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(ProgramDone()),
	})
	from, _, _ := reflect.Select(set)
	if from == len(set)-1 {
		return nil, ProgramExitingError
	}
	return tasks[from], nil
}

// 等待所有任务完成，如果收到程序退出信号，则马上返回且error不为nil
func WaitAllTasks(tasks ...*Task) error {
	if len(tasks) == 0 {
		return nil
	}

	set := []reflect.SelectCase{}
	for _, ch := range tasks {
		set = append(set, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch.Done()),
		})
	}
	set = append(set, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(ProgramDone()),
	})
	for {
		from, _, _ := reflect.Select(set)
		if from == len(set)-1 {
			return ProgramExitingError
		}
		set = append(set[0:from], set[from+1:]...)
		if len(set) == 1 {
			return nil
		}
	}
}
