package core

import (
	"gossip/message"
	"gossip/transaction"
	"gossip/util"
)

type proxy struct {
	Chan         chan util.Event
	client_trans transaction.Transaction
	server_trans transaction.Transaction
}

func Start(request *message.SIPMessage) {
	ch := make(chan util.Event, 3)

	core_cb := func(from transaction.Transaction, ev util.Event) {
		ch <- ev
	}

	trpt_cb := func(from transaction.Transaction, ev util.Event) {
		msg, ok := ev.Data.(*message.SIPMessage)
		if !ok {
			return
		}

		trprt := msg.Transport
		return
	}

	StartServerTransaction(request, core_cb, trpt_cb)
}
