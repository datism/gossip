package citrans

import (
	"gossip/event"
	"gossip/message"
	"gossip/transaction"
	"gossip/util"
)

const T1 = 500
const tia_dur = T1
const tib_dur = 64 * T1
const tid_dur = 32000

const (
	TIMERA = iota
	TIMERB
	TIMERD
)

type state int

const (
	calling = iota
	proceeding
	completed
	terminated
)

type Citrans struct {
	state   state
	message *message.SIPMessage
	timers  [3]util.Timer
	transc  chan event.Event
	trpt_cb func(transaction.Transaction, event.Event)
	core_cb func(transaction.Transaction, event.Event)
}

func Make(
	message *message.SIPMessage,
	transport_callback func(transaction.Transaction, event.Event),
	core_callback func(transaction.Transaction, event.Event),
) *Citrans {
	timera := util.NewTimer()
	timerb := util.NewTimer()
	timerd := util.NewTimer()

	return &Citrans{
		message: message,
		transc:  make(chan event.Event),
		timers:  [3]util.Timer{timera, timerb, timerd},
		state:   calling,
		trpt_cb: transport_callback,
		core_cb: core_callback,
	}
}

func (trans Citrans) Event(event event.Event) {
	trans.transc <- event
}

func (trans *Citrans) Start() {
	go trans.start()
}

func (trans *Citrans) start() {
	call_transport_callback(trans, event.Event{Type: event.MESS, Data: trans.message})
	trans.timers[TIMERA].Start(tia_dur)
	trans.timers[TIMERB].Start(tib_dur)

	var ev event.Event

	for {
		select {
		case ev = <-trans.transc:
		case <-trans.timers[TIMERA].Chan():
			ev = event.Event{Type: event.TIMEOUT, Data: TIMERA}
		case <-trans.timers[TIMERB].Chan():
			ev = event.Event{Type: event.TIMEOUT, Data: TIMERB}
		case <-trans.timers[TIMERD].Chan():
			ev = event.Event{Type: event.TIMEOUT, Data: TIMERD}
		}

		trans.handle_event(ev)
		if trans.state == terminated {
			close(trans.transc)
			break
		}
	}

}

func (trans *Citrans) handle_event(ev event.Event) {
	switch ev.Type {
	case event.TIMEOUT:
		trans.handle_timer(ev)
	case event.MESS:
		trans.handle_recv_msg(ev)
	default:
		return
	}
}

func (trans *Citrans) handle_timer(ev event.Event) {
	if ev.Data == TIMERB {
		call_core_callback(trans, event.Event{Type: event.TIMEOUT, Data: TIMERB})
		trans.state = terminated
	} else if ev.Data == TIMERA && trans.state == calling {
		call_transport_callback(trans, event.Event{Type: event.MESS, Data: trans.message})
		trans.timers[TIMERA].Start(trans.timers[TIMERA].Duration() * 2)
	} else if ev.Data == TIMERD && trans.state == completed {
		trans.state = terminated
	}
}

func (trans *Citrans) handle_recv_msg(ev event.Event) {
	response, ok := ev.Data.(*message.SIPMessage)
	if !ok || response.Response == nil {
		return
	}

	status_code := response.Response.StatusCode
	if status_code >= 100 && status_code < 200 {
		if trans.state == calling {
			trans.timers[TIMERA].Stop()
			call_core_callback(trans, ev)
			trans.state = proceeding
		} else if trans.state == proceeding {
			call_core_callback(trans, ev)
		}
	} else if status_code >= 200 && status_code <= 300 && trans.state < completed {
		call_core_callback(trans, ev)
		trans.state = terminated
	} else if status_code > 300 {
		if trans.state < completed {
			call_core_callback(trans, ev)
			ack := message.MakeGenericAck(trans.message, response)
			call_transport_callback(trans, event.Event{Type: event.MESS, Data: ack})

			trans.timers[TIMERB].Stop()
			trans.timers[TIMERD].Start(tid_dur)
			trans.state = completed
		} else if trans.state == completed {
			ack := message.MakeGenericAck(trans.message, response)
			call_transport_callback(trans, event.Event{Type: event.MESS, Data: ack})
		}
	}
}

func call_core_callback(citrans *Citrans, ev event.Event) {
	go citrans.core_cb(citrans, ev)
}

func call_transport_callback(citrans *Citrans, ev event.Event) {
	go citrans.trpt_cb(citrans, ev)
}
