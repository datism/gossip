package sipmess

import (
	"bytes"
	"fmt"
)

// SIPMethod represents a SIP request method.
type SIPMethod int

const (
	Invite SIPMethod = iota
	Ack
	Bye
	Cancel
	Options
	Register
	Prack
	Subscribe
	Notify
	Publish
	Info
	Refer
	Message
	Update
)

// Precomputed map for SIP method names.
var sipMethodNames = map[SIPMethod][]byte{
	Invite:    []byte("INVITE"),
	Ack:       []byte("ACK"),
	Bye:       []byte("BYE"),
	Cancel:    []byte("CANCEL"),
	Options:   []byte("OPTIONS"),
	Register:  []byte("REGISTER"),
	Prack:     []byte("PRACK"),
	Subscribe: []byte("SUBSCRIBE"),
	Notify:    []byte("NOTIFY"),
	Publish:   []byte("PUBLISH"),
	Info:      []byte("INFO"),
	Refer:     []byte("REFER"),
	Message:   []byte("MESSAGE"),
	Update:    []byte("UPDATE"),
}

// SIPMethodName returns the name of a SIP method.
func SerializeMethod(method SIPMethod) []byte {
	return sipMethodNames[method]
}

var nameSipMethods = map[string]SIPMethod{
	"INVITE":    Invite,
	"ACK":       Ack,
	"BYE":       Bye,
	"CANCEL":    Cancel,
	"OPTIONS":   Options,
	"REGISTER":  Register,
	"PRACK":     Prack,
	"SUBSCRIBE": Subscribe,
	"NOTIFY":    Notify,
	"PUBLISH":   Publish,
	"INFO":      Info,
	"REFER":     Refer,
	"MESSAGE":   Message,
	"UPDATE":    Update,
}

// ParseMethod parses a SIP method from a byte slice.
func ParseMethod(method []byte) (SIPMethod, error) {
	if m, ok := nameSipMethods[string(bytes.ToUpper(method))]; ok {
		return m, nil
	}
	return -1, fmt.Errorf("invalid SIP method %q", method)
}
