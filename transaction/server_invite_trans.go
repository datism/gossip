package transaction

import (
	"gossip/transport"
	"gossip/message"
)

type context struct{
	transa *Transaction
	transp *transport.Transport
	mess *message.SIPMessage
}

func start(transaction *Transaction, transport *transport.Transport, message *message.SIPMessage) {
	ctx := context{transa: transaction, transp: transport, mess: message}
}