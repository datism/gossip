package sip

// Sitrans represents the state machine for an INVITE server transaction
type Sitrans struct {
	id        TransID                               // Transaction ID
	state     state                                 // Current state of the transaction
	message   *SIPMessage                           // The SIP message associated with the transaction
	transport *SIPTransport                         // Transport layer for sending and receiving messages
	last_res  *SIPMessage                           // The last response received
	timerprv  *transTimer                           // Timer for provisional responses
	timerg    *transTimer                           // Timer G for retransmissions
	timerh    *transTimer                           // Timer H for timeouts
	timeri    *transTimer                           // Timer I for termination
	transc    chan *SIPMessage                      // Channel for receiving events like timeouts or messages
	trpt_cb   func(*SIPTransport, *SIPMessage) bool // Transport callback
	core_cb   func(*SIPTransport, *SIPMessage)      // Core callback
	term_cb   func(TransID, TERM_REASON)            // Termination callback
}

// MakeIST creates and initializes a new Sitrans instance with the given message and callbacks
func MakeIST(
	id TransID,
	msg *SIPMessage,
	transport *SIPTransport,
	core_callback func(*SIPTransport, *SIPMessage),
	transport_callback func(*SIPTransport, *SIPMessage) bool,
	term_callback func(TransID, TERM_REASON),
) *Sitrans {
	//log.Trace().Str("transaction_id", id.String()).Interface("message", msg).Interface("transport", transport).Msg("Creating new INVITE server transaction")
	return &Sitrans{
		id:        id,
		message:   msg,
		transport: transport,
		transc:    make(chan *SIPMessage, 5),
		timerprv:  newTransTimer("timer prv"),
		timerg:    newTransTimer("timer g"),
		timerh:    newTransTimer("timer h"),
		timeri:    newTransTimer("timer i"),
		state:     proceeding,
		trpt_cb:   transport_callback,
		core_cb:   core_callback,
		term_cb:   term_callback,
	}
}

// Event is used to send events to the transaction, which are handled in the Start() method
func (trans Sitrans) Event(msg *SIPMessage) {
	if trans.state == terminated {
		return
	}

	trans.transc <- msg
}

// Start initiates the transaction processing by running the main event loop
func (trans *Sitrans) Start() {
	//log.Trace().Str("transaction_id", trans.id.String()).Msg("Starting INVITE server transaction")
	trans.timerprv.start(tiprovsion_dur)

	trans.call_core_callback(trans.message)

	for {
		select {
		case msg := <-trans.transc:
			trans.handle_msg(msg)
		case <-trans.timerprv.Timer.C:
			trans.handle_timer(trans.timerprv)
		case <-trans.timerg.Timer.C:
			trans.handle_timer(trans.timerg)
		case <-trans.timerh.Timer.C:
			trans.handle_timer(trans.timerh)
		case <-trans.timeri.Timer.C:
			trans.handle_timer(trans.timeri)
		}

		if trans.state == terminated {
			//log.Trace().Str("transaction_id", trans.id.String()).Msg("Transaction terminated")
			close(trans.transc)
			break
		}
	}
}

// handle_timer processes events triggered by timer expirations
func (trans *Sitrans) handle_timer(timer *transTimer) {
	//log.Trace().Str("transaction_id", trans.id.String()).Str("timer", timer.ID).Msg("Handling timer event")
	switch timer {
	case trans.timerh:
		trans.state = terminated
		trans.call_term_callback(TIMEOUT)
	case trans.timerprv:
		if trans.state == proceeding {
			trying100 := makeGenericResponse(100, []byte("TRYING"), trans.message)
			trans.call_transport_callback(trying100)
		}
	case trans.timerg:
		if trans.state == completed {
			trans.timerg.start(min(2*trans.timerg.Duration, t2))
			trans.call_transport_callback(trans.last_res)
		}
	case trans.timeri:
		if trans.state == confirmed {
			trans.state = terminated
			trans.call_term_callback(NORMAL)
		}
	}
}

// handle_msg processes received SIP messages (requests or responses)
func (trans *Sitrans) handle_msg(msg *SIPMessage) {
	//log.Trace().Str("transaction_id", trans.id.String()).Interface("message", msg).Msg("Handling message event")

	if msg.Request != nil {
		if msg.Request.Method == Ack && trans.state == completed {
			trans.timerg.stop()
			trans.timerh.stop()
			trans.timeri.start(tii_dur)
			trans.state = confirmed
		} else if msg.Request.Method == Invite && trans.state == completed {
			trans.call_transport_callback(trans.last_res)
		}
		return
	}

	status_code := msg.Response.StatusCode
	if status_code >= 100 && status_code < 200 && trans.state == proceeding {
		trans.timerprv.stop()
		trans.call_transport_callback(msg)
	} else if status_code >= 200 && status_code <= 300 && trans.state == proceeding {
		trans.state = terminated
		trans.call_term_callback(NORMAL)
		trans.call_transport_callback(msg)
	} else if status_code > 300 {
		trans.timerg.start(tig_dur)
		trans.timerh.start(tih_dur)
		trans.last_res = msg
		trans.state = completed
		trans.call_transport_callback(msg)
	}
}

// Helper functions for callbacks
func (sitrans Sitrans) call_core_callback(message *SIPMessage) {
	//log.Trace().Str("transaction_id", sitrans.id.String()).Interface("message", message).Msg("Invoking core callback")
	sitrans.core_cb(sitrans.transport, message)
}

func (sitrans Sitrans) call_transport_callback(message *SIPMessage) {
	//log.Trace().Str("transaction_id", sitrans.id.String()).Interface("message", message).Msg("Invoking transport callback")
	if !sitrans.trpt_cb(sitrans.transport, message) {
		sitrans.state = terminated
		sitrans.call_term_callback(ERROR)
	}
}

func (sitrans Sitrans) call_term_callback(reason TERM_REASON) {
	//log.Trace().Str("transaction_id", sitrans.id.String()).Interface("termination_reason", reason).Msg("Invoking termination callback")
	sitrans.term_cb(sitrans.id, reason)
}

func makeGenericResponse(status_code int, reason []byte, request *SIPMessage) *SIPMessage {
	req_hdr := request.Headers
	res_hdr := make(map[SIPHeader][][]byte)

	if value, ok := req_hdr[Via]; ok {
		res_hdr[Via] = value
	}

	if value, ok := req_hdr[SessionID]; ok {
		res_hdr[SessionID] = value
	}

	res_hdr[ContentLength] = [][]byte{[]byte("0")}

	return &SIPMessage{
		Startline:  Startline{Response: &Response{StatusCode: status_code, ReasonPhrase: reason}},
		From:       request.From,
		To:         request.To,
		CallID:     request.CallID,
		TopmostVia: request.TopmostVia,
		CSeq:       request.CSeq,
		Headers:    res_hdr,
	}
}
