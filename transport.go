package sip

import (
	"net"
)

type SIPTransport struct {
	Conn       net.Conn
	Protocol   string
	LocalAddr  string
	RemoteAddr string
}
