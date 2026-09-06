package secconftest

import (
	"errors"
	"testing"
	"time"
)

var errTestSentinel = errors.New("secconftest test sentinel")

func TestFakeClockStartsPinned(t *testing.T) {
	t.Parallel()
	first := &Fake{}
	second := &Fake{}
	if !first.Now().Equal(second.Now()) {
		t.Fatalf("two fresh fakes disagree: %v vs %v", first.Now(), second.Now())
	}
	if first.Now().Location() != time.UTC {
		t.Fatalf("fake start location = %v, want UTC", first.Now().Location())
	}
}

func TestFakeClockIgnoresRealTime(t *testing.T) {
	t.Parallel()
	clock := &Fake{}
	before := clock.Now()
	wallBefore := time.Now()
	time.Sleep(20 * time.Millisecond)
	if got := clock.Now(); !got.Equal(before) {
		t.Fatalf("fake advanced without Advance: %v -> %v", before, got)
	}
	if !time.Now().After(wallBefore) {
		t.Fatal("wall clock did not advance; the test cannot tell fake from real")
	}
}

func TestFakeAdvanceAndSet(t *testing.T) {
	t.Parallel()
	clock := &Fake{}
	start := clock.Now()
	clock.Advance(90 * time.Second)
	if got := clock.Now().Sub(start); got != 90*time.Second {
		t.Fatalf("Advance(90s) moved %v", got)
	}
	clock.Advance(-time.Hour)
	if got := clock.Now().Sub(start); got != 90*time.Second {
		t.Fatalf("negative Advance moved the clock to %v", got)
	}
	pinned := time.Date(2030, time.May, 4, 12, 0, 0, 0, time.UTC)
	clock.Set(pinned)
	if got := clock.Now(); !got.Equal(pinned) {
		t.Fatalf("Set moved to %v, want %v", got, pinned)
	}
}

func TestRecordingNoticesUnwiredClock(t *testing.T) {
	t.Parallel()
	wired := Wrap(&Fake{})
	_ = wired.Now()
	if got := wired.Calls(); got != 1 {
		t.Fatalf("wired path recorded %d Now calls, want 1", got)
	}
	unwired := Wrap(&Fake{})
	// The buggy path takes the clock and never reads it (wall clock or
	// no clock instead). The suite must notice the zero, not pass it.
	if got := unwired.Calls(); got != 0 {
		t.Fatalf("unwired path recorded %d Now calls, want the detectable 0", got)
	}
}

func TestRecordingCountsConcurrentReads(t *testing.T) {
	t.Parallel()
	recording := Wrap(&Fake{})
	done := make(chan struct{})
	for range 8 {
		go func() {
			_ = recording.Now()
			done <- struct{}{}
		}()
	}
	for range 8 {
		<-done
	}
	if got := recording.Calls(); got != 8 {
		t.Fatalf("concurrent Now calls recorded %d, want 8", got)
	}
}
