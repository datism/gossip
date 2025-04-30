package sip

import (
	"net"
)

type Transport struct {
	Conn       net.Conn
	Protocol   string
	LocalAddr  string
	RemoteAddr string
}
