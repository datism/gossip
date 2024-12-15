package transport

import (
	"net"
	"gossip/message"
)

type Transport struct {
	Sock *net.UDPConn
	RemoteAddr *net.UDPAddr
}

func Send(msg *message.SIPMessage) {

}