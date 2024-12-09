package core

import (
	"gossip/transaction"
	"github.com/rs/zerolog/log"
)

func MessageProcessing(transChan chan transaction.Event) {
	for event := range transChan {
		switch event.Type {
		case transaction.RECV:
			log.Debug().Msg("Handling messsage")
		default:
			log.Debug().Msg("Unexpected message")
		}
	}
}