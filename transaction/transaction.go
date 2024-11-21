package transaction

import (
	"gossip/messsage"
	"sync"
)

type TransID struct {
	BranchID string
	Method string
}

type EventType int

const (
	RECEIVED = iota
	SEND
	TIMER
	TIMEOUT
)

type Event struct {
	Type EventType
	Data *interface{} 
}

type TransType int 

const (
	INVITE_CLIENT = iota
	NON_INVITE_CLIENT
	INVITE_SERVER
	NON_INVITE_SERVER
)

type TransCom struct {
	Sendtu <-chan *Event 
	Recvtu chan<- *Event
}

type TransContext struct {
	Type TransType
	ID *TransID
}

type Transaction interface {
	event() 
}


var m sync.Map


func startTransaction(transType TransType, transID *TransID, messsage *messsage.SIPMessage, sendtu <-chan *Event, recvtu chan<- *Event) {
	transContext := TransContext{ID: transID, Type: transType}
	registerTransaction(transID, sendtu, recvtu)
}

func registerTransaction(transID *TransID,  sendtu <-chan *Event, recvtu chan<- *Event) {
	m.Store(&transID, &TransCom{Sendtu: sendtu, Recvtu: recvtu})
}