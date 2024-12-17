package citrans

import (
	"gossip/message"
	"gossip/transaction"
	"gossip/transport"
	"gossip/util"
)

const T1 = 500
const T2 = 4000

const tiprv_dur = 200
const tih_dur = 64 * T1
const tig_dur = T1

const (
	timer_prv = iota
	timer_h
	timer_g
)

type state int

const (
	proceeding state = iota
	completed
	confirmed
	terminated
)

type context struct {
	recvc  chan transaction.Event
	sendc    chan transaction.Event
	timers   [3]util.Timer
	mess     *message.SIPMessage
	last_res *message.SIPMessage
	state    state
}

func Start(trans *transaction.Transaction, message *message.SIPMessage) {
	ctx := sinit(trans, message)

	var event transaction.Event
	ctx.sendc <- transaction.Event{Type: transaction.RECV, Data: message}
	ctx.timers[timer_prv].Start(tiprv_dur)

	for {
		select {
		case event = <-ctx.recvc:
		case event = <-ctx.sendc:
		case <-ctx.timers[timer_prv].Chan():
			event = transaction.Event{Type: transaction.TIMER, Data: timer_prv}
		case <-ctx.timers[timer_g].Chan():
			event = transaction.Event{Type: transaction.TIMER, Data: timer_g}
		case <-ctx.timers[timer_h].Chan():
			event = transaction.Event{Type: transaction.TIMER, Data: timer_h}
		}

		handle_event(ctx, event)
		if ctx.state == terminated {
			break
		}
	}

}

func sinit(trans *transaction.Transaction, message *message.SIPMessage) *context {
	timera := util.NewTimer()
	timerb := util.NewTimer()
	timerd := util.NewTimer()

	return &context{
		recvc: 	 trans.SendChannel,
		sendc:   trans.RecvChannel,
		timers:  [3]util.Timer{timera, timerb, timerd},
		mess:    message,
		state:   proceeding,
	}
}

func handle_event(ctx *context, event transaction.Event) {
	switch event.Type {
	case transaction.TIMER:
		handle_timer(ctx, event)
	case transaction.RECV:
		handle_recv_msg(ctx, event)
	default:
		return
	}
}

func handle_timer(ctx *context, event transaction.Event) {
	if event.Data == timer_h {
		ctx.sendc <- transaction.Event{Type: transaction.TIMER, Data: "timeout"}
		ctx.state = terminated
	} else if event.Data == timer_prv && ctx.state == proceeding {
		trying100 := message.MakeGeneralResponse(100, "TRYING", ctx.mess)
		transport.Send(trying100)
	} else if event.Data == timer_g && ctx.state == completed {
		transport.Send(ctx.last_res)
		ctx.timers[timer_g].Start(min(2*ctx.timers[timer_g].Duration(), T2))
	}
}

func handle_recv_msg(ctx *context, event transaction.Event) {
	msg, ok := event.Data.(*message.SIPMessage)
	if !ok {
		return
	}

	if msg.Response == nil {
		if msg.Request.Method == "ACK" && ctx.state == completed {
			ctx.state = terminated
		}
		return
	}

	status_code := msg.Response.StatusCode
	if status_code >= 100 && status_code < 200 && ctx.state == proceeding {
		ctx.timers[timer_prv].Stop()
		transport.Send(msg)
	} else if status_code >= 200 && status_code <= 300 && ctx.state == proceeding {
		ctx.timers[timer_prv].Stop()
		transport.Send(msg)
		ctx.state = terminated
	} else if status_code > 300 {
		transport.Send(msg)
		ctx.timers[timer_g].Start(tig_dur)
		ctx.timers[timer_h].Start(tih_dur)
		ctx.last_res = msg
		ctx.state = completed
	}
}
