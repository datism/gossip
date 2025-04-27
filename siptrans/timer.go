package siptrans

import (
	"time"
)

// Constants representing timer durations in milliseconds
const t1 = 500
const t2 = 4000
const t4 = 5000
const tia_dur = t1
const tib_dur = 64 * t1
const tid_dur = 32000
const tiprovsion_dur = 120
const tig_dur = t1
const tih_dur = 64 * t1
const tii_dur = t4
const tif_dur = 64 * t1 // Timer F duration (64*T1)
const tie_dur = t1
const tik_dur = t4
const tij_dur = 64 * t1

type transTimer struct {
	ID       string
	Timer    *time.Timer
	Duration int
}

func newTransTimer(ID string) *transTimer {
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	return &transTimer{ID: ID, Timer: timer, Duration: 0}
}

func (t *transTimer) start(duration int) {
	if !t.Timer.Stop() {
		select {
		case <-t.Timer.C:
		default:
		}
	}

	t.Duration = duration
	t.Timer.Reset(time.Duration(duration) * time.Millisecond)
}

func (t *transTimer) stop() {
	if !t.Timer.Stop() {
		select {
		case <-t.Timer.C:
		default:
		}
	}
}
