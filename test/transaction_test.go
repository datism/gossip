package test

import (
	"gossip/message"
	"gossip/message/cseq"
	"gossip/message/via"
	"gossip/transaction"
	"reflect"
	"testing"
)

func TestMakeTransactionIDFromRequest(t *testing.T) {
	data := &message.SIPMessage{
		Startline: message.Startline{
			Request: &message.Request{
				Method:     "INVITE",
			},
		},
		TopmostVia: &via.SIPVia{
			Domain: "server10.biloxi.com",
			Branch: "123",
		},
	}

	expected := &transaction.TransID{
		BranchID: "123",
		Method:   "INVITE",
		SentBy:   "server10.biloxi.com",
	}

	tid, err := transaction.MakeTransactionID(data)
	if err != nil {
		t.Fatalf("Error make transaction ID: %v", err)
	}

	if !reflect.DeepEqual(tid, expected) {
		t.Errorf("Make transaction ID from request does not match expected result.\nGot: %+v\nExpected: %+v", tid, expected)
	}
}

func TestMakeTransactionIDFromResponse(t *testing.T) {
	data := &message.SIPMessage{
		TopmostVia: &via.SIPVia{
			Branch: "123",
		},
		CSeq: &cseq.SIPCseq{
			Method: "INVITE",
		},
	}

	expected := &transaction.TransID{
		BranchID: "123",
		Method:   "INVITE",
		SentBy:   "",
	}

	tid, err := transaction.MakeTransactionID(data)
	if err != nil {
		t.Fatalf("Error make transaction ID: %v", err)
	}

	if !reflect.DeepEqual(tid, expected) {
		t.Errorf("Make transaction ID from response does not match expected result.\nGot: %+v\nExpected: %+v", tid, expected)
	}
}
