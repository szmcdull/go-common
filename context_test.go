package common

import (
	"context"
	"testing"
)

func TestWithCancel(t *testing.T) {
	ctx := NewCancelCtx(context.Background())
	ctx.Cancel()

	done := false
	select {
	case <-ctx.Done():
		done = true
	default:
	}

	if !done {
		t.Log(`Should be done`)
		t.Fail()
	}
}

func TestNotCanceled(t *testing.T) {
	ctx := NewCancelCtx(context.Background())

	done := false
	select {
	case <-ctx.Done():
		done = true
	default:
	}

	if done {
		t.Log(`Should not be done`)
		t.Fail()
	}
}

func TestLinkedCancelCtx1(t *testing.T) {
	ctx1 := NewCancelCtx(context.Background())
	ctx2 := NewCancelCtx(context.Background())

	ctx := ctx1.NewLinkedCancelCtx(ctx2)
	ctx1.Cancel()

	done := false
	select {
	case <-ctx.Done():
		done = true
	default:
	}

	if !done {
		t.Log(`Should be done`)
		t.Fail()
	}
}

func TestLinkedCancelCtx2(t *testing.T) {
	ctx1 := NewCancelCtx(context.Background())
	ctx2 := NewCancelCtx(context.Background())

	ctx := ctx1.NewLinkedCancelCtx(ctx2)
	ctx2.Cancel()

	done := false
	select {
	case <-ctx.Done():
		done = true
	default:
	}

	if !done {
		t.Log(`Should be done`)
		t.Fail()
	}
}

func TestErr(t *testing.T) {
	c, cancel := context.WithCancel(context.Background())
	c2 := NewCancelCtx(c)
	cancel()
	if c2.Err() == nil {
		t.Errorf(`err is nil`)
	}
}
