package reconcile

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDebouncerFiresOnceAfterQuietPeriod(t *testing.T) {
	var count atomic.Int32
	done := make(chan struct{}, 1)

	d := NewDebouncer(50*time.Millisecond, func() {
		count.Add(1)
		done <- struct{}{}
	})

	d.Trigger()
	d.Trigger()
	d.Trigger()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("debouncer did not fire")
	}

	assert.Equal(t, int32(1), count.Load())
}

func TestDebouncerCancelPreventsFire(t *testing.T) {
	var count atomic.Int32

	d := NewDebouncer(30*time.Millisecond, func() {
		count.Add(1)
	})

	d.Trigger()
	d.Cancel()
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(0), count.Load())
}

func TestDebouncerResetExtendsQuietPeriod(t *testing.T) {
	var count atomic.Int32

	d := NewDebouncer(40*time.Millisecond, func() {
		count.Add(1)
	})

	d.Trigger()
	time.Sleep(25 * time.Millisecond)
	d.Trigger()
	time.Sleep(25 * time.Millisecond)

	assert.Equal(t, int32(0), count.Load())

	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, int32(1), count.Load())
}
