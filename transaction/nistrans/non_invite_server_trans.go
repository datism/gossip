package nistrans

import (
	"gossip/message"
	"gossip/transaction"
	"gossip/transport"

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

// Timer constants
const t1 = 500

const tij_dur = 64 * t1 // Timer J duration (64*T1)

type timer int

// Timer constants (Indexes)
const (
	timer_j = iota
)

func (t timer) String() string {
	switch t {
	case timer_j:
		return "Timer J"
	default:
		return "Unknown"
	}
}

// Define the states for the Non-Invite Server Transaction
type state int

const (
	trying     state = iota // The transaction is waiting for a response
	proceeding              // The transaction has sent a provisional response
	completed               // The transaction has sent a final response
	terminated              // The transaction has been terminated
)

func (s state) String() string {
	switch s {
	case trying:
		return "Trying"
	case proceeding:
		return "Proceeding"
	case completed:
		return "Completed"
	case terminated:
		return "Terminated"
	default:
		return "Unknown"
	}
}

// NIstrans represents the state machine for a Non-Invite Server Transaction
type NIstrans struct {
	id        transaction.TransID                                  // Transaction ID
	state     state                                                // Current state of the transaction
	message   *message.SIPMessage                                  // The SIP message associated with the transaction
	transport *transport.Transport                                 // Transport layer for sending and receiving messages
	last_res  *message.SIPMessage                                  // The last response received
	timers    [1]transaction.Timer                                 // Timer J for retransmission
	transc    chan *message.SIPMessage                             // Channel for receiving events like timeouts or messages
	trpt_cb   func(*transport.Transport, *message.SIPMessage) bool // Callback for transport layer
	core_cb   func(*transport.Transport, *message.SIPMessage)      // Callback for core layer
	term_cb   func(transaction.TransID, transaction.TERM_REASON)
}

// Make creates and initializes a new NIstrans instance with the given message and callbacks
func Make(
	id transaction.TransID, // Transaction ID
	msg *message.SIPMessage,
	transport *transport.Transport,
	core_callback func(*transport.Transport, *message.SIPMessage),
	transport_callback func(*transport.Transport, *message.SIPMessage) bool,
	term_callback func(transaction.TransID, transaction.TERM_REASON),
) *NIstrans {
	// Create new timers for the transaction state machine
	timerJ := transaction.NewTimer() // Timer J for the Completed state

	log.Trace().Str("transaction_id", id.String()).Interface("message", msg).Interface("transport", transport).Msg("Creating new Non-Invite server transaction")
	// Return a new NIstrans instance with the provided parameters
	return &NIstrans{
		id:        id,  // Set transaction ID
		message:   msg, // Set the SIP message
		transport: transport,
		transc:    make(chan *message.SIPMessage, 5), // Channel for event communication
		timers:    [1]transaction.Timer{timerJ},      // Initialize Timer J
		state:     trying,                            // Initial state is "trying"
		trpt_cb:   transport_callback,                // Transport callback for message transmission
		core_cb:   core_callback,                     // Core callback for message handling in the application logic
		term_cb:   term_callback,                     // Termination callback for the transaction
	}
}

// Event is used to send events to the transaction, which are handled in the start() method
func (trans NIstrans) Event(msg *message.SIPMessage) {
	if trans.state == terminated {
		return
	}

	trans.transc <- msg // Push the event to the transc channel for processing
}

// Start initiates the transaction processing by running the start method in a goroutine
func (trans *NIstrans) Start() {
	log.Trace().Str("transaction_id", trans.id.String()).Msg("Starting Non-Invite server transaction")
	// Start the timer for provisional responses (Timer J)
	trans.timers[timer_j].Start(tij_dur)

	// Call the core callback with the original message
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", trans.message).Msg("Initial action: Sending request to core")
	call_core_callback(trans, trans.message)

	// Main event loop for processing the transaction
	for {
		// Wait for events from the transc channel or from the timers
		select {
		case msg := <-trans.transc: // Event received from the transc channel
			trans.handle_msg(msg)
		case <-trans.timers[timer_j].Chan(): // Timer J has expired
			trans.handle_timer(timer_j)
		}

		// If the state is terminated, break the loop and stop the transaction
		if trans.state == terminated {
			log.Trace().Str("transaction_id", trans.id.String()).Msg("Transaction terminated")
			close(trans.transc) // Close the event channel when the transaction ends
			break
		}
	}
}

// handle_timer processes events triggered by timer expirations
func (trans *NIstrans) handle_timer(timer timer) {
	log.Trace().Str("transaction_id", trans.id.String()).Str("state", trans.state.String()).Str("timer", timer.String()).Msg("Handling timer event")
	// Handle the event based on which timer expired
	if timer == timer_j && trans.state == completed {
		// Timer J expired: Transaction is terminated due to timeout
		trans.state = terminated
		call_term_callback(trans, transaction.NORMAL)
	}
}

// handle_msg processes received SIP messages (requests or responses)
func (trans *NIstrans) handle_msg(msg *message.SIPMessage) {
	log.Trace().Str("transaction_id", trans.id.String()).Str("state", trans.state.String()).Interface("message", msg).Msg("Handling message event")

	// Handle incoming SIP request
	if msg.Request != nil {
		if trans.state == proceeding || trans.state == completed {
			// Retransmit the last response if in proceeding or completed state
			call_transport_callback(trans, trans.last_res)
		}
		return
	}

	// Handle SIP responses based on their status code
	status_code := msg.Response.StatusCode
	if status_code >= 100 && status_code < 200 {
		// Provisional responses (1xx): Move to Proceeding state
		trans.state = proceeding
		call_core_callback(trans, msg)
		call_transport_callback(trans, msg)
	} else if status_code >= 200 && status_code <= 699 {
		// Final responses (2xx-6xx): Move to Completed state
		trans.state = completed
		trans.last_res = msg
		call_core_callback(trans, msg)
		call_transport_callback(trans, msg)
		// Start Timer J for retransmissions
		trans.timers[timer_j].Start(tij_dur)
	}
}

// call_core_callback invokes the core callback to handle transaction-related events
func call_core_callback(trans *NIstrans, msg *message.SIPMessage) {
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", msg).Msg("Invoking core callback")
	trans.core_cb(trans.transport, msg)
}

// call_transport_callback invokes the transport callback to send or receive messages
func call_transport_callback(trans *NIstrans, msg *message.SIPMessage) {
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", msg).Msg("Invoking transport callback")
	if !trans.trpt_cb(trans.transport, msg) {
		call_term_callback(trans, transaction.ERROR)
		trans.state = terminated
	}
}

// call_term_callback invokes the termination callback with the provided reason
func call_term_callback(trans *NIstrans, reason transaction.TERM_REASON) {
	log.Trace().Str("transaction_id", trans.id.String()).Interface("termination_reason", reason).Msg("Invoking termination callback")
	trans.term_cb(trans.id, reason)
}
