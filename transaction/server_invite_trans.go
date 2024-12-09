package transaction

import (
	"gossip/transport"
	"gossip/message"
	"github.com/rs/zerolog/log"
)

type context struct{
	transa *Transaction
	transp *transport.Transport
	mess *message.SIPMessage
}

func start(transaction *Transaction, transport *transport.Transport, message *message.SIPMessage) {
	ctx := context{transa: transaction, transp: transport, mess: message}

	for {
		select {
		case transpEvent := <-transaction.TransportChannel:
			log.Debug().Interface("Event", transpEvent).Msg("Received event from transport")
		case coreEvent := <-transaction.TransportChannel:
			log.Debug().Interface("Event", coreEvent).Msg("Received event from core")
		}
	}
}