package transport

import (
	"net"
	"gossip/message"
)

type Transport struct {
	Protocol string
	Socket *net.UDPConn
	LocalAddr *net.UDPAddr
	RemoteAddr *net.UDPAddr
}


func Send(trans *Transport, msg *message.SIPMessage) {

}
