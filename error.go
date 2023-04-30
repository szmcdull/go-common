package common

import (
	"errors"
	"fmt"
	"runtime"
	"time"
)

// Wrapper 与errors标准库一致
type Wrapper interface {
	Unwrap() error
}

// BaseError BaseError
type BaseError struct {
	Msg   string
	Err   error
	Stack string
}

// Unwrap 实现errors标准库接口
func (me *BaseError) Unwrap() error {
	return me.Err
}

func callers() *[]uintptr {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	var st []uintptr = pcs[0:n]
	return &st
}

// NewError 将 err 包装成子错误，并记录 stacktrace, err为空表示这是一个根错误。
// 如果 err 已经是 BaseError，不会重复记录 stacktrace。
// 对于标准库和第三方库返回的 error，都要通过 NewError 封装后再返回给上层调用，以便记录日志的时候有调用栈信息
func NewError(msg string, err interface{}) BaseError {
	var err3 error

	switch err2 := err.(type) {
	case error:
		var base BaseError
		if err2 != nil && errors.As(err2, &base) { // 如果err已经是BaseError，则只添加msg
			return BaseError{msg, err2, ``}
		} else { // 如果是未知的err，则还记录stacktrace
			err3 = err2
		}
	default:
		err3 = fmt.Errorf(`%v`, err2)
	}

	buf := make([]byte, 2048)
	len := runtime.Stack(buf, false)
	buf = buf[:len]
	return BaseError{msg, err3, string(buf)}
}

func NewErrorf(err interface{}, template string, args ...interface{}) BaseError {
	msg := fmt.Sprintf(template, args...)
	return NewError(msg, err)
}

func WrapError(err error, template string, args ...interface{}) BaseError {
	if e, ok := err.(BaseError); ok {
		e.AddMsg(template, args...)
		return e
	}
	return NewErrorf(err, template, args...)
}

func (me BaseError) Error() string {
	err := ``
	if me.Err != nil {
		err = me.Err.Error()
	}
	msg := me.Msg
	if msg != `` {
		msg += `: `
	}
	msg += err
	if me.Stack == `` {
		return msg
	} else {
		return fmt.Sprintf("%s\n%s", msg, me.Stack)
	}
}

func (me *BaseError) AddMsg(template string, args ...interface{}) {
	if len(args) == 0 {
		me.Msg = template + me.Msg
	} else {
		me.Msg = fmt.Sprintf(template+`: `, args...) + me.Msg
	}
}

var (
	NotImplementedError = errors.New(`method not implemented`)
	ProgramExitingError = errors.New(`program exiting`)
)

func PanicWithTime(v interface{}) {
	panic(fmt.Errorf(`%s - %v`, time.Now().Format(`2006-01-02 15:04:05`), v))
}
