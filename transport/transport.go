package transport

import (
	"net"
)

type Transport struct {
	Sock *net.UDPConn
	RemoteAddr *net.UDPAddr
}