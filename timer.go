package main

import (
	"time"
)

// implement a timer keeping track of end time
// to calculate TimeRemaining
type SecondsTimer struct {
	timer *time.Timer
	end   time.Time
}

func NewSecondsTimer(t time.Duration) *SecondsTimer {
	return &SecondsTimer{time.NewTimer(t), time.Now().Add(t)}
}

func (s *SecondsTimer) Reset(t time.Duration) {
	s.timer.Reset(t)
	s.end = time.Now().Add(t)
}

func (s *SecondsTimer) Stop() {
	s.timer.Stop()
}

func (s *SecondsTimer) TimeRemaining() time.Duration {
	remaining := s.end.Sub(time.Now())
	if remaining > 0 {
		return remaining
	} else {
		return time.Duration(0)
	}
}
