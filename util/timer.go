package util

import (
	"time"
)

type Timer struct{
	timer *time.Timer
	duration int
	is_running bool
}

func NewTimer() Timer{
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<- timer.C
	}
	return Timer{timer: timer, duration: 0, is_running: false}
}

func (t *Timer) Start(duration int) {
	if t.is_running {
		if !t.timer.Stop() {
			<- t.timer.C
		}
	}
	
	t.timer.Reset(time.Duration(duration) * time.Microsecond)
	t.is_running = true
} 

func (t *Timer) Stop() {
	if t.is_running {
		if !t.timer.Stop() {
			<- t.timer.C
		}
	}

	t.is_running = false
}

func (t Timer) Chan() <-chan time.Time {
	return t.timer.C
}

func (t Timer) Duration() int {
	return t.duration
}

