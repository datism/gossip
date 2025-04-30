package sip

//                     |INVITE from TU
//              Timer A fires     |INVITE sent
//              Reset A,          V                      Timer B fires
//              INVITE sent +-----------+                or Transport Err.
//                +---------|           |---------------+inform TU
//                |         |  Calling  |               |
//                +-------->|           |-------------->|
//                          +-----------+ 2xx           |
//                             |  |       2xx to TU     |
//                             |  |1xx                  |
//     300-699 +---------------+  |1xx to TU            |
//    ACK sent |                  |                     |
// resp. to TU |  1xx             V                     |
//             |  1xx to TU  -----------+               |
//             |  +---------|           |               |
//             |  |         |Proceeding |-------------->|
//             |  +-------->|           | 2xx           |
//             |            +-----------+ 2xx to TU     |
//             |       300-699    |                     |
//             |       ACK sent,  |                     |
//             |       resp. to TU|                     |
//             |                  |                     |      NOTE:
//             |  300-699         V                     |
//             |  ACK sent  +-----------+Transport Err. |  transitions
//             |  +---------|           |Inform TU      |  labeled with
//             |  |         | Completed |-------------->|  the event
//             |  +-------->|           |               |  over the action
//             |            +-----------+               |  to take
//             |              ^   |                     |
//             |              |   | Timer D fires       |
//             +--------------+   | -                   |
//                                |                     |
//                                V                     |
//                          +-----------+               |
//                          |           |               |
//                          | Terminated|<--------------+
//                          |           |
//                          +-----------+

// Ictrans represents a SIP INVITE client transaction
type Ictrans struct {
	id        TransID       // Transaction ID
	state     state         // Current state of the transaction
	transport *SIPTransport // Transport layer
	message   *SIPMessage   // The SIP message being processed (INVITE or response)
	ack       *SIPMessage   // The ACK message to be generated
	timera    *transTimer
	timerb    *transTimer
	timerd    *transTimer
	transc    chan *SIPMessage                      // Channel for receiving events and processing them
	trpt_cb   func(*SIPTransport, *SIPMessage) bool // Transport callback
	core_cb   func(*SIPTransport, *SIPMessage)      // Core callback
	term_cb   func(TransID, TERM_REASON)
}

// Make creates a new instance of a client transaction, initializing timers and setting initial state
func MakeICT(
	id TransID, // siptrans ID
	msg *SIPMessage, // The INVITE message to be processed
	transport *SIPTransport, // Transport layer
	core_callback func(*SIPTransport, *SIPMessage), // Core callback
	transport_callback func(*SIPTransport, *SIPMessage) bool, // Transport layer callback
	term_callback func(TransID, TERM_REASON), // Termination callback
) *Ictrans {
	//log.Trace().Str("siptrans_id", id.String()).Interface("message", msg).Interface("transport", transport).Msg("Creating new INVITE client transaction")
	return &Ictrans{
		id:        id,                        // Set transaction ID
		message:   msg,                       // The initial SIP message (INVITE)
		ack:       initAck(msg),              // ACK message to be generated
		transc:    make(chan *SIPMessage, 5), // Channel to communicate events
		timera:    newTransTimer("timer a"),
		timerb:    newTransTimer("timer b"),
		timerd:    newTransTimer("timer d"),
		state:     calling,            // Start with the calling state
		trpt_cb:   transport_callback, // Set transport callback
		core_cb:   core_callback,      // Set core callback
		term_cb:   term_callback,      // Set termination callback
		transport: transport,          // Set transport
	}
}

// Event triggers an event in the transaction. The event can be a SIP message or timeout.
func (trans Ictrans) Event(msg *SIPMessage) {
	if trans.state == terminated {
		return
	}

	trans.transc <- msg
}

// start is the main loop that processes events in the client transaction.
func (trans *Ictrans) Start() {
	//log.Trace().Str("transaction_id", trans.id.String()).Msg("Starting INVITE client transaction")

	// Initial action: Call transport callback to send INVITE message
	//log.Trace().Str("transaction_id", trans.id.String()).Interface("message", trans.message).Msg("Initial action: Sending request")
	trans.call_transport_callback(trans.message)
	// Start Timer A (T1) for retransmissions and Timer B (64*T1) for transaction timeout
	trans.timera.start(tia_dur)
	trans.timerb.start(tib_dur)

	// Event loop that listens for events (SIP messages or timer expirations)
	for {
		select {
		case msg := <-trans.transc: // Message event (SIP response)
			trans.handle_msg(msg)
		case <-trans.timera.Timer.C: // Timer A expired, triggering a retransmission
			trans.handle_timer(trans.timera)
		case <-trans.timerb.Timer.C: // Timer B expired, transaction timed out
			trans.handle_timer(trans.timerb)
		case <-trans.timerd.Timer.C: // Timer D expired, termination after final response
			trans.handle_timer(trans.timerd)
		}

		// If the transaction is terminated, exit the loop
		if trans.state == terminated {
			//log.Trace().Str("transaction_id", trans.id.String()).Msg("Transaction terminated")
			close(trans.transc) // Close the event channel when the transaction ends
			break
		}
	}
}

