package core

import (
	"gossip/message"
	"gossip/message/via"
	"gossip/transaction"
	"gossip/transport"
	"math/rand"
	"net"
	"strconv"

	"github.com/rs/zerolog/log"
)

func Statefull_route(request *message.SIPMessage, transp *transport.Transport) {
	strans_chan := make(chan *message.SIPMessage, 3)
	ctrans_chan := make(chan *message.SIPMessage, 3)

	term_cb := func(id transaction.TransID, reason transaction.TERM_REASON) {
		if reason != transaction.NORMAL {
			log.Error().Str("transaction_id", id.String()).Msg("Transaction terminated with error " + reason.String())
		} else {
			log.Debug().Str("transaction_id", id.String()).Msg("Transaction terminated normally")
		}
		DeleteTransaction(id)
	}

	trpt_cb := func(transport *transport.Transport, msg *message.SIPMessage) bool {
		bin := message.Serialize(msg)
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

	strans_cb := func(transport *transport.Transport, message *message.SIPMessage) {
		strans_chan <- message
	}

	ctrans_cb := func(transport *transport.Transport, message *message.SIPMessage) {
		ctrans_chan <- message
	}

	server_trans := StartServerTransaction(request, transp, strans_cb, trpt_cb, term_cb)

	request = <-strans_chan

	to_uri := request.To.Uri
	address := net.JoinHostPort(to_uri.Domain, strconv.Itoa(to_uri.Port))
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

	request.AddVia(&via.SIPVia{
		Proto:  "UDP",
		Domain: dest_transp.LocalAddr.IP.String(),
		Port:   dest_transp.LocalAddr.Port,
		Branch: randSeq(5),
	})

	StartClientTransaction(request, dest_transp, ctrans_cb, trpt_cb, term_cb)

	for {
		response := <-ctrans_chan

		log.Debug().Msg("Forward response to server transaction")

		if len(response.Headers["via"]) == 0 {
			log.Error().Str("transaction_id", "ehh").Interface("handle_message", response).Msg("No via header in response")
		}

		response.RemoveVia()
		server_trans.Event(response)

		status := response.Response.StatusCode
		if status >= 200 {
			return
		}
	}
}

func Stateless_route(request *message.SIPMessage, transp *transport.Transport) {
	if request.Request == nil {
		return
	}

	to_uri := request.To.Uri
	address := net.JoinHostPort(to_uri.Domain, strconv.Itoa(to_uri.Port))
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

	bin := message.Serialize(request)
	if bin == nil {
		log.Error().Msg("Serialize error")
		return
	}

	_, err = DestTransport.Conn.WriteToUDP(bin, DestTransport.RemoteAddr)
	if err != nil {
		log.Error().Msg("Transport error")
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
