package main

import (
	"sync"
	"testing"
	"time"
)

// TestNewSecondsTimer verifies that a new timer is created with correct initial state
func TestNewSecondsTimer(t *testing.T) {
	duration := 5 * time.Second
	timer := NewSecondsTimer(duration)

	if timer == nil {
		t.Fatal("NewSecondsTimer returned nil")
	}

	// Check that end time is approximately correct (within 100ms tolerance)
	expectedEnd := time.Now().Add(duration)
	actualEnd := timer.End()
	diff := actualEnd.Sub(expectedEnd).Abs()

	if diff > 100*time.Millisecond {
		t.Errorf("End time diff too large: expected ~%v, got %v (diff: %v)", expectedEnd, actualEnd, diff)
	}

	// Clean up
	timer.Stop()
}

// TestTimerReset verifies that Reset correctly updates the timer
func TestTimerReset(t *testing.T) {
	timer := NewSecondsTimer(time.Hour)
	timer.Stop()

	// Reset to a new duration
	newDuration := 10 * time.Second
	timer.Reset(newDuration)

	// Verify end time is updated
	expectedEnd := time.Now().Add(newDuration)
	actualEnd := timer.End()
	diff := actualEnd.Sub(expectedEnd).Abs()

	if diff > 100*time.Millisecond {
		t.Errorf("After Reset, end time diff too large: diff=%v", diff)
	}

	// Clean up
	timer.Stop()
}

// TestTimerStop verifies that Stop prevents timer from firing
func TestTimerStop(t *testing.T) {
	timer := NewSecondsTimer(100 * time.Millisecond)

	// Stop immediately
	stopped := timer.Stop()

	if !stopped {
		t.Error("Stop() returned false, expected true for active timer")
	}

	// Wait to ensure timer doesn't fire
	time.Sleep(200 * time.Millisecond)

	// Timer should not have fired
	select {
	case <-timer.C():
		t.Error("Timer fired after Stop()")
	default:
		// Expected - timer didn't fire
	}
}

// TestTimerTimeRemaining verifies TimeRemaining calculation
func TestTimerTimeRemaining(t *testing.T) {
	duration := 2 * time.Second
	timer := NewSecondsTimer(duration)
	defer timer.Stop()

	// Check immediately after creation
	remaining := timer.TimeRemaining()
	if remaining < time.Second || remaining > duration {
		t.Errorf("TimeRemaining outside expected range: got %v, expected ~%v", remaining, duration)
	}

	// Wait and check again
	time.Sleep(500 * time.Millisecond)
	remaining = timer.TimeRemaining()

	if remaining < 1*time.Second || remaining > 2*time.Second {
		t.Errorf("After 500ms, TimeRemaining = %v, expected ~1.5s", remaining)
	}
}

// TestTimerTimeRemainingExpired verifies TimeRemaining returns 0 for expired timer
func TestTimerTimeRemainingExpired(t *testing.T) {
	timer := NewSecondsTimer(50 * time.Millisecond)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	remaining := timer.TimeRemaining()
	if remaining != 0 {
		t.Errorf("Expired timer TimeRemaining = %v, expected 0", remaining)
	}

	// Drain the channel
	select {
	case <-timer.C():
	default:
	}
}

// TestTimerConcurrentAccess verifies thread safety
func TestTimerConcurrentAccess(t *testing.T) {
	timer := NewSecondsTimer(time.Hour)
	defer timer.Stop()

	var wg sync.WaitGroup
	iterations := 100

	// Multiple goroutines reading End()
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = timer.End()
				time.Sleep(time.Microsecond)
			}
		}()
	}

	// Multiple goroutines reading TimeRemaining()
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = timer.TimeRemaining()
				time.Sleep(time.Microsecond)
			}
		}()
	}

	// One goroutine doing Reset
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < iterations; j++ {
			timer.Reset(time.Hour)
			time.Sleep(time.Millisecond)
		}
	}()

	// Wait for all goroutines
	wg.Wait()
}

// TestTimerFiring verifies timer actually fires
func TestTimerFiring(t *testing.T) {
	duration := 100 * time.Millisecond
	timer := NewSecondsTimer(duration)

	// Wait for timer to fire
	select {
	case <-timer.C():
		// Expected - timer fired
	case <-time.After(200 * time.Millisecond):
		t.Error("Timer did not fire within expected time")
	}
}

// TestTimerResetMultipleTimes verifies multiple Reset calls work correctly
func TestTimerResetMultipleTimes(t *testing.T) {
	timer := NewSecondsTimer(time.Hour)
	timer.Stop()

	for i := 0; i < 5; i++ {
		duration := time.Duration(i+1) * time.Second
		timer.Reset(duration)

		remaining := timer.TimeRemaining()
		if remaining < duration-100*time.Millisecond || remaining > duration {
			t.Errorf("Reset #%d: TimeRemaining = %v, expected ~%v", i, remaining, duration)
		}
	}

	timer.Stop()
}

// TestTimerChannelReadOnly verifies C() returns read-only channel
func TestTimerChannelReadOnly(t *testing.T) {
	timer := NewSecondsTimer(time.Hour)
	defer timer.Stop()

	// This should compile - C() returns <-chan time.Time (read-only)
	ch := timer.C()

	// Verify it's a channel
	if ch == nil {
		t.Error("C() returned nil channel")
	}
}

// BenchmarkTimerTimeRemaining benchmarks TimeRemaining performance
func BenchmarkTimerTimeRemaining(b *testing.B) {
	timer := NewSecondsTimer(time.Hour)
	defer timer.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = timer.TimeRemaining()
	}
}

// BenchmarkTimerEnd benchmarks End performance
func BenchmarkTimerEnd(b *testing.B) {
	timer := NewSecondsTimer(time.Hour)
	defer timer.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = timer.End()
	}
}

// BenchmarkTimerReset benchmarks Reset performance
func BenchmarkTimerReset(b *testing.B) {
	timer := NewSecondsTimer(time.Hour)
	defer timer.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timer.Reset(time.Hour)
	}
}
