package transaction

import (
	"fmt"
	"gossip/message"
)

type TransType int

const (
	INVITE_CLIENT = iota
	NON_INVITE_CLIENT
	INVITE_SERVER
	NON_INVITE_SERVER
)

type TERM_REASON int

func (t TERM_REASON) String() string {
	switch t {
	case NORMAL:
		return "NORMAL"
	case TIMEOUT:
		return "TIMEOUT"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

const (
	NORMAL = iota
	TIMEOUT
	ERROR
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

type Transaction interface {
	Event(*message.SIPMessage)
	Start()
}

/*
	 RFC3261
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
func MakeServerTransactionID(msg *message.SIPMessage) (TransID, error) {
	topmostVia := msg.TopmostVia
	branch := topmostVia.Branch

	if msg.Request == nil {
		return TransID{}, fmt.Errorf("request is nil")
	}

	method := msg.Request.Method
	if method == "ACK" {
		method = "INVITE"
	}

	return TransID{
		BranchID: branch,
		Method:   method,
		SentBy:   topmostVia.Domain,
	}, nil
}

func MakeClientTransactionID(msg *message.SIPMessage) (TransID, error) {
	topmostVia := msg.TopmostVia
	branch := topmostVia.Branch

	var method string
	if msg.Request != nil {
		method = msg.Request.Method
	} else {
		method = msg.CSeq.Method
	}

	return TransID{
		BranchID: branch,
		Method:   method,
	}, nil
}
