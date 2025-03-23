package core

import (
	"gossip/sipmess"
	"gossip/transaction"
	"gossip/transport"
	"math/rand"
	"net"
	"strconv"

	"github.com/rs/zerolog/log"
)

func GetMapSize() int {
	mu.Lock()
	defer mu.Unlock()
	return len(m)
}

func StatefullRoute(request *sipmess.SIPMessage, transp *transport.Transport) {
	strans_chan := make(chan *sipmess.SIPMessage, 3)
	ctrans_chan := make(chan *sipmess.SIPMessage, 3)

	strans_core_cb := func(transport *transport.Transport, message *sipmess.SIPMessage) {
		strans_chan <- message
	}

	ctrans_core_cb := func(transport *transport.Transport, message *sipmess.SIPMessage) {
		ctrans_chan <- message
	}

	trpt_cb := func(transport *transport.Transport, msg *sipmess.SIPMessage) bool {
		bin := msg.Serialize()
		if bin == nil {
			//serialize error
			return false
		}

		_, err := transport.Conn.WriteToUDP(bin, transport.RemoteAddr)
		if err != nil {
			//error transport error
			return false
		}

		return true
	}

	strans_term_cb := func(id transaction.TransID, reason transaction.TERM_REASON) {
		if reason != transaction.NORMAL {
			log.Error().Str("transaction_id", id.String()).Msg("Transaction terminated with error " + reason.String())
			strans_chan <- nil
		} else {
			log.Debug().Str("transaction_id", id.String()).Msg("Transaction terminated normally")
		}
		DeleteTransaction(id)
	}

	ctrans_term_cb := func(id transaction.TransID, reason transaction.TERM_REASON) {
		if reason != transaction.NORMAL {
			log.Error().Str("transaction_id", id.String()).Msg("Transaction terminated with error " + reason.String())
			ctrans_chan <- nil
		} else {
			log.Debug().Str("transaction_id", id.String()).Msg("Transaction terminated normally")
		}
		DeleteTransaction(id)
	}

	server_trans := StartServerTransaction(request, transp, strans_core_cb, trpt_cb, strans_term_cb)

	request = <-strans_chan

	to_uri := request.To.Uri
	address := net.JoinHostPort(string(to_uri.Domain), strconv.Itoa(to_uri.Port))
	dest_addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return
	}
	dest_transp := &transport.Transport{
		Protocol:   "UDP",
		Conn:       transp.Conn,
		LocalAddr:  transp.LocalAddr,
		RemoteAddr: dest_addr,
	}

	request.AddVia(sipmess.SIPVia{
		Proto:  []byte("UDP"),
		Domain: []byte(dest_transp.LocalAddr.IP.String()),
		Port:   dest_transp.LocalAddr.Port,
		Branch: randSeq(5),
	})

	StartClientTransaction(request, dest_transp, ctrans_core_cb, trpt_cb, ctrans_term_cb)

	for {
		select {
		case result := <-strans_chan:
			if result == nil {
				return
			}
		case response := <-ctrans_chan:
			if response == nil {
				log.Error().Msg("Error in client transaction")
				return
			}

			log.Debug().Msg("Forward response to server transaction")

			response.DeleteVia()
			server_trans.Event(response)

			status := response.Response.StatusCode
			if status >= 200 {
				return
			}
		}
	}
}

func StatelessRoute(request *sipmess.SIPMessage, transp *transport.Transport) {
	if request.Request == nil {
		return
	}

	to_uri := request.To.Uri
	address := net.JoinHostPort(string(to_uri.Domain), strconv.Itoa(to_uri.Port))
	DestAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return
	}
	DestTransport := transport.Transport{
		Protocol:   "UDP",
		Conn:       transp.Conn,
		LocalAddr:  transp.LocalAddr,
		RemoteAddr: DestAddr,
	}

	bin := request.Serialize()
	if bin == nil {
		log.Error().Msg("Serialize error")
		return
	}

	_, err = DestTransport.Conn.WriteToUDP(bin, DestTransport.RemoteAddr)
	if err != nil {
		log.Error().Msg("Transport error")
	}
}

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randSeq(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return b
}
