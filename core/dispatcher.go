package core

import (
	"gossip/sipmess"
	"gossip/transaction"
	"gossip/transaction/ictrans"
	"gossip/transaction/istrans"
	"gossip/transaction/nictrans"
	"gossip/transaction/nistrans"
	"gossip/transport"
	"sync"

	"github.com/rs/zerolog/log"
)

var (
	mu sync.Mutex
	m  = make(map[transaction.TransID]transaction.Transaction)
)

func HandleMessage(msg *sipmess.SIPMessage, transport *transport.Transport) {
	log.Trace().Interface("message", msg).Msg("Handle message")

	mu.Lock()
	if trans := FindTransaction(msg); trans != nil {
		trans.Event(msg)
		mu.Unlock()
	} else {
		mu.Unlock()
		if msg.Request == nil {
			//log.Error().Msg("Cannot start new transaction with response")
			return
		}

		if msg.Request.Method == sipmess.Ack {
			log.Debug().Msg("Cannot start new transaction with ack request...process stateless")
			StatelessRoute(msg, transport)
			return
		}

		StatefullRoute(msg, transport)
	}
}

func StartServerTransaction(
	msg *sipmess.SIPMessage,
	transport *transport.Transport,
	core_cb func(*transport.Transport, *sipmess.SIPMessage),
	tranport_cb func(*transport.Transport, *sipmess.SIPMessage) bool,
	term_cb func(transaction.TransID, transaction.TERM_REASON),
) transaction.Transaction {
	tid, err := transaction.MakeServerTransactionID(msg)
	if err != nil {
		log.Error().Msg("Cannot create transaction ID")
		return nil
	}

	var trans transaction.Transaction

	if msg.Request.Method == sipmess.Ack {
		trans = istrans.Make(tid, msg, transport, core_cb, tranport_cb, term_cb)
	} else {
		trans = nistrans.Make(tid, msg, transport, core_cb, tranport_cb, term_cb)
	}

	log.Debug().Msg("Start server transaction with trans id: " + tid.String())

	mu.Lock()
	m[tid] = trans
	mu.Unlock()
	go trans.Start()
	return trans
}

func StartClientTransaction(
	msg *sipmess.SIPMessage,
	transport *transport.Transport,
	core_cb func(*transport.Transport, *sipmess.SIPMessage),
	tranport_cb func(*transport.Transport, *sipmess.SIPMessage) bool,
	term_cb func(transaction.TransID, transaction.TERM_REASON),
) transaction.Transaction {

	tid, err := transaction.MakeClientTransactionID(msg)
	if err != nil {
		log.Error().Msg("Cannot create transaction ID")
		return nil
	}

	var trans transaction.Transaction

	if msg.CSeq.Method == sipmess.Invite {
		trans = ictrans.Make(tid, msg, transport, core_cb, tranport_cb, term_cb)
	} else {
		trans = nictrans.Make(tid, msg, transport, core_cb, tranport_cb, term_cb)
	}

	log.Debug().Msg("Start client transaction with trans id: " + tid.String())

	mu.Lock()
	m[tid] = trans
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

func FindTransaction(msg *sipmess.SIPMessage) transaction.Transaction {
	var tid transaction.TransID
	var err error
	if msg.Request != nil {
		tid, err = transaction.MakeServerTransactionID(msg)
	} else {
		tid, err = transaction.MakeClientTransactionID(msg)
	}

	if err != nil {
		log.Error().Msg("Cannot create transaction ID")
		return nil
	}

	if result, ok := m[tid]; ok {
		log.Debug().Msg("Found transaction with ID: " + tid.String())
		return result
	}

	return nil
}
