package core

import (
	"gossip/message"
	"gossip/transaction"
	"gossip/transaction/ictrans"
	"gossip/transaction/istrans"
	"gossip/transaction/nictrans"
	"gossip/transaction/nistrans"
	"gossip/transport"
	"gossip/util"
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
		trans.Event(util.Event{Type: util.MESS, Data: msg})
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

		log.Debug().Msg("Create start transaction with trans id: " + tid.String())
	}
}

func StartServerTransaction(
	msg *message.SIPMessage,
	core_cb func(transaction.Transaction, util.Event),
	tranport_cb func(transaction.Transaction, util.Event),
) transaction.Transaction {
	tid, err := transaction.MakeTransactionID(msg)
	if err != nil {
		log.Error().Err(err).Msg("Cannot create transaction ID")
		return nil
	}

	var trans transaction.Transaction

	if msg.Request.Method == "INVITE" {
		trans = istrans.Make(msg, tranport_cb, core_cb)
	} else {
		trans = nistrans.Make(msg, tranport_cb, core_cb)
	}

	m.Store(&tid, trans)
	go trans.Start()
	return trans
}

func StartClientTransaction(
	msg *message.SIPMessage,
	core_cb func(transaction.Transaction, util.Event),
	tranport_cb func(transaction.Transaction, util.Event),
) transaction.Transaction {

	tid, err := transaction.MakeTransactionID(msg)
	if err != nil {
		log.Error().Err(err).Msg("Cannot create transaction ID")
		return nil
	}

	var trans transaction.Transaction

	if msg.CSeq.Method == "INVITE" {
		trans = ictrans.Make(msg, tranport_cb, core_cb)
	} else {
		trans = nictrans.Make(msg, tranport_cb, core_cb)
	}

	m.Store(&tid, trans)
	go trans.Start()
	return trans
}

func FindTransaction(transID *transaction.TransID) transaction.Transaction {
	if trans, ok := m.Load(transID); ok {
		if transCom, ok := trans.(transaction.Transaction); ok {
			return transCom
		}
	}

	return nil
}
