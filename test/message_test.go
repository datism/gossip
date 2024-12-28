package test

import (
	"gossip/message"
	"gossip/message/contact"
	"gossip/message/cseq"
	"gossip/message/fromto"
	"gossip/message/uri"
	"gossip/message/via"
	"reflect"
	"testing"
)

func TestParseRequest(t *testing.T) {
	data := []byte("INVITE sip:bob@biloxi.com SIP/2.0\r\n" +
		"Via: SIP/2.0/UDP server10.biloxi.com;branch=1234\r\n" +
		"Via: SIP/2.0/UDP server11.biloxi.com;branch=abcd\r\n" +
		"Max-Forwards: 70\r\n" +
		"To: <sip:bob@biloxi.com>\r\n" +
		"From: <sip:alice@atlanta.com>;tag=1928301774\r\n" +
		"Call-ID: a84b4c76e66710\r\n" +
		"CSeq: 314159 INVITE\r\n" +
		"Contact: <sip:alice@pc33.atlanta.com>,<sip:alice1@pc33.atlanta.com>\r\n" +
		"Content-Type: application/sdp\r\n" +
		"Content-Length: 0\r\n" +
		"\r\n")

	expected := &message.SIPMessage{
		Startline: message.Startline{
			Request: &message.Request{
				Method: "INVITE",
				RequestURI: &uri.SIPUri{
					Scheme:  "sip",
					User:    "bob",
					Domain:  "biloxi.com",
					Port:    -1,
					Opts:    map[string]string{},
					Headers: map[string]string{},
				},
			},
		},
		From: &fromto.SIPFromTo{
			Uri: &uri.SIPUri{
				Scheme:  "sip",
				User:    "alice",
				Domain:  "atlanta.com",
				Port:    -1,
				Opts:    map[string]string{},
				Headers: map[string]string{},
			},
			Tag:   "1928301774",
			Paras: map[string]string{},
		},
		To: &fromto.SIPFromTo{
			Uri: &uri.SIPUri{
				Scheme:  "sip",
				User:    "bob",
				Domain:  "biloxi.com",
				Port:    -1,
				Opts:    map[string]string{},
				Headers: map[string]string{},
			},
			Paras: map[string]string{},
		},
		CallID: "a84b4c76e66710",
		Contacts: []*contact.SIPContact{
			{
				DisName: "",
				Uri: &uri.SIPUri{
					Scheme:  "sip",
					User:    "alice",
					Domain:  "pc33.atlanta.com",
					Port:    -1,
					Opts:    map[string]string{},
					Headers: map[string]string{},
				},
				Paras: map[string]string{},
				Supported: []string{},
			},
			{
				DisName: "",
				Uri: &uri.SIPUri{
					Scheme:  "sip",
					User:    "alice1",
					Domain:  "pc33.atlanta.com",
					Port:    -1,
					Opts:    map[string]string{},
					Headers: map[string]string{},
				},
				Paras: map[string]string{},
				Supported: []string{},
			},
		},
		CSeq: &cseq.SIPCseq{
			Method: "INVITE",
			Seq:    314159,
		},
		TopmostVia: &via.SIPVia{
			Proto:  "UDP",
			Domain: "server10.biloxi.com",
			Branch: "1234",
			Opts:   map[string]string{},
		},

		Headers: map[string][]string{
			"via":            {"SIP/2.0/UDP server11.biloxi.com;branch=abcd"},
			"max-forwards":   {"70"},
			"content-type":   {"application/sdp"},
			"content-length": {"0"},
		},
		Body: []byte(""),
	}

	msg, err := message.Parse(data)
	if err != nil {
		t.Fatalf("Error parsing SIP request: %v", err)
	}

	if !reflect.DeepEqual(msg, expected) {
		t.Errorf("Parsed SIP request does not match expected result.\nGot: %+v\nExpected: %+v", msg.Contacts, expected.Contacts)
	}
}

// func TestParseResponse(t *testing.T) {
// 	data := []byte("SIP/2.0 200 OK\r\n" +
// 		"Via: SIP/2.0/UDP server10.biloxi.com\r\n" +
// 		"To: <sip:bob@biloxi.com>;tag=314159\r\n" +
// 		"From: <sip:alice@atlanta.com>;tag=1928301774\r\n" +
// 		"Call-ID: a84b4c76e66710\r\n" +
// 		"CSeq: 314159 INVITE\r\n" +
// 		"Contact: <sip:bob@biloxi.com>\r\n" +
// 		"Content-Length: 0\r\n" +
// 		"\r\n")

// 	expected := &message.SIPMessage{
// 		Startline: message.Startline{
// 			Response: &message.Response{
// 				StatusCode:   200,
// 				ReasonPhrase: "OK",
// 			},
// 		},
// 		Headers: map[string][]string{
// 			"via":            {"SIP/2.0/UDP server10.biloxi.com"},
// 			"to":             {"<sip:bob@biloxi.com>;tag=314159"},
// 			"from":           {"<sip:alice@atlanta.com>;tag=1928301774"},
// 			"call-id":        {"a84b4c76e66710"},
// 			"cseq":           {"314159 INVITE"},
// 			"contact":        {"<sip:bob@biloxi.com>"},
// 			"content-length": {"0"},
// 		},
// 		Body: []byte(""),
// 	}

// 	msg, err := message.Parse(data)
// 	if err != nil {
// 		t.Fatalf("Error parsing SIP response: %v", err)
// 	}

// 	if !reflect.DeepEqual(msg, expected) {
// 		t.Errorf("Parsed SIP response does not match expected result.\nGot: %+v\nExpected: %+v", msg, expected)
// 	}
// }
