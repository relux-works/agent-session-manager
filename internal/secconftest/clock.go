package secconftest

import (
	"sync"
	"time"
)

// Clock abstracts time for code under test. Production code that needs
// the time takes a Clock; tests pass a Fake. A Fake never reads the
// wall clock, so suite timing cannot leak into results.
type Clock interface {
	Now() time.Time
}

// fixedStart pins every Fake to one answer until advanced. The value is
// arbitrary and carries no product meaning.
var fixedStart = time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)

// Fake is a manually advanced clock. The zero value is usable and reads
// fixedStart. It calls nothing outside its own mutex: in particular it
// never calls time.Now, so a real-clock difference (a sleep, a slow
// host, midnight) cannot change its answer.
type Fake struct {
	mutex   sync.Mutex
	current time.Time
	set     bool
}

// Now reports the fake time.
func (clock *Fake) Now() time.Time {
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	if !clock.set {
		return fixedStart
	}
	return clock.current
}

// Advance moves the fake time forward by delta. A negative delta is a
// no-op: clocks under test never run backwards through this type.
func (clock *Fake) Advance(delta time.Duration) {
	if delta < 0 {
		return
	}
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	if !clock.set {
		clock.current = fixedStart
		clock.set = true
	}
	clock.current = clock.current.Add(delta)
}

// Set pins the fake time to an exact value.
func (clock *Fake) Set(moment time.Time) {
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	clock.current = moment
	clock.set = true
}

// Recording wraps a Clock and counts Now calls. Passing a Recording
// into code under test and then reading Calls tells the suite whether
// the fake was actually wired: zero calls after a clock-dependent
// operation means the code read the wall clock (or no clock) instead,
// and the suite must treat that as a failure, not as a pass.
type Recording struct {
	inner Clock
	mutex sync.Mutex
	calls int64
}

// Wrap counts Now calls through inner.
func Wrap(inner Clock) *Recording {
	return &Recording{inner: inner}
}

// Now delegates and records the call.
func (clock *Recording) Now() time.Time {
	clock.mutex.Lock()
	clock.calls++
	clock.mutex.Unlock()
	return clock.inner.Now()
}

// Calls reports how many times Now was read through this wrapper.
func (clock *Recording) Calls() int64 {
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	return clock.calls
}
