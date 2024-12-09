package transaction

import (
	"errors"
	"fmt"
	"gossip/message"
	"sync"
)

type TransType int

const (
	INVITE_CLIENT = iota
	NON_INVITE_CLIENT
	INVITE_SERVER
	NON_INVITE_SERVER
)

type TransID struct {
	// Type     TransType
	BranchID string
	Method   string
	SentBy   string
}

func (tid TransID) String() string {
	return fmt.Sprintf("%s;%s;%s", tid.BranchID, tid.Method, tid.SentBy)
}

type EventType int

const (
	RECV = iota
	SEND
	TIMER
)

type Event struct {
	Type EventType
	Data interface{}
}

type Transaction struct {
	ID *TransID
	Type TransType
	TransportChannel chan Event
	CoreChannel chan Event
}

var m sync.Map

func MakeTransactionID(msg *message.SIPMessage) (*TransID, error) {
	/* RFC3261
	A response matches a client transaction under two conditions:

		1.  If the response has the same value of the branch parameter in
			the top Via header field as the branch parameter in the top
			Via header field of the request that created the transaction.

		2.  If the method parameter in the CSeq header field matches the
			method of the request that created the transaction.  The
			method is needed since a CANCEL request constitutes a
			different transaction, but shares the same value of the branch
			parameter.

	The request matches a transaction if:
		1. the branch parameter in the request is equal to the one in the
			top Via header field of the request that created the
			transaction, and

		2. the sent-by value in the top Via of the request is equal to the
			one in the request that created the transaction, and

		3. the method of the request matches the one that created the
			transaction, except for ACK, where the method of the request
			that created the transaction is INVITE.
	*/

	vias := message.GetHeader(msg, "via")
	if vias == nil {
		return nil, errors.New("empty via header")
	}
	topmostVia := vias[0]

	branch := message.GetParam(topmostVia, "branch")
	if branch == "" {
		return nil, errors.New("empty branch value")
	}

	if msg.Request == nil {
		cseq := message.GetHeader(msg, "cseq")
		if cseq == nil {
			return nil, errors.New("empty cseq header")
		}
		method := message.GetValueALWS(message.GetValue(cseq[0]))

		return &TransID{
			BranchID: branch,
			Method:   method,
		}, nil
	} else {
		method := msg.Request.Method
		if method == "ACK" {
			method = "INVITE"
		}

		return &TransID{
			BranchID: branch,
			Method:   msg.Request.Method,
			SentBy:   message.GetValueALWS(message.GetValue(topmostVia)),
		}, nil
	}
}

func StartTransaction(transID *TransID, transType TransType) (*Transaction) {
	// chan := make(chan Event, 3) 
	trans := &Transaction{ID: transID, Type: transType, Channel: make(chan Event, 3)}
	m.Store(&transID, trans)
	return trans
}

func FindTransaction(transID *TransID) *Transaction {
	if trans, ok := m.Load(transID); ok {
		if transCom, ok := trans.(*Transaction); ok {
			return transCom
		}
	}

	return nil
}
