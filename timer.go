package main

import (
	"sync/atomic"
	"time"
)

// implement a timer keeping track of end time
// to calculate TimeRemaining
type SecondsTimer struct {
	timer *time.Timer
	end   atomic.Value // stores time.Time
}

func NewSecondsTimer(t time.Duration) *SecondsTimer {
	st := &SecondsTimer{timer: time.NewTimer(t)}
	st.end.Store(time.Now().Add(t))
	return st
}

func (s *SecondsTimer) Reset(t time.Duration) {
	// Safely drain the channel if timer already fired
	if !s.timer.Stop() {
		select {
		case <-s.timer.C:
		default:
		}
	}
	s.timer.Reset(t)
	s.end.Store(time.Now().Add(t))
}

func (s *SecondsTimer) Stop() {
	s.timer.Stop()
}

func (s *SecondsTimer) TimeRemaining() time.Duration {
	endTime := s.end.Load().(time.Time)
	remaining := endTime.Sub(time.Now())
	if remaining > 0 {
		return remaining
	} else {
		return time.Duration(0)
	}
}

func (s *SecondsTimer) End() time.Time {
	return s.end.Load().(time.Time)
}
