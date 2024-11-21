package core

import (
	"gossip/message"
)


func HandleMessage(msg *message.SIPMessage) {
	if (msg.Request == nil) {
		handleRequest(msg)
	} else {
		handleResponse(msg)
	}
}

func handleRequest(msg *message.SIPMessage) {

}

func handleResponse(msg *message.SIPMessage) {

}

