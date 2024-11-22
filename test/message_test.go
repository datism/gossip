package test

import (
	"reflect"
	"testing"

	"gossip/message"
)

func TestParseRequest(t *testing.T) {
	data := []byte("INVITE sip:bob@biloxi.com SIP/2.0\r\n" +
		"Via: SIP/2.0/UDP server10.biloxi.com\r\n" +
		"Via: SIP/2.0/UDP server11.biloxi.com\r\n" +
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
				Method:     "INVITE",
				RequestURI: "sip:bob@biloxi.com",
			},
		},
		Headers: map[string][]string{
			"via":            {"SIP/2.0/UDP server10.biloxi.com", "SIP/2.0/UDP server11.biloxi.com"},
			"max-forwards":   {"70"},
			"to":             {"<sip:bob@biloxi.com>"},
			"from":           {"<sip:alice@atlanta.com>;tag=1928301774"},
			"call-id":        {"a84b4c76e66710"},
			"cseq":           {"314159 INVITE"},
			"contact":        {"<sip:alice@pc33.atlanta.com>", "<sip:alice1@pc33.atlanta.com>"},
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
		t.Errorf("Parsed SIP request does not match expected result.\nGot: %+v\nExpected: %+v", msg, expected)
	}
}

func TestParseResponse(t *testing.T) {
	data := []byte("SIP/2.0 200 OK\r\n" +
		"Via: SIP/2.0/UDP server10.biloxi.com\r\n" +
		"To: <sip:bob@biloxi.com>;tag=314159\r\n" +
		"From: <sip:alice@atlanta.com>;tag=1928301774\r\n" +
		"Call-ID: a84b4c76e66710\r\n" +
		"CSeq: 314159 INVITE\r\n" +
		"Contact: <sip:bob@biloxi.com>\r\n" +
		"Content-Length: 0\r\n" +
		"\r\n")

	expected := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode:   200,
				ReasonPhrase: "OK",
			},
		},
		Headers: map[string][]string{
			"via":            {"SIP/2.0/UDP server10.biloxi.com"},
			"to":             {"<sip:bob@biloxi.com>;tag=314159"},
			"from":           {"<sip:alice@atlanta.com>;tag=1928301774"},
			"call-id":        {"a84b4c76e66710"},
			"cseq":           {"314159 INVITE"},
			"contact":        {"<sip:bob@biloxi.com>"},
			"content-length": {"0"},
		},
		Body: []byte(""),
	}

	msg, err := message.Parse(data)
	if err != nil {
		t.Fatalf("Error parsing SIP response: %v", err)
	}

	if !reflect.DeepEqual(msg, expected) {
		t.Errorf("Parsed SIP response does not match expected result.\nGot: %+v\nExpected: %+v", msg, expected)
	}
}
