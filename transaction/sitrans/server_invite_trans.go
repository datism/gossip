package sitrans

import (
	"gossip/event"
	"gossip/message"
	"gossip/transaction"
	"gossip/util"
)

const T1 = 500
const T2 = 4000
const T4 = 5000

const tiprv_dur = 200
const tig_dur = T1
const tih_dur = 64 * T1
const tii_dur = T4

const (
	timer_prv = iota
	timer_g
	timer_h
	timer_i
)

type state int

const (
	proceeding state = iota
	completed
	confirmed
	terminated
)

type Sitrans struct {
	state    state
	message  *message.SIPMessage
	last_res *message.SIPMessage
	timers   [4]util.Timer
	transc   chan event.Event
	trpt_cb  func(transaction.Transaction, event.Event)
	core_cb  func(transaction.Transaction, event.Event)
}

func Make(
	message *message.SIPMessage,
	transport_callback func(transaction.Transaction, event.Event),
	core_callback func(transaction.Transaction, event.Event),
) *Sitrans {
	timerprv := util.NewTimer()
	timerg := util.NewTimer()
	timerdh := util.NewTimer()
	timeri := util.NewTimer()

	return &Sitrans{
		message: message,
		transc:  make(chan event.Event),
		timers:  [4]util.Timer{timerprv, timerg, timerdh, timeri},
		state:   proceeding,
		trpt_cb: transport_callback,
		core_cb: core_callback,
	}
}

func (trans Sitrans) Event(event event.Event) {
	trans.transc <- event
}

func (trans *Sitrans) Start() {
	go trans.start()
}

func (ctx *Sitrans) start() {
	ctx.timers[timer_prv].Start(tiprv_dur)
	call_core_callback(ctx, event.Event{Type: event.MESS, Data: ctx.message})

	var ev event.Event

	for {
		select {
		case ev = <-ctx.transc:
		case <-ctx.timers[timer_prv].Chan():
			ev = event.Event{Type: event.TIMEOUT, Data: timer_prv}
		case <-ctx.timers[timer_g].Chan():
			ev = event.Event{Type: event.TIMEOUT, Data: timer_g}
		case <-ctx.timers[timer_h].Chan():
			ev = event.Event{Type: event.TIMEOUT, Data: timer_h}
		}

		ctx.handle_event(ev)
		if ctx.state == terminated {
			break
		}
	}

}

func (ctx *Sitrans) handle_event(ev event.Event) {
	switch ev.Type {
	case event.TIMEOUT:
		ctx.handle_timer(ev)
	case event.MESS:
		ctx.handle_recv_msg(ev)
	default:
		return
	}
}

func (trans *Sitrans) handle_timer(ev event.Event) {
	if ev.Data == timer_h {
		call_core_callback(trans, event.Event{Type: event.TIMEOUT, Data: timer_h})
		trans.state = terminated
	} else if ev.Data == timer_prv && trans.state == proceeding {
		trying100 := message.MakeGenericResponse(100, "TRYING", trans.message)
		call_transport_callback(trans, event.Event{Type: event.MESS, Data: trying100})
	} else if ev.Data == timer_g && trans.state == completed {
		call_transport_callback(trans, event.Event{Type: event.MESS, Data: trans.last_res})
		trans.timers[timer_g].Start(min(2*trans.timers[timer_g].Duration(), T2))
	} else if ev.Data == timer_i && trans.state == confirmed {
		trans.state = terminated
	}
}

func (trans *Sitrans) handle_recv_msg(ev event.Event) {
	msg, ok := ev.Data.(*message.SIPMessage)
	if !ok {
		return
	}

	if msg.Response == nil {
		if msg.Request.Method == "ACK" && trans.state == completed {
			trans.timers[timer_g].Stop()
			trans.timers[timer_h].Stop()
			trans.timers[timer_i].Start(tii_dur)
			trans.state = confirmed
		}
		return
	}

	status_code := msg.Response.StatusCode
	if status_code >= 100 && status_code < 200 && trans.state == proceeding {
		trans.timers[timer_prv].Stop()
		call_transport_callback(trans, event.Event{Type: event.MESS, Data: msg})
	} else if status_code >= 200 && status_code <= 300 && trans.state == proceeding {
		call_transport_callback(trans, event.Event{Type: event.MESS, Data: msg})
		trans.state = terminated
	} else if status_code > 300 {
		call_transport_callback(trans, event.Event{Type: event.MESS, Data: msg})
		trans.timers[timer_g].Start(tig_dur)
		trans.timers[timer_h].Start(tih_dur)
		trans.last_res = msg
		trans.state = completed
	}
}

func call_core_callback(sitrans *Sitrans, ev event.Event) {
	go sitrans.core_cb(sitrans, ev)
}

func call_transport_callback(sitrans *Sitrans, ev event.Event) {
	go sitrans.trpt_cb(sitrans, ev)
}
