package reconcile

import (
	"sync"
	"time"
)

// Debouncer coalesces rapid triggers into a single callback after a quiet period.
type Debouncer struct {
	delay  time.Duration
	onFire func()

	mu      sync.Mutex
	timer   *time.Timer
	stopped bool
}

// NewDebouncer creates a debouncer that calls onFire after delay without further triggers.
func NewDebouncer(delay time.Duration, onFire func()) *Debouncer {
	return &Debouncer{
		delay:  delay,
		onFire: onFire,
	}
}

// Trigger schedules or resets the debounced callback.
func (d *Debouncer) Trigger() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stopped {
		return
	}

	if d.timer != nil {
		if !d.timer.Stop() {
			select {
			case <-d.timer.C:
			default:
			}
		}
	}

	d.timer = time.AfterFunc(d.delay, func() {
		d.mu.Lock()
		if d.stopped {
			d.mu.Unlock()
			return
		}
		d.mu.Unlock()
		if d.onFire != nil {
			d.onFire()
		}
	})
}

// Cancel stops pending debounced callbacks.
func (d *Debouncer) Cancel() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.stopped = true
	if d.timer != nil {
		if !d.timer.Stop() {
			select {
			case <-d.timer.C:
			default:
			}
		}
		d.timer = nil
	}
}
