package core

import (
	"gossip/message"
	"gossip/transaction"
	"github.com/rs/zerolog/log"
)

func HandleMessage(msg *message.SIPMessage) {
	tid, err := transaction.MakeTransactionID(msg)
	if err != nil {
		log.Error().Err(err).Msg("Cannot create transaction ID")
		return
	}

	if trans := transaction.FindTransaction(tid); trans != nil {
		log.Debug().Msg("Found transaction")
		trans.TransportChannel <- transaction.Event{Type: transaction.RECV, Data: msg}
	} else {
		if msg.Request != nil  {
			log.Error().Msg("Cannot start new transaction with response")
			return
		}

		if msg.Request.Method == "ACK {
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

		c := transaction.StartTransaction(transType, tid, msg)
		log.Debug().Msg("Create start transaction with trans id: " + tid.String())
	}
}
