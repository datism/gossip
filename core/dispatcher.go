package core

import (
	"gossip/sipmess"
	"gossip/siptrans"
	"gossip/siptransp"
	"sync"

	"github.com/rs/zerolog/log"
)

var (
	mu sync.Mutex
	m  = make(map[siptrans.TransID]siptrans.Transaction)
)

func HandleMessage(msg *sipmess.SIPMessage, transport *siptransp.Transport) {
	log.Trace().Interface("message", msg).Msg("Handle message")

	mu.Lock()
	if trans := FindTrans(msg); trans != nil {
		trans.Event(msg)
		mu.Unlock()
	} else {
		mu.Unlock()
		if msg.Request == nil {
			//log.Error().Msg("Cannot start new siptrans with response")
			return
		}

		if msg.Request.Method == sipmess.Ack {
			log.Debug().Msg("Cannot start new siptrans with ack request...process stateless")
			StatelessRoute(msg, transport)
			return
		}

		StatefullRoute(msg, transport)
	}
}

func StartServerTrans(
	msg *sipmess.SIPMessage,
	transport *siptransp.Transport,
	core_cb func(*siptransp.Transport, *sipmess.SIPMessage),
	tranport_cb func(*siptransp.Transport, *sipmess.SIPMessage) bool,
	term_cb func(siptrans.TransID, siptrans.TERM_REASON),
) siptrans.Transaction {
	tid, err := siptrans.MakeServerTransactionID(msg)
	if err != nil {
		log.Error().Msg("Cannot create siptrans")
		return nil
	}

	var trans siptrans.Transaction

	if msg.Request.Method == sipmess.Invite {
		trans = siptrans.MakeSIT(tid, msg, transport, core_cb, tranport_cb, term_cb)
	} else {
		trans = siptrans.MakeNIST(tid, msg, transport, core_cb, tranport_cb, term_cb)
	}

	log.Debug().Msg("Start server siptrans with trans id: " + tid.String())

	mu.Lock()
	m[tid] = trans
	mu.Unlock()
	go trans.Start()
	return trans
}

func StartClientTrans(
	msg *sipmess.SIPMessage,
	transport *siptransp.Transport,
	core_cb func(*siptransp.Transport, *sipmess.SIPMessage),
	tranport_cb func(*siptransp.Transport, *sipmess.SIPMessage) bool,
	term_cb func(siptrans.TransID, siptrans.TERM_REASON),
) siptrans.Transaction {

	tid, err := siptrans.MakeClientTransactionID(msg)
	if err != nil {
		log.Error().Msg("Cannot create siptrans ID")
		return nil
	}

	var trans siptrans.Transaction

	if msg.CSeq.Method == sipmess.Invite {
		trans = siptrans.MakeICT(tid, msg, transport, core_cb, tranport_cb, term_cb)
	} else {
		trans = siptrans.MakeNICT(tid, msg, transport, core_cb, tranport_cb, term_cb)
	}

	log.Debug().Msg("Start client siptrans with trans id: " + tid.String())

	mu.Lock()
	m[tid] = trans
	mu.Unlock()
	go trans.Start()
	return trans
}

func DeleteTrans(tid siptrans.TransID) {
	log.Debug().Msg("Delete siptrans with ID: " + tid.String())
	mu.Lock()
	delete(m, tid)
	mu.Unlock()
}

func FindTrans(msg *sipmess.SIPMessage) siptrans.Transaction {
	var tid siptrans.TransID
	var err error
	if msg.Request != nil {
		tid, err = siptrans.MakeServerTransactionID(msg)
	} else {
		tid, err = siptrans.MakeClientTransactionID(msg)
	}

	if err != nil {
		log.Error().Msg("Cannot create siptrans ID")
		return nil
	}

	if result, ok := m[tid]; ok {
		log.Debug().Msg("Found siptrans with ID: " + tid.String())
		return result
	}

	return nil
}
