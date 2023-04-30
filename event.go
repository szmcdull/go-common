package common

import (
	"context"
)

// 广播事件，当事件发生(Set)的时候，所有等待者都会收到通知
type Event struct {
	cancel CancelCtx
}

func NewEvent() *Event {
	return &Event{
		cancel: *NewCancelCtx(context.Background()),
	}
}

// 让事件发生，返回false表示事件已经发生过
func (me *Event) Set() bool {
	return me.cancel.Cancel()
}

// 事件是否正在发生
func (me *Event) IsSet() bool {
	return me.cancel.isDone != 0
}

// 等待事件发生
func (me *Event) Wait() {
	<-me.cancel.Done()
}

func (me *Event) Done() <-chan struct{} {
	return me.cancel.Done()
}

// 停止事件发生
func (me *Event) Unset() {
	me.cancel = *NewCancelCtx(context.Background())
}

// 等待事件发生，然后停止事件，只能在只有一个等待者的时候使用，否则就可能是用错了
func (me *Event) WaitAndReset() {
	me.Wait()
	me.Unset()
}

// 可等待变化的值
type WaitableValue struct {
	e *Event
	v interface{}
}

func NewWaitableValue() *WaitableValue {
	return &WaitableValue{
		e: NewEvent(),
	}
}

// 设置一个新的值，并通知所有等待者
func (me *WaitableValue) Set(v interface{}) {
	me.v = v
	me.e.Set()
}

// 等待值变化，并返回新的值
func (me *WaitableValue) Wait() interface{} {
	me.e.Wait()
	return me.v
}

// 等待值变化，并返回新的值，只能在只有一个等待者的时候使用，否则就可能是用错了
func (me *WaitableValue) WaitAndReset() interface{} {
	me.e.WaitAndReset()
	return me.v
}

func (me *WaitableValue) Updated() bool {
	return me.e.cancel.isDone != 0
}

type WaitableValueG[T any] struct {
	e *Event
	v T
}

func NewWaitableValue2[T any]() *WaitableValueG[T] {
	return &WaitableValueG[T]{
		e: NewEvent(),
	}
}

// 设置一个新的值，并通知所有等待者
func (me *WaitableValueG[T]) Set(v T) {
	me.v = v
	me.e.Set()
}

// 等待值变化，并返回新的值
func (me *WaitableValueG[T]) Wait() T {
	me.e.Wait()
	return me.v
}

// 等待值变化，并返回新的值，只能在只有一个等待者的时候使用，否则就可能是用错了
func (me *WaitableValueG[T]) WaitAndReset() T {
	me.e.WaitAndReset()
	return me.v
}

func (me *WaitableValueG[T]) Updated() bool {
	return me.e.cancel.isDone != 0
}
