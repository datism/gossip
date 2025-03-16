package ictrans

import (
	"gossip/message"
	"gossip/message/cseq"
	"gossip/transaction"
	"gossip/transport"

	"github.com/rs/zerolog/log"
)

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

// Constants representing timer durations in milliseconds
const t1 = 500          // Default value for T1 (Round Trip Time estimate between client and server)
const tia_dur = t1      // Timer A duration, starts with T1 duration
const tib_dur = 64 * t1 // Timer B duration, starts with 64*T1
const tid_dur = 32000   // Timer D duration (in case of transport reliability issues), typically 32 seconds

type timer int

// Constants representing timer types
const (
	timer_a = iota // Timer A: Retransmit the request on timeout
	timer_b        // Timer B: Transaction timeout (in the calling state)
	timer_d        // Timer D: Completion timeout after receiving final response
)

func (t timer) String() string {
	switch t {
	case timer_a:
		return "Timer A"
	case timer_b:
		return "Timer B"
	case timer_d:
		return "Timer D"
	default:
		return "Unknown"
	}
}

// Define the states of the INVITE client transaction
type state int

const (
	calling    = iota // Initial state after INVITE is sent, waiting for provisional or final response
	proceeding        // State after receiving a provisional (1xx) response
	completed         // State after receiving a final (2xx-6xx) response
	terminated        // Final state, transaction is completed and destroyed
)

// Ictrans represents a SIP INVITE client transaction
type Ictrans struct {
	id        transaction.TransID                                  // Transaction ID
	state     state                                                // Current state of the transaction
	transport *transport.Transport                                 // Transport layer
	message   *message.SIPMessage                                  // The SIP message being processed (INVITE or response)
	ack       *message.SIPMessage                                  // The ACK message to be generated
	timers    [3]transaction.Timer                                 // The timers used to manage retransmissions and timeouts
	transc    chan *message.SIPMessage                             // Channel for receiving events and processing them
	trpt_cb   func(*transport.Transport, *message.SIPMessage) bool // Transport callback
	core_cb   func(*transport.Transport, *message.SIPMessage)      // Core callback
	term_cb   func(transaction.TransID, transaction.TERM_REASON)
}

// Make creates a new instance of a client transaction, initializing timers and setting initial state
func Make(
	id transaction.TransID, // Transaction ID
	msg *message.SIPMessage, // The INVITE message to be processed
	transport *transport.Transport, // Transport layer
	core_callback func(*transport.Transport, *message.SIPMessage), // Core callback
	transport_callback func(*transport.Transport, *message.SIPMessage) bool, // Transport layer callback
	term_callback func(transaction.TransID, transaction.TERM_REASON), // Termination callback
) *Ictrans {
	timera := transaction.NewTimer() // Timer A for retransmissions
	timerb := transaction.NewTimer() // Timer B for transaction timeout
	timerd := transaction.NewTimer() // Timer D for completion timeout

	log.Trace().Str("transaction_id", id.String()).Interface("message", msg).Interface("transport", transport).Msg("Creating new INVITE client transaction")
	return &Ictrans{
		id:        id,                                           // Set transaction ID
		message:   msg,                                          // The initial SIP message (INVITE)
		ack:       initAck(msg),                                 // ACK message to be generated
		transc:    make(chan *message.SIPMessage, 5),            // Channel to communicate events
		timers:    [3]transaction.Timer{timera, timerb, timerd}, // Initialize timers
		state:     calling,                                      // Start with the calling state
		trpt_cb:   transport_callback,                           // Set transport callback
		core_cb:   core_callback,                                // Set core callback
		term_cb:   term_callback,                                // Set termination callback
		transport: transport,                                    // Set transport
	}
}

// Event triggers an event in the transaction. The event can be a SIP message or timeout.
func (trans Ictrans) Event(msg *message.SIPMessage) {
	if trans.state == terminated {
		return
	}

	trans.transc <- msg
}

// start is the main loop that processes events in the client transaction.
func (trans *Ictrans) Start() {
	log.Trace().Str("transaction_id", trans.id.String()).Msg("Starting INVITE client transaction")

	// Initial action: Call transport callback to send INVITE message
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", trans.message).Msg("Initial action: Sending request")
	call_transport_callback(trans, trans.message)
	// Start Timer A (T1) for retransmissions and Timer B (64*T1) for transaction timeout
	trans.timers[timer_a].Start(tia_dur)
	trans.timers[timer_b].Start(tib_dur)

	// Event loop that listens for events (SIP messages or timer expirations)
	for {
		select {
		case msg := <-trans.transc: // Message event (SIP response)
			trans.handle_msg(msg)
		case <-trans.timers[timer_a].Chan(): // Timer A expired, triggering a retransmission
			trans.handle_timer(timer_a)
		case <-trans.timers[timer_b].Chan(): // Timer B expired, transaction timed out
			trans.handle_timer(timer_b)
		case <-trans.timers[timer_d].Chan(): // Timer D expired, termination after final response
			trans.handle_timer(timer_d)
		}

		// If the transaction is terminated, exit the loop
		if trans.state == terminated {
			log.Trace().Str("transaction_id", trans.id.String()).Msg("Transaction terminated")
			close(trans.transc) // Close the event channel when the transaction ends
			break
		}
	}
}

