package sip

import (
	"bytes"
	"reflect"
	"testing"
)

func TestParseSIPMessage(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *SIPMessage
		wantErr bool
	}{
		{
			name: "Parse simple INVITE request",
			input: "INVITE sip:bob@example.com SIP/2.0\r\n" +
				"Via: SIP/2.0/UDP 192.168.1.1:5060;branch=z9hG4bK776asdhds\r\n" +
				"From: Alice <sip:alice@example.com>;tag=1928301774\r\n" +
				"To: Bob <sip:bob@example.com>\r\n" +
				"Call-ID: a84b4c76e66710@pc33.example.com\r\n" +
				"CSeq: 314159 INVITE\r\n" +
				"Contact: <sip:alice@192.168.1.1>\r\n" +
				"Content-Type: application/sdp\r\n" +
				"Content-Length: 13\r\n" +
				"\r\n" +
				"Test SDP body",
			want: &SIPMessage{
				Startline: Startline{
					Request: &Request{
						Method:     Invite,
						RequestURI: SIPUri{Scheme: []byte("sip"), User: []byte("bob"), Port: -1, Domain: []byte("example.com")},
					},
				},
				TopmostVia: SIPVia{
					Tranport: "udp",
					Domain:   []byte("192.168.1.1"),
					Port:     5060,
					Branch:   []byte("z9hG4bK776asdhds"),
				},
				From: SIPFromTo{
					Uri: SIPUri{Scheme: []byte("sip"), User: []byte("alice"), Domain: []byte("example.com"), Port: -1},
					Tag: []byte("1928301774"),
				},
				To: SIPFromTo{
					Uri: SIPUri{Scheme: []byte("sip"), User: []byte("bob"), Domain: []byte("example.com"), Port: -1},
				},
				CallID: []byte("a84b4c76e66710@pc33.example.com"),
				CSeq: SIPCseq{
					Seq:    314159,
					Method: Invite,
				},
				Contacts: []SIPContact{
					{
						Uri: SIPUri{Scheme: []byte("sip"), User: []byte("alice"), Domain: []byte("192.168.1.1"), Port: -1},
					},
				},
				Headers: map[SIPHeader][][]byte{
					ContentType:   {[]byte("application/sdp")},
					ContentLength: {[]byte("13")},
				},
				Body: []byte("Test SDP body"),
				Options: ParseOptions{
					ParseTopMostVia: true,
					ParseFrom:       true,
					ParseTo:         true,
					ParseCallID:     true,
					ParseCseq:       true,
					ParseContacts:   true,
				},
			},
			wantErr: false,
		},
		{
			name: "Parse simple SIP response",
			input: "SIP/2.0 200 OK\r\n" +
				"Via: SIP/2.0/UDP 192.168.1.1:5060;branch=z9hG4bK776asdhds\r\n" +
				"From: Alice <sip:alice@example.com>;tag=1928301774\r\n" +
				"To: Bob <sip:bob@example.com>;tag=a6c85cf\r\n" +
				"Call-ID: a84b4c76e66710@pc33.example.com\r\n" +
				"CSeq: 314159 INVITE\r\n" +
				"Contact: <sip:bob@192.168.1.2>\r\n" +
				"Content-Length: 0\r\n" +
				"\r\n",
			want: &SIPMessage{
				Startline: Startline{
					Response: &Response{
						StatusCode:   200,
						ReasonPhrase: []byte("OK"),
					},
				},
				TopmostVia: SIPVia{
					Tranport: "udp",
					Domain:   []byte("192.168.1.1"),
					Port:     5060,
					Branch:   []byte("z9hG4bK776asdhds"),
				},
				From: SIPFromTo{
					Uri: SIPUri{Scheme: []byte("sip"), User: []byte("alice"), Domain: []byte("example.com"), Port: -1},
					Tag: []byte("1928301774"),
				},
				To: SIPFromTo{
					Uri: SIPUri{Scheme: []byte("sip"), User: []byte("bob"), Domain: []byte("example.com"), Port: -1},
					Tag: []byte("a6c85cf"),
				},
				CallID: []byte("a84b4c76e66710@pc33.example.com"),
				CSeq: SIPCseq{
					Seq:    314159,
					Method: Invite,
				},
				Contacts: []SIPContact{
					{
						Uri: SIPUri{Scheme: []byte("sip"), User: []byte("bob"), Domain: []byte("192.168.1.2"), Port: -1},
					},
				},
				Headers: map[SIPHeader][][]byte{
					ContentLength: {[]byte("0")},
				},
				Body: []byte(""),
				Options: ParseOptions{
					ParseTopMostVia: true,
					ParseFrom:       true,
					ParseTo:         true,
					ParseCallID:     true,
					ParseCseq:       true,
					ParseContacts:   true,
				},
			},
			wantErr: false,
		},
		{
			name:    "Parse invalid message",
			input:   "This is not a SIP message",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSipMessage([]byte(tt.input), ParseOptions{
				ParseTopMostVia: true,
				ParseFrom:       true,
				ParseTo:         true,
				ParseCallID:     true,
				ParseCseq:       true,
				ParseContacts:   true,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSIPMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			// Compare specific fields for more readable test failures
			if !reflect.DeepEqual(got.Request, tt.want.Request) {
				t.Errorf("Request mismatch, got = %v, want %v", got.Request, tt.want.Request)
			}
			if !reflect.DeepEqual(got.Response, tt.want.Response) {
				t.Errorf("Response mismatch, got = %v, want %v", got.Response, tt.want.Response)
			}
			if !reflect.DeepEqual(got.TopmostVia, tt.want.TopmostVia) {
				t.Errorf("TopmostVia mismatch, got = %#v, want %#v", got.TopmostVia, tt.want.TopmostVia)
			}
			if !reflect.DeepEqual(got.From, tt.want.From) {
				t.Errorf("From mismatch, got = %v, want %v", got.From, tt.want.From)
			}
			if !reflect.DeepEqual(got.To, tt.want.To) {
				t.Errorf("To mismatch, got = %v, want %v", got.To, tt.want.To)
			}
			if !bytes.Equal(got.CallID, tt.want.CallID) {
				t.Errorf("CallID mismatch, got = %s, want %s", got.CallID, tt.want.CallID)
			}
			if !reflect.DeepEqual(got.CSeq, tt.want.CSeq) {
				t.Errorf("CSeq mismatch, got = %v, want %v", got.CSeq, tt.want.CSeq)
			}
			if !reflect.DeepEqual(got.Contacts, tt.want.Contacts) {
				t.Errorf("Contacts mismatch, got = %v, want %v", got.Contacts, tt.want.Contacts)
			}
			if !bytes.Equal(got.Body, tt.want.Body) {
				t.Errorf("Body mismatch, got = %s, want %s", got.Body, tt.want.Body)
			}
		})
	}
}
