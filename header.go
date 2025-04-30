package sip

import (
	"bytes"
	"fmt"
)

type SIPHeader int

const (
	From SIPHeader = iota
	To
	CSeq
	CallID
	MaxForwards
	Via
	RecordRoute
	Contact
	Expires
	ContentLength
	ContentType
	ContentDisposition
	ContentEncoding
	Authorization
	ProxyAuthorization
	WWWAuthenticate
	ProxyAuthenticate
	Route
	Allow
	AllowEvents
	Event
	Require
	ProxyRequire
	Unsupported
	Supported
	Warning
	MinExpires
	Organization
	UserAgent
	Server
	Subject
	Reason
	Accept
	AcceptEncoding
	AcceptLanguage
	AcceptContact
	AcceptResourcePriority
	AlertInfo
	AuthenticationInfo
	Date
	Priority
	SessionExpires
	PAssertedIdentity
	PPreferredIdentity
	Privacy
	ReferredBy
	Replaces
	Join
	SubscriptionState
	Identity
	IdentityInfo
	HistoryInfo
	Diversion
	SessionID
)

var sipHeaderNames = map[SIPHeader][]byte{
	From:                   []byte("From"),
	To:                     []byte("To"),
	CSeq:                   []byte("CSeq"),
	CallID:                 []byte("Call-ID"),
	MaxForwards:            []byte("Max-Forwards"),
	Via:                    []byte("Via"),
	RecordRoute:            []byte("Record-Route"),
	Contact:                []byte("Contact"),
	Expires:                []byte("Expires"),
	ContentLength:          []byte("Content-Length"),
	ContentType:            []byte("Content-Type"),
	ContentDisposition:     []byte("Content-Disposition"),
	ContentEncoding:        []byte("Content-Encoding"),
	Authorization:          []byte("Authorization"),
	ProxyAuthorization:     []byte("Proxy-Authorization"),
	WWWAuthenticate:        []byte("WWW-Authenticate"),
	ProxyAuthenticate:      []byte("Proxy-Authenticate"),
	Route:                  []byte("Route"),
	Allow:                  []byte("Allow"),
	AllowEvents:            []byte("Allow-Events"),
	Event:                  []byte("Event"),
	Require:                []byte("Require"),
	ProxyRequire:           []byte("Proxy-Require"),
	Unsupported:            []byte("Unsupported"),
	Supported:              []byte("Supported"),
	Warning:                []byte("Warning"),
	MinExpires:             []byte("Min-Expires"),
	Organization:           []byte("Organization"),
	UserAgent:              []byte("User-Agent"),
	Server:                 []byte("Server"),
	Subject:                []byte("Subject"),
	Reason:                 []byte("Reason"),
	Accept:                 []byte("Accept"),
	AcceptEncoding:         []byte("Accept-Encoding"),
	AcceptLanguage:         []byte("Accept-Language"),
	AcceptContact:          []byte("Accept-Contact"),
	AcceptResourcePriority: []byte("Accept-Resource-Priority"),
	AlertInfo:              []byte("Alert-Info"),
	AuthenticationInfo:     []byte("Authentication-Info"),
	Date:                   []byte("Date"),
	Priority:               []byte("Priority"),
	SessionExpires:         []byte("Session-Expires"),
	PAssertedIdentity:      []byte("P-Asserted-Identity"),
	PPreferredIdentity:     []byte("P-Preferred-Identity"),
	Privacy:                []byte("Privacy"),
	ReferredBy:             []byte("Referred-By"),
	Replaces:               []byte("Replaces"),
	Join:                   []byte("Join"),
	SubscriptionState:      []byte("Subscription-State"),
	Identity:               []byte("Identity"),
	IdentityInfo:           []byte("Identity-Info"),
	HistoryInfo:            []byte("History-Info"),
	Diversion:              []byte("Diversion"),
	SessionID:              []byte("Session-ID"),
}

// SIPHeaderName returns the name of a SIP header.
func SerializeHeaderName(header SIPHeader) []byte {
	return sipHeaderNames[header]
}

var nameSipHeaders = map[string]SIPHeader{
	"from":                     From,
	"to":                       To,
	"cseq":                     CSeq,
	"call-id":                  CallID,
	"max-forwards":             MaxForwards,
	"via":                      Via,
	"record-route":             RecordRoute,
	"contact":                  Contact,
	"expires":                  Expires,
	"content-length":           ContentLength,
	"content-type":             ContentType,
	"content-disposition":      ContentDisposition,
	"content-encoding":         ContentEncoding,
	"authorization":            Authorization,
	"proxy-authorization":      ProxyAuthorization,
	"www-authenticate":         WWWAuthenticate,
	"proxy-authenticate":       ProxyAuthenticate,
	"route":                    Route,
	"allow":                    Allow,
	"allow-events":             AllowEvents,
	"event":                    Event,
	"require":                  Require,
	"proxy-require":            ProxyRequire,
	"unsupported":              Unsupported,
	"supported":                Supported,
	"warning":                  Warning,
	"min-expires":              MinExpires,
	"organization":             Organization,
	"user-agent":               UserAgent,
	"server":                   Server,
	"subject":                  Subject,
	"reason":                   Reason,
	"accept":                   Accept,
	"accept-encoding":          AcceptEncoding,
	"accept-language":          AcceptLanguage,
	"accept-contact":           AcceptContact,
	"accept-resource-priority": AcceptResourcePriority,
	"alert-info":               AlertInfo,
	"authentication-info":      AuthenticationInfo,
	"date":                     Date,
	"priority":                 Priority,
	"session-expires":          SessionExpires,
	"p-asserted-identity":      PAssertedIdentity,
	"p-preferred-identity":     PPreferredIdentity,
	"privacy":                  Privacy,
	"referred-by":              ReferredBy,
	"replaces":                 Replaces,
	"join":                     Join,
	"subscription-state":       SubscriptionState,
	"identity":                 Identity,
	"identity-info":            IdentityInfo,
	"history-info":             HistoryInfo,
	"diversion":                Diversion,
	"session-id":               SessionID,
}

// ParseHader parses a header name and returns the corresponding SIPHeader.
func ParseHeaderName(header []byte) (SIPHeader, error) {
	if h, ok := nameSipHeaders[string(bytes.ToLower(header))]; ok {
		return h, nil
	}
	return 0, fmt.Errorf("unrecognized header %q", header)
}