// handle_timer processes timeout events, which can trigger retransmissions or state transitions
func (trans *Ictrans) handle_timer(timer timer) {
	log.Trace().Str("transaction_id", trans.id.String()).Str("timer", timer.String()).Msg("Handling timer event")

	if timer == timer_b { // Timer B expired, inform TU of timeout and terminate transaction
		trans.state = terminated
		call_term_callback(trans, transaction.TIMEOUT)
	} else if timer == timer_a && trans.state == calling { // Timer A expired in calling state, retransmit INVITE
		trans.timers[timer_a].Start(trans.timers[timer_a].Duration() * 2) // Double Timer A duration
		call_transport_callback(trans, trans.message)
	} else if timer == timer_d && trans.state == completed { // Timer D expired in completed state, terminate transaction
		trans.state = terminated
		call_term_callback(trans, transaction.NORMAL)
	}
}

// handle_msg processes received SIP messages, transitioning states based on response codes
func (trans *Ictrans) handle_msg(response *message.SIPMessage) {
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", response).Msg("Handling message event")

	if response.Response == nil { // Invalid or missing response, ignore the event
		return
	}

	status_code := response.Response.StatusCode // Get the response's status code

	if status_code >= 100 && status_code < 200 { // Provisional response (1xx)
		if trans.state == calling { // If in calling state, transition to proceeding
			trans.timers[timer_a].Stop()        // Stop Timer A as no more retransmissions are needed
			trans.state = proceeding            // Transition to proceeding state
			call_core_callback(trans, response) // Pass 1xx response to the core callback
		} else if trans.state == proceeding { // In proceeding state, pass 1xx to the TU
			call_core_callback(trans, response)
		}
	} else if status_code >= 200 && status_code <= 300 && trans.state < completed { // Final success response (2xx)
		trans.state = terminated            // Transition to terminated state
		call_core_callback(trans, response) // Pass the final response to the core
		call_term_callback(trans, transaction.NORMAL)
	} else if status_code > 300 { // Error response (3xx-6xx)
		if trans.state < completed { // If in calling or proceeding state, generate ACK and stop Timer B
			updateAck(trans.ack, response)            // Create an ACK for the response
			trans.timers[timer_b].Stop()              // Stop Timer B (transaction timeout)
			trans.timers[timer_d].Start(tid_dur)      // Start Timer D (completion timeout)
			trans.state = completed                   // Transition to completed state
			call_transport_callback(trans, trans.ack) // Send the ACK
			call_core_callback(trans, response)
		} else if trans.state == completed { // In completed state, just retransmit the ACK
			updateAck(trans.ack, response)
			call_transport_callback(trans, trans.ack)
		}
	}
}

// call_core_callback invokes the core callback to handle transaction-related events
func call_core_callback(citrans *Ictrans, message *message.SIPMessage) {
	log.Trace().Str("transaction_id", citrans.id.String()).Interface("message", message).Msg("Invoking core callback")
	citrans.core_cb(citrans.transport, message) // Call the core callback
}

// call_transport_callback invokes the transport callback to send or receive messages
func call_transport_callback(citrans *Ictrans, message *message.SIPMessage) {
	log.Trace().Str("transaction_id", citrans.id.String()).Interface("message", message).Msg("Invoking transport callback")
	if !citrans.trpt_cb(citrans.transport, message) { // Call the transport callback
		call_term_callback(citrans, transaction.ERROR)
		citrans.state = terminated
	}
}

func call_term_callback(citrans *Ictrans, reason transaction.TERM_REASON) {
	log.Trace().Str("transaction_id", citrans.id.String()).Interface("termination_reason", reason).Msg("Invoking termination callback")
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
func initAck(inv *message.SIPMessage) *message.SIPMessage {
	ack_hdr := make(map[string][]string)
	if sessionid := inv.GetHeader("session-id"); sessionid != nil {
		ack_hdr["session-id"] = sessionid
	}
	if routes := inv.GetHeader("route"); routes != nil {
		ack_hdr["route"] = routes
	}

	return &message.SIPMessage{
		Startline: message.Startline{
			Request: &message.Request{
				Method:     "ACK",
				RequestURI: inv.Request.RequestURI,
			},
		},
		From:       inv.From,
		CallID:     inv.CallID,
		TopmostVia: inv.TopmostVia,
		CSeq: &cseq.SIPCseq{
			Method: "ACK",
			Seq:    inv.CSeq.Seq,
		},
		Headers: ack_hdr,
	}
}

func updateAck(ack *message.SIPMessage, response *message.SIPMessage) {
	ack.To = response.To
}
