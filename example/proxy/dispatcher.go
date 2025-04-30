package proxy

import (
	"sync"

	"github.com/datism/sip"
	"github.com/rs/zerolog/log"
)

var (
	mu sync.Mutex
	m  = make(map[sip.TransID]sip.SIPTransaction)
)

func HandleMessage(msg *sip.SIPMessage, transport *sip.SIPTransport) {
	log.Trace().Interface("message", msg).Msg("Handle message")

	mu.Lock()
	if trans := FindTrans(msg); trans != nil {
		trans.Event(msg)
		mu.Unlock()
	} else {
		mu.Unlock()
		if msg.Request == nil {
			//log.Error().Msg("Cannot start new sip with response")
			return
		}

		if msg.Request.Method == sip.Ack {
			log.Debug().Msg("Cannot start new sip with ack request...process stateless")
			StatelessRoute(msg, transport)
			return
		}

		StatefullRoute(msg, transport)
	}
}

func StartServerTrans(
	msg *sip.SIPMessage,
	transport *sip.SIPTransport,
	core_cb func(*sip.SIPTransport, *sip.SIPMessage),
	tranport_cb func(*sip.SIPTransport, *sip.SIPMessage) bool,
	term_cb func(sip.TransID, sip.TERM_REASON),
) sip.SIPTransaction {
	tid, err := sip.MakeServerTransactionID(msg)
	if err != nil {
		log.Error().Msg("Cannot create sip")
		return nil
	}

	var trans sip.SIPTransaction

	if msg.Request.Method == sip.Invite {
		trans = sip.MakeIST(tid, msg, transport, core_cb, tranport_cb, term_cb)
	} else {
		trans = sip.MakeNIST(tid, msg, transport, core_cb, tranport_cb, term_cb)
	}

	log.Debug().Msg("Start server sip with trans id: " + tid.String())

	mu.Lock()
	m[tid] = trans
	mu.Unlock()
	go trans.Start()
	return trans
}

func StartClientTrans(
	msg *sip.SIPMessage,
	transport *sip.SIPTransport,
	core_cb func(*sip.SIPTransport, *sip.SIPMessage),
	tranport_cb func(*sip.SIPTransport, *sip.SIPMessage) bool,
	term_cb func(sip.TransID, sip.TERM_REASON),
) sip.SIPTransaction {

	tid, err := sip.MakeClientTransactionID(msg)
	if err != nil {
		log.Error().Msg("Cannot create sip ID")
		return nil
	}

	var trans sip.SIPTransaction

	if msg.CSeq.Method == sip.Invite {
		trans = sip.MakeICT(tid, msg, transport, core_cb, tranport_cb, term_cb)
	} else {
		trans = sip.MakeNICT(tid, msg, transport, core_cb, tranport_cb, term_cb)
	}

	log.Debug().Msg("Start client sip with trans id: " + tid.String())

	mu.Lock()
	m[tid] = trans
	mu.Unlock()
	go trans.Start()
	return trans
}

func DeleteTrans(tid sip.TransID) {
	log.Debug().Msg("Delete sip with ID: " + tid.String())
	mu.Lock()
	delete(m, tid)
	mu.Unlock()
}

func FindTrans(msg *sip.SIPMessage) sip.SIPTransaction {
	var tid sip.TransID
	var err error
	if msg.Request != nil {
		tid, err = sip.MakeServerTransactionID(msg)
	} else {
		tid, err = sip.MakeClientTransactionID(msg)
	}

	if err != nil {
		log.Error().Msg("Cannot create sip ID")
		return nil
	}

	if result, ok := m[tid]; ok {
		log.Debug().Msg("Found sip with ID: " + tid.String())
		return result
	}

	return nil
}
