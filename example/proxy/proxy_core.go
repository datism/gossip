package main

import (
	"math/rand"
	"net"
	"strconv"

	"github.com/datism/sip"
	"github.com/rs/zerolog/log"
)

func GetMapSize() int {
	mu.Lock()
	defer mu.Unlock()
	return len(m)
}

func StatefullRoute(request *sip.SIPMessage, transp *sip.SIPTransport) {
	strans_chan := make(chan *sip.SIPMessage, 3)
	ctrans_chan := make(chan *sip.SIPMessage, 3)

	strans_core_cb := func(transport *sip.SIPTransport, message *sip.SIPMessage) {
		strans_chan <- message
	}

	ctrans_core_cb := func(transport *sip.SIPTransport, message *sip.SIPMessage) {
		ctrans_chan <- message
	}

	trpt_cb := func(transport *sip.SIPTransport, msg *sip.SIPMessage) bool {
		bin := msg.Serialize()
		if bin == nil {
			//serialize
			log.Error().Msg("Error serialize sip message")
			return false
		}

		udpConn, ok := transport.Conn.(*net.UDPConn)
		if !ok {
			log.Error().Msg("Error transport type")
			return false
		}

		daddr, err := net.ResolveUDPAddr("udp", transport.RemoteAddr)
		if err != nil {
			log.Error().Err(err).Msg("Failed to resolve UDP address")
			return false
		}

		_, err = udpConn.WriteTo(bin, daddr)
		if err != nil {
			log.Error().Err(err).Msg("Failed to write to UDP connection")
			return false
		}

		return true
	}

	strans_term_cb := func(id sip.TransID, reason sip.TERM_REASON) {
		if reason != sip.NORMAL {
			log.Error().Str("siptrans_id", id.String()).Msg("sip terminated with error " + reason.String())
			strans_chan <- nil
		} else {
			log.Debug().Str("siptrans_id", id.String()).Msg("sip terminated normally")
		}
		DeleteTrans(id)
	}

	ctrans_term_cb := func(id sip.TransID, reason sip.TERM_REASON) {
		if reason != sip.NORMAL {
			log.Error().Str("siptrans_id", id.String()).Msg("sip terminated with error " + reason.String())
			ctrans_chan <- nil
		} else {
			log.Debug().Str("siptrans_id", id.String()).Msg("sip terminated normally")
		}
		DeleteTrans(id)
	}

	server_trans := StartServerTrans(request, transp, strans_core_cb, trpt_cb, strans_term_cb)

	request = <-strans_chan

	to_uri := request.To.Uri
	dest := net.JoinHostPort(string(to_uri.Domain), strconv.Itoa(to_uri.Port))
	dest_transp := &sip.SIPTransport{
		Protocol:   "udp",
		Conn:       transp.Conn,
		LocalAddr:  transp.LocalAddr,
		RemoteAddr: dest,
	}

	request.AddVia(sip.SIPVia{
		Tranport: "udp",
		Domain:   []byte(to_uri.Domain),
		Port:     to_uri.Port,
		Branch:   randSeq(5),
	})

	StartClientTrans(request, dest_transp, ctrans_core_cb, trpt_cb, ctrans_term_cb)

	for {
		select {
		case result := <-strans_chan:
			if result == nil {
				return
			}
		case response := <-ctrans_chan:
			if response == nil {
				log.Error().Msg("Error in client sip")
				return
			}

			log.Debug().Msg("Forward response to server sip")

			response.DeleteVia()
			server_trans.Event(response)

			status := response.Response.StatusCode
			if status >= 200 {
				return
			}
		}
	}
}

func StatelessRoute(request *sip.SIPMessage, transp *sip.SIPTransport) {
	if request.Request == nil {
		return
	}

	to_uri := request.To.Uri
	dest := net.JoinHostPort(string(to_uri.Domain), strconv.Itoa(to_uri.Port))
	daddr, err := net.ResolveUDPAddr("udp", dest)
	if err != nil {
		log.Error().Err(err).Msg("Failed to resolve UDP address")
		return
	}
	// conn, err := net.DialUDP("udp", nil, daddr)
	// if err != nil {
	// 	log.Error().Err(err).Msg("Failed to dial UDP address")
	// 	return
	// }

	bin := request.Serialize()
	if bin == nil {
		log.Error().Msg("Serialize error")
		return
	}

	// _, err = tr.Write(bin)
	// if err != nil {
	// 	log.Error().Err(err).Msg("Failed to write to UDP connection")
	// 	return
	// }

	udpConn, ok := transp.Conn.(*net.UDPConn)
	if !ok {
		log.Error().Msg("Error transport type")
		return
	}

	_, err = udpConn.WriteTo(bin, daddr)
	if err != nil {
		log.Error().Err(err).Msg("Failed to write to UDP connection")
		return
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