// handle_timer processes timeout events, which can trigger retransmissions or state transitions
func (trans *Ictrans) handle_timer(timer *transTimer) {
	//log.Trace().Str("transaction_id", trans.id.String()).Str("timer", timer.ID).Msg("Handling timer event")

	if timer == trans.timerb { // Timer B expired, inform TU of timeout and terminate transaction
		trans.state = terminated
		trans.call_term_callback(TIMEOUT)
	} else if timer == trans.timera && trans.state == calling { // Timer A expired in calling state, retransmit INVITE
		trans.timera.start(trans.timera.Duration * 2) // Double Timer A duration
		trans.call_transport_callback(trans.message)
	} else if timer == trans.timerd && trans.state == completed { // Timer D expired in completed state, terminate transaction
		trans.state = terminated
		trans.call_term_callback(NORMAL)
	}
}

// handle_msg processes received SIP messages, transitioning states based on response codes
func (trans *Ictrans) handle_msg(response *SIPMessage) {
	//log.Trace().Str("transaction_id", trans.id.String()).Interface("message", response).Msg("Handling message event")

	if response.Response == nil { // Invalid or missing response, ignore the event
		return
	}

	status_code := response.Response.StatusCode // Get the response's status code

	if status_code >= 100 && status_code < 200 { // Provisional response (1xx)
		if trans.state == calling { // If in calling state, transition to proceeding
			trans.timera.stop()                // Stop Timer A as no more retransmissions are needed
			trans.state = proceeding           // Transition to proceeding state
			trans.call_core_callback(response) // Pass 1xx response to the core callback
		} else if trans.state == proceeding { // In proceeding state, pass 1xx to the TU
			trans.call_core_callback(response)
		}
	} else if status_code >= 200 && status_code <= 300 && trans.state < completed { // Final success response (2xx)
		trans.state = terminated           // Transition to terminated state
		trans.call_core_callback(response) // Pass the final response to the core
		trans.call_term_callback(NORMAL)
	} else if status_code > 300 { // Error response (3xx-6xx)
		if trans.state < completed { // If in calling or proceeding state, generate ACK and stop Timer B
			updateAck(trans.ack, response)           // Create an ACK for the response
			trans.timerb.stop()                      // Stop Timer B (transaction timeout)
			trans.timerd.start(tid_dur)              // Start Timer D (completion timeout)
			trans.state = completed                  // Transition to completed state
			trans.call_transport_callback(trans.ack) // Send the ACK
			trans.call_core_callback(response)
		} else if trans.state == completed { // In completed state, just retransmit the ACK
			updateAck(trans.ack, response)
			trans.call_transport_callback(trans.ack)
		}
	}
}

// call_core_callback invokes the core callback to handle transaction-related events
func (citrans Ictrans) call_core_callback(message *SIPMessage) {
	//log.Trace().Str("transaction_id", citrans.id.String()).Interface("message", message).Msg("Invoking core callback")
	citrans.core_cb(citrans.transport, message) // Call the core callback
}

// call_transport_callback invokes the transport callback to send or receive messages
func (citrans Ictrans) call_transport_callback(message *SIPMessage) {
	//log.Trace().Str("transaction_id", citrans.id.String()).Interface("message", message).Msg("Invoking transport callback")
	if !citrans.trpt_cb(citrans.transport, message) { // Call the transport callback
		citrans.state = terminated
		citrans.call_term_callback(ERROR)
	}
}

func (citrans Ictrans) call_term_callback(reason TERM_REASON) {
	//log.Trace().Str("transaction_id", citrans.id.String()).Interface("termination_reason", reason).Msg("Invoking termination callback")
	citrans.term_cb(citrans.id, reason)
}

/*
	 RFC 3261 17.1.1.3
			The ACK request constructed by the client transaction MUST contain
		 	values for the Call-ID, From, and Request-URI that are equal to the
		  	values of those header fields in the request passed to the transport
		  	by the client transaction (call this the "original request").

		   	The To header field in the ACK MUST equal the To header field in the
		  	response being acknowledged, and therefore will usually differ from
		  	the To header field in the original request by the addition of the
		  	tag parameter.

		   	The ACK MUST contain a single Via header field, and
		 	this MUST be equal to the top Via header field of the original
		  	request.

		   	The CSeq header field in the ACK MUST contain the same
		  	value for the sequence number as was present in the original request,
		  	but the method parameter MUST be equal to "ACK".

		   	If the INVITE request whose response is being acknowledged had Route
		 	header fields, those header fields MUST appear in the ACK.  This is
		  	to ensure that the ACK can be routed properly through any downstream
		  	stateless proxies.
*/
func initAck(inv *SIPMessage) *SIPMessage {
	ack_hdr := make(map[SIPHeader][][]byte)
	if val, ok := inv.Headers[SessionID]; ok {
		ack_hdr[SessionID] = val
	}
	if val, ok := inv.Headers[Route]; ok {
		ack_hdr[Route] = val
	}

	return &SIPMessage{
		Startline: Startline{
			Request: &Request{
				Method:     Ack,
				RequestURI: inv.Request.RequestURI,
			},
		},
		From:       inv.From,
		CallID:     inv.CallID,
		TopmostVia: inv.TopmostVia,
		CSeq: SIPCseq{
			Method: Ack,
			Seq:    inv.CSeq.Seq,
		},
		Headers: ack_hdr,
	}
}

func updateAck(ack *SIPMessage, response *SIPMessage) {
	ack.To = response.To
}
