package siptransp

import (
	"net"
)

type Transport struct {
	Protocol   string
	Conn       *net.UDPConn
	LocalAddr  *net.UDPAddr
	RemoteAddr *net.UDPAddr
}
