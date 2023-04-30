package common

import (
	"reflect"
	"time"
)

// 同时读取多个chan
// timeout: 指定超时时间，<=0表示永不超时
// which: 	第几个chan读取到了数据
// value:	读取到的数据
// ok: 		true=读取成功，false=chan已关闭
func SelectChans(timeout time.Duration, chans ...interface{}) (which int, value interface{}, ok bool) {
	if len(chans) == 0 {
		return 1, nil, false
	}

	length := len(chans)
	if timeout > 0 {
		length++
	}

	set := make([]reflect.SelectCase, length)
	for i := range chans {
		set[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(chans[i]),
		}
	}

	if timeout > 0 {
		set[length] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(time.NewTimer(timeout).C),
		}
	}

	which, value, ok = reflect.Select(set)
	return
}

// 等待所有chan发出信号(写入数据或close)，一般用于chan struct{}
// timeout: 指定超时时间，必须>0
func WaitAllChans(timeout time.Duration, chans ...interface{}) bool {
	if len(chans) == 0 {
		return true
	}

	length := len(chans)
	length++

	set := make([]reflect.SelectCase, length)
	for i := range chans {
		set[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(chans[i]),
		}
	}

	set[len(chans)] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(time.NewTimer(timeout).C),
	}

	for len(set) > 1 {
		which, _, _ := reflect.Select(set)
		if which == len(set)-1 { // 超时
			return false
		}
		set = append(set[0:which], set[which+1:]...)
	}

	return true
}
