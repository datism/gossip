package transport

import (
	"net"
)

type Transport struct {
	Protocol   string
	Conn       *net.UDPConn
	LocalAddr  *net.UDPAddr
	RemoteAddr *net.UDPAddr
}

func (t Transport) DeepCopy() *Transport {
	// Deep copy LocalAddr
	var newLocalAddr *net.UDPAddr
	if t.LocalAddr != nil {
		newLocalAddr = &net.UDPAddr{
			IP:   append([]byte{}, t.LocalAddr.IP...), // Copy IP slice
			Port: t.LocalAddr.Port,
			Zone: t.LocalAddr.Zone,
		}
	}

	// Deep copy RemoteAddr
	var newRemoteAddr *net.UDPAddr
	if t.RemoteAddr != nil {
		newRemoteAddr = &net.UDPAddr{
			IP:   append([]byte{}, t.RemoteAddr.IP...), // Copy IP slice
			Port: t.RemoteAddr.Port,
			Zone: t.RemoteAddr.Zone,
		}
	}

	// Since net.UDPConn cannot be deep copied (it's bound to a specific OS resource),
	// we will only copy the reference. You may need to recreate a new connection
	// if deep cloning is strictly necessary.

	return &Transport{
		Protocol:   t.Protocol,
		Conn:       t.Conn,        // Shallow copy of the connection
		LocalAddr:  newLocalAddr,  // Deep copied LocalAddr
		RemoteAddr: newRemoteAddr, // Deep copied RemoteAddr
	}
}
