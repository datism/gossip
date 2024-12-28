package transport

import (
	"net"
)

type Transport struct {
	Protocol string
	Socket *net.UDPConn
	LocalAddr *net.UDPAddr
	RemoteAddr *net.UDPAddr
}


