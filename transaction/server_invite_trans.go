package transaction

import (
	"time"
	"gossip/transport"
	"gossip/message"
	"github.com/rs/zerolog/log"
)

const T1 = 10
const (
	TIMER_A = iota
	TIMTER_B
	TIMER_D
)

type State int

const (
	CALLING = iota
	PROCEEDDING
	COMPLETED
	TERMINATED
)

type context struct{
	timerc chan Event
	transpc chan Event
	corec chan Event
	transp *transport.Transport
	mess *message.SIPMessage
	state State
}

func start(transaction *Transaction, transport *transport.Transport, message *message.SIPMessage) {
	ctx := sinit(transaction, transport, message) 

	for {
		select {
		case transpEvent := <-ctx.transpc:
			log.Debug().Interface("Event", transpEvent).Msg("Received event from transport")
			ctx = handle_event(ctx, transpEvent)
		case coreEvent := <-ctx.corec:
			log.Debug().Interface("Event", coreEvent).Msg("Received event from core")
			ctx = handle_event(ctx, coreEvent)
		case timerEvent := <-ctx.timerc:
			log.Debug().Interface("Event", timerEvent).Msg("Received event from timer")
			ctx = handle_event(ctx, timerEvent)
		}
	}
}

func sinit(transaction *Transaction, transport *transport.Transport, message *message.SIPMessage) (*context) {
	return &context{
		timerc: make(chan Event, 3),
		transpc: transaction.TransportChannel,
		corec: transaction.CoreChannel, 
		transp: transport, 
		mess: message, 
		state: CALLING,
	}

	
}

func handle_event(ctx *context, event Event) (*context) {

}

func start_timer(duration int, data *interface{}) {
	go func ()  {
		timer := time.NewTimer(2 * time.Second)
		<-timer.C
		
	}()
}
