package core

import (
	"gossip/transaction"
  "gossip/message"
  "gossip/util"
	"github.com/rs/zerolog/log"
)

type proxy struct {
	Chan chan util.Event
	client_trans transaction.Transaction
	server_trans transaction.Transaction
	
}

func Make(request *message.SIPmessage) { 
	ch = make(chan util.Event, 3)
  tid, err := transaction.MakeTransactionID(msg)
	if err != nil {
		log.Error().Err(err).Msg("Cannot create transaction ID")
		return
	}

  core_cb = func(from transaction.Transaction, ev util.Event) {
    ch 
  }

  StartServerTransaction(tid, request, trpt_cb, core_cb)
  
  
}

func start() {
  
}

// func MessageProcessing(transChan chan transaction.Event) {
// 	for event := range transChan {
// 		switch event.Type {
// 		case transaction.RECV:
// 			log.Debug().Msg("Handling messsage")
// 		default:
// 			log.Debug().Msg("Unexpected message")
// 		}
// 	}
// }
