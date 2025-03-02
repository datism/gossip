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

var (
	mu sync.Mutex
	m  = make(map[transaction.TransID]transaction.Transaction)
)

func HandleMessage(msg *message.SIPMessage) {
	log.Trace().Str("event", "handle_sip_message").Interface("sip_message", msg).Msg("Handle message")

	mu.Lock()
	if trans := FindTransaction(msg); trans != nil {
		trans.Event(util.Event{Type: util.MESSAGE, Data: msg})
		mu.Unlock()
	} else {
		mu.Unlock()
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
		trans = istrans.Make(*tid, msg, tranport_cb, core_cb)
	} else {
		trans = nistrans.Make(*tid, msg, tranport_cb, core_cb)
	}

	log.Debug().Msg("Start server transaction with trans id: " + tid.String())

	mu.Lock()
	m[*tid] = trans
	mu.Unlock()
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
		trans = ictrans.Make(*tid, msg, tranport_cb, core_cb)
	} else {
		trans = nictrans.Make(*tid, msg, tranport_cb, core_cb)
	}

	log.Debug().Msg("Start client transaction with trans id: " + tid.String())

	mu.Lock()
	m[*tid] = trans
	mu.Unlock()
	go trans.Start()
	return trans
}

func DeleteTransaction(tid transaction.TransID) {
	log.Debug().Msg("Delete transaction with ID: " + tid.String())
	mu.Lock()
	delete(m, tid)
	mu.Unlock()
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

	if result, ok := m[*tid]; ok {
		log.Debug().Msg("Found transaction with ID: " + tid.String())
		return result
	}

	return nil
}
