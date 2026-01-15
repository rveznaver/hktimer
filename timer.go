package main

import (
	"sync/atomic"
	"time"
)

// SecondsTimer wraps time.Timer and tracks the end time atomically.
// This allows thread-safe access to timer state from multiple goroutines,
// particularly for calculating time remaining and formatting end times.
type SecondsTimer struct {
	timer *time.Timer
	end   atomic.Value // stores time.Time - provides lock-free thread safety
}

// NewSecondsTimer creates a new timer that will fire after duration t.
// The timer starts immediately and the end time is calculated and stored atomically.
func NewSecondsTimer(t time.Duration) *SecondsTimer {
	st := &SecondsTimer{timer: time.NewTimer(t)}
	st.end.Store(time.Now().Add(t))
	return st
}

// Reset changes the timer to expire after duration t.
// It safely handles the case where the timer has already fired by draining
// the channel in a non-blocking way. This follows Go's timer best practices.
func (s *SecondsTimer) Reset(t time.Duration) {
	// Attempt to stop the timer first
	if !s.timer.Stop() {
		// Timer already fired or was stopped - drain channel if needed
		select {
		case <-s.timer.C:
			// Successfully drained
		default:
			// Channel was already empty, nothing to do
		}
	}
	// Now it's safe to reset the timer
	s.timer.Reset(t)
	s.end.Store(time.Now().Add(t))
}

// Stop prevents the timer from firing.
// It returns true if the call stops the timer, false if the timer has already
// expired or been stopped.
func (s *SecondsTimer) Stop() bool {
	return s.timer.Stop()
}

// TimeRemaining returns the duration until the timer fires.
// Returns 0 if the timer has already expired or is not set.
// This method is thread-safe and can be called from multiple goroutines.
func (s *SecondsTimer) TimeRemaining() time.Duration {
	endTime := s.end.Load().(time.Time)
	remaining := endTime.Sub(time.Now())
	if remaining > 0 {
		return remaining
	}
	return time.Duration(0)
}

// End returns the time when the timer will expire.
// This method is thread-safe and can be called from multiple goroutines.
func (s *SecondsTimer) End() time.Time {
	return s.end.Load().(time.Time)
}

// C returns the timer's channel for receiving expiration events.
// The channel is read-only to prevent external code from sending values.
// Callers should read from this channel to know when the timer fires.
func (s *SecondsTimer) C() <-chan time.Time {
	return s.timer.C
}
