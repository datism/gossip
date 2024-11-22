package test

import (
	"gossip/message"
	"gossip/transaction"
	"reflect"
	"testing"
)

func TestMakeTransactionIDFromRequest(t *testing.T) {
	data := &message.SIPMessage{
		Startline: message.Startline{
			Request: &message.Request{
				Method:     "INVITE",
				RequestURI: "sip:bob@biloxi.com",
			},
		},
		Headers: map[string][]string{
			"via":  {"SIP/2.0/UDP server10.biloxi.com;ttl=1;branch=123", "SIP/2.0/UDP server11.biloxi.com"},
			"cseq": {"314159 INVITE"},
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
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode:   200,
				ReasonPhrase: "OK",
			},
		},
		Headers: map[string][]string{
			"via":  {"SIP/2.0/UDP server10.biloxi.com;ttl=1;branch=123", "SIP/2.0/UDP server11.biloxi.com"},
			"cseq": {"314159 INVITE"},
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
