package core

import (
	"gossip/message"
	"gossip/transaction"
	"gossip/transaction/ictrans"
	"gossip/transaction/istrans"
	"gossip/transaction/nictrans"
	"gossip/transaction/nistrans"
	"gossip/util"
	"sync"

	"github.com/rs/zerolog/log"
)

var m sync.Map

func HandleMessage(msg *message.SIPMessage) {
	log.Debug().Str("event", "handle_sip_message").Interface("sip_message", msg).Msg("Handle message")

	if trans := FindTransaction(msg); trans != nil {
		trans.Event(util.Event{Type: util.MESS, Data: msg})
	} else {
		if msg.Request == nil {
			log.Error().Msg("Cannot start new transaction with response")
			return
		}

		if msg.Request.Method == "ACK" {
			log.Debug().Msg("Cannot start new transaction with ack request...process stateless")
			Stateless_route(msg)
			return
		}

		Statefull_route(msg)
	}
}

func StartServerTransaction(
	msg *message.SIPMessage,
	core_cb func(transaction.Transaction, util.Event),
	tranport_cb func(transaction.Transaction, util.Event),
) transaction.Transaction {
	tid := transaction.MakeServerTransactionID(msg)
	if tid == nil {
		log.Error().Msg("Cannot create transaction ID")
		return nil
	}

	var trans transaction.Transaction

	if msg.Request.Method == "INVITE" {
		trans = istrans.Make(msg, tranport_cb, core_cb)
	} else {
		trans = nistrans.Make(msg, tranport_cb, core_cb)
	}

	log.Debug().Msg("Start server transaction with trans id: " + tid.String())

	m.Store(*tid, trans)
	go trans.Start()
	return trans
}

func StartClientTransaction(
	msg *message.SIPMessage,
	core_cb func(transaction.Transaction, util.Event),
	tranport_cb func(transaction.Transaction, util.Event),
) transaction.Transaction {

	tid := transaction.MakeClientTransactionID(msg)
	if tid == nil {
		log.Error().Msg("Cannot create transaction ID")
		return nil
	}

	var trans transaction.Transaction

	if msg.CSeq.Method == "INVITE" {
		trans = ictrans.Make(msg, tranport_cb, core_cb)
	} else {
		trans = nictrans.Make(msg, tranport_cb, core_cb)
	}

	log.Debug().Msg("Start client transaction with trans id: " + tid.String())

	m.Store(*tid, trans)
	go trans.Start()
	return trans
}

func FindTransaction(msg *message.SIPMessage) transaction.Transaction {
	var tid *transaction.TransID
	if msg.Request != nil {
		tid = transaction.MakeServerTransactionID(msg)
	} else {
		tid = transaction.MakeClientTransactionID(msg)
	}

	if tid == nil {
		log.Error().Msg("Cannot create transaction ID")
		return nil
	}

	if result, ok := m.Load(*tid); ok {
		log.Debug().Msg("Found transaction with ID: " + tid.String())
		return result.(transaction.Transaction)
	}

	return nil
}
