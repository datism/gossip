package util

import (
	"time"
)

type Timer struct {
	timer    *time.Timer
	duration int
}

func NewTimer() Timer {
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	return Timer{timer: timer, duration: 0}
}

func (t *Timer) Start(duration int) {
	if !t.timer.Stop() {
		select {
		case <-t.timer.C:
		default:
		}
	}

	t.duration = duration
	t.timer.Reset(time.Duration(duration) * time.Millisecond)
}

func (t *Timer) Stop() {
	if !t.timer.Stop() {
		select {
		case <-t.timer.C:
		default:
		}
	}
}

func (t Timer) Chan() <-chan time.Time {
	return t.timer.C
}

func (t Timer) Duration() int {
	return t.duration
}
