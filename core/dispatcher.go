package core

import (
	"gossip/event"
	"gossip/message"
	"gossip/transaction"
	"gossip/transaction/citrans"
	"gossip/transport"
	"sync"

	"github.com/rs/zerolog/log"
)

var m sync.Map

func HandleMessage(transport *transport.Transport, msg *message.SIPMessage) {
	tid, err := transaction.MakeTransactionID(msg)
	if err != nil {
		log.Error().Err(err).Msg("Cannot create transaction ID")
		return
	}

	if trans := FindTransaction(tid); trans != nil {
		log.Debug().Msg("Found transaction")
		trans.Event(event.Event{Type: event.MESS, Data: msg})
	} else {
		if msg.Request != nil {
			log.Error().Msg("Cannot start new transaction with response")
			return
		}

		if msg.Request.Method == "ACK" {
			log.Debug().Msg("Cannot start new transaction with ack request...process stateless")
			//do something
			return
		}

		var transType transaction.TransType
		if msg.Request.Method == "INVITE" {
			transType = transaction.INVITE_SERVER
		} else {
			transType = transaction.NON_INVITE_SERVER
		}

		StartTransaction(transType, tid, transport, msg)
		log.Debug().Msg("Create start transaction with trans id: " + tid.String())
	}
}

func StartTransaction(
	transType transaction.TransType, 
	transID *transaction.TransID, 
	trpt *transport.Transport,
	msg *message.SIPMessage,
	)  {

	var trans transaction.Transaction

	switch transType {
	case transaction.INVITE_CLIENT:
		trans = citrans.Make(msg, TransportCallback, CoreCallback)
	}

	m.Store(&transID, trans)
	trans.Start()
}

func CoreCallback(from transaction.Transaction, ev event.Event) {

}

func TransportCallback(from transaction.Transaction, ev event.Event) {

}

func FindTransaction(transID *transaction.TransID) transaction.Transaction {
	if trans, ok := m.Load(transID); ok {
		if transCom, ok := trans.(transaction.Transaction); ok {
			return transCom
		}
	}

	return nil
}
