package siptrans

import (
	"gossip/sipmess"
	"gossip/siptransp"

	"github.com/rs/zerolog/log"
)

/*
           				  |Request received
                         |pass to TU
                         V
                   +-----------+
                   |           |
                   | Trying    |-------------+
                   |           |             |
                   +-----------+             |200-699 from TU
                         |                   |send response
                         |1xx from TU        |
                         |send response      |
                         |                   |
      Request            V      1xx from TU  |
      send response+-----------+send response|
          +--------|           |--------+    |
          |        | Proceeding|        |    |
          +------->|           |<-------+    |
   +<--------------|           |             |
   |Trnsprt Err    +-----------+             |
   |Inform TU            |                   |
   |                     |                   |
   |                     |200-699 from TU    |
   |                     |send response      |
   |  Request            V                   |
   |  send response+-----------+             |
   |      +--------|           |             |
   |      |        | Completed |<------------+
   |      +------->|           |
   +<--------------|           |
   |Trnsprt Err    +-----------+
   |Inform TU            |
   |                     |Timer J fires
   |                     |-
   |                     |
   |                     V
   |               +-----------+
   |               |           |
   +-------------->| Terminated|
                   |           |
                   +-----------+
*/

// NIstrans represents the state machine for a Non-Invite Server Transaction
type NIstrans struct {
	id        TransID                                              // Transaction ID
	state     state                                                // Current state of the transaction
	message   *sipmess.SIPMessage                                  // The SIP message associated with the transaction
	transport *siptransp.Transport                                 // Transport layer for sending and receiving messages
	last_res  *sipmess.SIPMessage                                  // The last response received
	timerJ    *transTimer                                          // Timer J for retransmission
	transc    chan *sipmess.SIPMessage                             // Channel for receiving events like timeouts or messages
	trpt_cb   func(*siptransp.Transport, *sipmess.SIPMessage) bool // Callback for transport layer
	core_cb   func(*siptransp.Transport, *sipmess.SIPMessage)      // Callback for core layer
	term_cb   func(TransID, TERM_REASON)                           // Termination callback
}

// Make creates and initializes a new NIstrans instance with the given message and callbacks
func MakeNIST(
	id TransID,
	msg *sipmess.SIPMessage,
	transport *siptransp.Transport,
	core_callback func(*siptransp.Transport, *sipmess.SIPMessage),
	transport_callback func(*siptransp.Transport, *sipmess.SIPMessage) bool,
	term_callback func(TransID, TERM_REASON),
) *NIstrans {
	log.Trace().Str("transaction_id", id.String()).Interface("message", msg).Interface("transport", transport).Msg("Creating new Non-Invite server transaction")
	return &NIstrans{
		id:        id,
		message:   msg,
		transport: transport,
		transc:    make(chan *sipmess.SIPMessage, 5),
		timerJ:    newTransTimer("Timer J"),
		state:     trying,
		trpt_cb:   transport_callback,
		core_cb:   core_callback,
		term_cb:   term_callback,
	}
}

// Event is used to send events to the transaction, which are handled in the Start() method
func (trans *NIstrans) Event(msg *sipmess.SIPMessage) {
	if trans.state == terminated {
		return
	}

	trans.transc <- msg
}

// Start initiates the transaction processing by running the main event loop
func (trans *NIstrans) Start() {
	log.Trace().Str("transaction_id", trans.id.String()).Msg("Starting Non-Invite server transaction")

	// Call the core callback with the original message
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", trans.message).Msg("Initial action: Passing request to core")
	trans.call_core_callback(trans.message)

	for {
		select {
		case msg := <-trans.transc:
			trans.handle_msg(msg)
		case <-trans.timerJ.Timer.C:
			trans.handle_timer(trans.timerJ)
		}

		if trans.state == terminated {
			log.Trace().Str("transaction_id", trans.id.String()).Msg("Transaction terminated")
			close(trans.transc)
			break
		}
	}
}

// handle_timer processes timeout events (Timer J)
func (trans *NIstrans) handle_timer(timer *transTimer) {
	log.Trace().Str("transaction_id", trans.id.String()).Str("timer", timer.ID).Msg("Handling timer event")
	if timer == trans.timerJ && trans.state == completed {
		trans.state = terminated
		trans.call_term_callback(NORMAL)
	}
}

// handle_msg processes received SIP messages (requests or responses)
func (trans *NIstrans) handle_msg(msg *sipmess.SIPMessage) {
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", msg).Msg("Handling message event")

	if msg.Request != nil {
		if trans.state == proceeding || trans.state == completed {
			trans.call_transport_callback(trans.last_res)
		}
		return
	}

	status_code := msg.Response.StatusCode
	if status_code >= 100 && status_code < 200 {
		trans.state = proceeding
		trans.call_core_callback(msg)
		trans.call_transport_callback(msg)
	} else if status_code >= 200 && status_code <= 699 {
		trans.state = completed
		trans.last_res = msg
		trans.call_core_callback(msg)
		trans.call_transport_callback(msg)
		trans.timerJ.start(tij_dur)
	}
}

// call_core_callback invokes the core callback with the provided event
func (trans *NIstrans) call_core_callback(msg *sipmess.SIPMessage) {
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", msg).Msg("Invoking core callback")
	trans.core_cb(trans.transport, msg)
}

// call_transport_callback invokes the transport callback with the provided event
func (trans *NIstrans) call_transport_callback(msg *sipmess.SIPMessage) {
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", msg).Msg("Invoking transport callback")
	if !trans.trpt_cb(trans.transport, msg) {
		trans.state = terminated
		trans.call_term_callback(ERROR)
	}
}

// call_term_callback invokes the termination callback with the provided reason
func (trans *NIstrans) call_term_callback(reason TERM_REASON) {
	log.Trace().Str("transaction_id", trans.id.String()).Interface("termination_reason", reason).Msg("Invoking termination callback")
	trans.term_cb(trans.id, reason)
}
