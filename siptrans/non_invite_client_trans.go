package siptrans

import (
	"gossip/sipmess"
	"gossip/siptransp"

	"github.com/rs/zerolog/log"
)

/*
                             |Request from TU
                             |send request
         Timer E             V
         send request  +-----------+
             +---------|           |-------------------+
             |         |  Trying   |  Timer F          |
             +-------->|           |  or Transport Err.|
                       +-----------+  inform TU        |
          200-699         |  |                         |
          resp. to TU     |  |1xx                      |
          +---------------+  |resp. to TU              |
          |                  |                         |
          |   Timer E        V       Timer F           |
          |   send req +-----------+ or Transport Err. |
          |  +---------|           | inform TU         |
          |  |         |Proceeding |------------------>|
          |  +-------->|           |-----+             |
          |            +-----------+     |1xx          |
          |              |      ^        |resp to TU   |
          | 200-699      |      +--------+             |
          | resp. to TU  |                             |
          |              |                             |
          |              V                             |
          |            +-----------+                   |
          |            |           |                   |
          |            | Completed |                   |
          |            |           |                   |
          |            +-----------+                   |
          |              ^   |                         |
          |              |   | Timer K                 |
          +--------------+   | -                       |
                             |                         |
                             V                         |
       NOTE:           +-----------+                   |
                       |           |                   |
   transitions         | Terminated|<------------------+
   labeled with        |           |
   the event           +-----------+
   over the action
   to take
*/

// NIctrans represents the state machine for a Non-Invite Client Transaction
type NIctrans struct {
	id        TransID                                              // Transaction ID
	state     state                                                // Current state of the transaction
	message   *sipmess.SIPMessage                                  // The SIP message associated with the transaction
	transport *siptransp.Transport                                 // Transport layer for sending and receiving messages
	timerE    *transTimer                                          // Timer E for retransmissions
	timerF    *transTimer                                          // Timer F for transaction timeout
	timerK    *transTimer                                          // Timer K for termination after completion
	transc    chan *sipmess.SIPMessage                             // Channel for receiving events like timeouts or messages
	trpt_cb   func(*siptransp.Transport, *sipmess.SIPMessage) bool // Callback for transport layer
	core_cb   func(*siptransp.Transport, *sipmess.SIPMessage)      // Callback for core layer
	term_cb   func(TransID, TERM_REASON)                           // Termination callback
}

// Make creates and initializes a new NIctrans instance with the given message and callbacks
func MakeNICT(
	id TransID,
	msg *sipmess.SIPMessage,
	transport *siptransp.Transport,
	core_callback func(*siptransp.Transport, *sipmess.SIPMessage),
	transport_callback func(*siptransp.Transport, *sipmess.SIPMessage) bool,
	term_callback func(TransID, TERM_REASON),
) *NIctrans {
	log.Trace().Str("transaction_id", id.String()).Interface("message", msg).Interface("transport", transport).Msg("Creating new Non-Invite client transaction")
	return &NIctrans{
		id:        id,
		message:   msg,
		transport: transport,
		transc:    make(chan *sipmess.SIPMessage, 5),
		timerE:    newTransTimer("Timer E"),
		timerF:    newTransTimer("Timer F"),
		timerK:    newTransTimer("Timer K"),
		state:     trying,
		trpt_cb:   transport_callback,
		core_cb:   core_callback,
		term_cb:   term_callback,
	}
}

// Event is used to send events to the transaction, which are handled in the Start() method
func (trans *NIctrans) Event(msg *sipmess.SIPMessage) {
	if trans.state == terminated {
		return
	}

	trans.transc <- msg
}

// Start initiates the transaction processing by running the main event loop
func (trans *NIctrans) Start() {
	log.Trace().Str("transaction_id", trans.id.String()).Msg("Starting Non-Invite client transaction")
	// Start Timer F (64*T1)
	trans.timerF.start(tif_dur)

	// Send the request to the transport layer
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", trans.message).Msg("Initial action: Sending request")
	trans.call_transport_callback(trans.message)

	// Set Timer E for retransmission to fire at T1
	trans.timerE.start(tie_dur)

	for {
		select {
		case msg := <-trans.transc:
			trans.handle_message(msg)
		case <-trans.timerE.Timer.C:
			trans.handle_timer(trans.timerE)
		case <-trans.timerF.Timer.C:
			trans.handle_timer(trans.timerF)
		case <-trans.timerK.Timer.C:
			trans.handle_timer(trans.timerK)
		}

		if trans.state == terminated {
			log.Trace().Str("transaction_id", trans.id.String()).Msg("Transaction terminated")
			close(trans.transc)
			break
		}
	}
}

// handle_timer processes timeout events (Timer E, F, K)
func (trans *NIctrans) handle_timer(timer *transTimer) {
	log.Trace().Str("transaction_id", trans.id.String()).Str("timer", timer.ID).Msg("Handling timer event")
	switch timer {
	case trans.timerF:
		if trans.state < completed {
			trans.state = terminated
			trans.call_term_callback(TIMEOUT)
		}
	case trans.timerE:
		if trans.state < completed {
			trans.timerE.start(min(trans.timerE.Duration*2, t2))
			trans.call_transport_callback(trans.message)
		}
	case trans.timerK:
		if trans.state == completed {
			trans.state = terminated
			trans.call_term_callback(NORMAL)
		}
	}
}

// handle_message processes received SIP messages (responses)
func (trans *NIctrans) handle_message(msg *sipmess.SIPMessage) {
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", msg).Msg("Handling message event")
	if msg.Response == nil {
		return
	}

	status_code := msg.Response.StatusCode
	if status_code >= 100 && status_code < 200 {
		trans.state = proceeding
		trans.call_core_callback(msg)
	} else if status_code >= 200 && status_code <= 699 {
		trans.timerK.start(tik_dur)
		trans.state = completed
		trans.call_core_callback(msg)
	}
}

// call_core_callback invokes the core callback with the provided event
func (trans *NIctrans) call_core_callback(msg *sipmess.SIPMessage) {
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", msg).Msg("Invoking core callback")
	trans.core_cb(trans.transport, msg)
}

// call_transport_callback invokes the transport callback with the provided event
func (trans *NIctrans) call_transport_callback(msg *sipmess.SIPMessage) {
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", msg).Msg("Invoking transport callback")
	if !trans.trpt_cb(trans.transport, msg) {
		trans.state = terminated
		trans.call_term_callback(ERROR)
	}
}

// call_term_callback invokes the termination callback with the provided reason
func (trans *NIctrans) call_term_callback(reason TERM_REASON) {
	log.Trace().Str("transaction_id", trans.id.String()).Interface("termination_reason", reason).Msg("Invoking termination callback")
	trans.term_cb(trans.id, reason)
}
