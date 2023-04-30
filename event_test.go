package common

import (
	"testing"
	"time"
)

func init() {
}

func TestEvent(t *testing.T) {
	e := NewEvent()
	finished1 := false
	finished2 := false

	fun := func(v *bool) {
		e.Wait()
		*v = true
	}

	go fun(&finished1)
	go fun(&finished2)
	time.Sleep(1 * time.Second)
	if finished1 {
		t.Logf(`finished1!`)
		t.Fail()
	}
	if finished2 {
		t.Logf(`finished2!`)
		t.Fail()
	}

	e.Set()
	time.Sleep(10 * time.Millisecond)
	if !finished1 {
		t.Logf(`finished1 not signaled!`)
		t.Fail()
	}
	if !finished2 {
		t.Logf(`finished2 not signaled!`)
		t.Fail()
	}

	finished1 = false
	finished2 = false
	e.Unset()

	go fun(&finished1)
	go fun(&finished2)
	time.Sleep(1 * time.Second)
	if finished1 {
		t.Logf(`finished1!`)
		t.Fail()
	}
	if finished2 {
		t.Logf(`finished2!`)
		t.Fail()
	}
}
