package core

import (
	"gossip/message"
	"gossip/transaction"
	"gossip/util"
)

// type state int

// const (
// 	wait_for_request state = iota
// 	wait_for_response
// 	terminated
// )


// type proxy struct {
// 	strans transaction.Transaction
// 	strans_chan chan util.Event
// 	ctrans transaction.Transaction
// 	ctrans_chan chan util.Event   
// 	state 
// }

// func Make(request *message.SIPMessage) *proxy {
// 	strans_chan := make(chan util.Event, 3)
// 	ctrans_chan := make(chan util.Event, 3)

// 	core_cb := func(from transaction.Transaction, ev util.Event) {
// 		strans_chan <- ev
// 	}

// 	server_trans: StartServerTransaction(request, core_cb, trpt_cb)
	   
// 	return &proxy{
// 		strans_chan: strans_chan,
// 		ctrans_chan: ctrans_chan,
// 		strans: server_trans
// 	}
// }

// func (prx *proxy) Start() {
// 	defer prx.ch.Close()

// 	for {
// 		select {
// 		case ev := <- prx.strans_chan:
// 			prx.handle_uac(ev)
// 		case ev := <- prx.ctrans_chan:
// 			prx.handle_uas(ev)
// 		}

// 		if prx.state == terminated {
// 			return
// 		}
// 	}
// }

// func (prx *proxy) handle_uac(ev util.Event) {
// 	Type = ev.Type

// 	if (Type == util.Mess && prx.state = wait_for_request) {
// 		msg, ok := ev.Data.(*message.SIPMessage)
// 		if !ok {
// 			return
// 		}

// 		core_cb := func(from transaction.Transaction, ev util.Event) {
// 			prx.ctrans_chan <- ev
// 		}

// 		prx.ctrnas = StartClientTransaction(request, core_cb, trpt_cb)
// 	}

// }

func statefull_route(request *message.SIPMessage) {
	strans_chan := make(chan util.Event, 3)
	ctrans_chan := make(chan util.Event, 3)

	strans_cb := func(from transaction.Transaction, ev util.Event) {
		strans_chan <- ev
	}

	ctrans_cb := func(from transaction.Transaction, ev util.Event) {
		ctrans_chan <- ev
	}

	server_trans: StartServerTransaction(request, core_cb, trpt_cb)

	ev <- strans_chan:
	request, ok := ev.Data.(*message.SIPMessage)
	if !ok {
		return
	}

	client_trans: StartServerTransaction(request, core_cb, trpt_cb)

	for {
	case ev := <- ctrans_chan: 
		if (ev.Type == util.MESS) {
			response, ok := ev.Data.(*message.SIPMessage)
			if !ok {
				continue
			}
			server_trans.Event(ev)

			status := response.Response.StatusCode 
			if (status >= 200 && status < 300) {
				return
			}
		} else if (ev.Type == util.ERROR) {
			return
		}
	case ev := <- strans_chan:
		if (ev.Type == util.ERROR)  {
			return
		}
	}
}

func stateless_route(request *message.SIPMessage) {
	
}

func trpt_cb (from transaction.Transaction, ev util.Event) {
	msg, ok := ev.Data.(*message.SIPMessage)
	if !ok {
		return
	}

	trprt := msg.Transport
	bin, err := message.Serialize(msg)
	if (err != nil) {
		//serialize error
		return
	}
	
	 _, err := trprt.Socket.WriteToUDP(bin, trprt.RemoteAddr)
	if (err != nil) {
		//error transport error
		from.Event(util.Event{Type: util.ERROR, Data: msg)
	}
	return
}



