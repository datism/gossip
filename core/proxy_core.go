package core

import (
	"gossip/message"
	"gossip/message/via"
	"gossip/transaction"
	"gossip/transport"
	"gossip/util"
	"math/rand"
	"net"
	"strconv"

	"github.com/rs/zerolog/log"
)

func Statefull_route(request *message.SIPMessage) {
	strans_chan := make(chan util.Event, 3)
	ctrans_chan := make(chan util.Event, 3)

	strans_cb := func(from transaction.Transaction, ev util.Event) {
		if ev.Type == util.ERROR || ev.Type == util.TERMINATED {
			id, ok := ev.Data.(transaction.TransID)
			if ok {
				DeleteTransaction(id)
			}
		} else {
			strans_chan <- ev
		}

	}

	ctrans_cb := func(from transaction.Transaction, ev util.Event) {
		if ev.Type == util.ERROR || ev.Type == util.TERMINATED {
			id, ok := ev.Data.(transaction.TransID)
			if ok {
				DeleteTransaction(id)
			}
		} else {
			ctrans_chan <- ev

		}
	}

	server_trans := StartServerTransaction(request, strans_cb, trpt_cb)

	ev := <-strans_chan
	request, ok := ev.Data.(*message.SIPMessage)
	if !ok {
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
		Conn:       request.Transport.Conn,
		LocalAddr:  request.Transport.LocalAddr,
		RemoteAddr: DestAddr,
	}

	request.AddVia(&via.SIPVia{
		Proto:  "UDP",
		Domain: DestTransport.LocalAddr.IP.String(),
		Port:   DestTransport.LocalAddr.Port,
		Branch: randSeq(5),
	})
	request.Transport = &DestTransport

	_ = StartClientTransaction(request, ctrans_cb, trpt_cb)

	for {
		select {
		case ev := <-ctrans_chan:
			if ev.Type == util.MESSAGE {
				response, ok := ev.Data.(*message.SIPMessage)
				if !ok {
					continue
				}

				log.Debug().Msg("Forward response to server transaction")
				response.RemoveVia()
				server_trans.Event(util.Event{Type: util.MESSAGE, Data: response})

				status := response.Response.StatusCode
				if status >= 200 {
					return
				}
			} else if ev.Type == util.ERROR {
				return
			}
		case ev := <-strans_chan:
			if ev.Type == util.ERROR {
				return
			}
		}
	}
}

func Stateless_route(request *message.SIPMessage) {
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
		Conn:       request.Transport.Conn,
		LocalAddr:  request.Transport.LocalAddr,
		RemoteAddr: DestAddr,
	}

	bin := message.Serialize(request)
	if bin == nil {
		//serialize error
		return
	}

	_, err = DestTransport.Conn.WriteToUDP(bin, DestTransport.RemoteAddr)
	if err != nil {
		log.Error().Msg("Transport error")
	}
}

func trpt_cb(from transaction.Transaction, ev util.Event) {
	msg, ok := ev.Data.(*message.SIPMessage)
	if !ok {
		return
	}

	trprt := msg.Transport
	bin := message.Serialize(msg)
	if bin == nil {
		//serialize error
		return
	}

	_, err := trprt.Conn.WriteToUDP(bin, trprt.RemoteAddr)
	if err != nil {
		//error transport error
		log.Error().Msg("Transport error")
		from.Event(util.Event{Type: util.ERROR, Data: msg})
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
