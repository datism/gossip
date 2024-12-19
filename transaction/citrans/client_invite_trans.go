package citrans

import (
	"gossip/message"
	"gossip/transaction"
	"gossip/transport"
	"gossip/util"
)

const T1 = 500
const tia_dur = T1
const tib_dur = 64 * T1
const tid_dur = 32000

const (
	timer_a = iota
	timer_b
	timer_d
)

type state int

const (
	calling = iota
	proceeding
	completed
	terminated
)

type context struct {
	recvc  <-chan transaction.Event
	sendc  chan<- transaction.Event
	timers [3]util.Timer
	mess   *message.SIPMessage
	state  state
}

func Start(trans *transaction.Transaction, message *message.SIPMessage) {
	ctx := sinit(trans, message)

	ctx.timers[timer_b].Start(tib_dur)
	ctx.timers[timer_a].Start(tia_dur)

	var event transaction.Event

	for {
		select {
		case event = <-ctx.recvc:
		case <-ctx.timers[timer_a].Chan():
			event = transaction.Event{Type: transaction.TIMER, Data: timer_a}
		case <-ctx.timers[timer_b].Chan():
			event = transaction.Event{Type: transaction.TIMER, Data: timer_b}
		case <-ctx.timers[timer_d].Chan():
			event = transaction.Event{Type: transaction.TIMER, Data: timer_d}
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
		recvc:  trans.RecvChannel,
		sendc:  trans.SendChannel,
		timers: [3]util.Timer{timera, timerb, timerd},
		mess:   message,
		state:  calling,
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
	if event.Data == timer_b {
		ctx.sendc <- transaction.Event{Type: transaction.TIMER, Data: "timeout"}
		ctx.state = terminated
	} else if event.Data == timer_a && ctx.state == calling {
		transport.Send(ctx.mess)
		ctx.timers[timer_a].Start(ctx.timers[timer_a].Duration() * 2)
	} else if event.Data == timer_d && ctx.state == completed {
		ctx.state = terminated
	}
}

func handle_recv_msg(ctx *context, event transaction.Event) {
	response, ok := event.Data.(*message.SIPMessage)
	if !ok || response.Response == nil {
		return
	}

	status_code := response.Response.StatusCode
	if status_code >= 100 && status_code < 200 {
		if ctx.state == calling {
			ctx.timers[timer_a].Stop()
			ctx.sendc <- event
			ctx.state = proceeding
		} else if ctx.state == proceeding {
			ctx.sendc <- event
		}
	} else if status_code >= 200 && status_code <= 300 && ctx.state < completed {
		ctx.timers[timer_a].Stop()
		ctx.timers[timer_b].Stop()
		ctx.sendc <- event
		ctx.state = terminated
	} else if status_code > 300 {
		if ctx.state < completed {
			ctx.sendc <- event
			ack := message.MakeGenericAck(ctx.mess, response)
			transport.Send(ack)
			ctx.timers[timer_d].Start(tid_dur)
		} else if ctx.state == completed {
			ack := message.MakeGenericAck(ctx.mess, response)
			transport.Send(ack)
		}
	}
}
