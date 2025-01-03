package transaction

import (
	"fmt"
	"gossip/util"
	"gossip/message"
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

type Transaction interface {
	Event(event util.Event)
	Start()
}


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

	// vias := message.GetHeader(msg, "via")
	// if vias == nil {
	// 	return nil, errors.New("empty via header")
	// }

	// branch := message.GetParam(topmostVia, "branch")
	// if branch == "" {
	// 	return nil, errors.New("empty branch value")
	// }

	topmostVia := msg.TopmostVia
	branch := topmostVia.Branch

	if msg.Request == nil {
		return &TransID{
			BranchID: branch,
			Method:   msg.CSeq.Method,
		}, nil
	} else {
		method := msg.Request.Method
		if method == "ACK" {
			method = "INVITE"
		}

		return &TransID{
			BranchID: branch,
			Method:   msg.Request.Method,
			SentBy:   topmostVia.Domain,
		}, nil
	}
}
