package istrans

import (
	"gossip/message"
	"gossip/transaction"
	"gossip/transport"

	"github.com/rs/zerolog/log"
)

//      		  |INVITE
//                    |pass INV to TU
// INVITE             V send 100 if TU won't in 200ms
// send response+-----------+
//     +--------|           |--------+101-199 from TU
//     |        | Proceeding|        |send response
//     +------->|           |<-------+
//              |           |          Transport Err.
//              |           |          Inform TU
//              |           |--------------->+
//              +-----------+                |
// 300-699 from TU |     |2xx from TU        |
// send response   |     |send response      |
//                 |     +------------------>+
//                 |                         |
// INVITE          V          Timer G fires  |
// send response+-----------+ send response  |
//     +--------|           |--------+       |
//     |        | Completed |        |       |
//     +------->|           |<-------+       |
//              +-----------+                |
//                 |     |                   |
//             ACK |     |                   |
//             -   |     +------------------>+
//                 |        Timer H fires    |
//                 V        or Transport Err.|
//              +-----------+  Inform TU     |
//              |           |                |
//              | Confirmed |                |
//              |           |                |
//              +-----------+                |
//                    |                      |
//                    |Timer I fires         |
//                    |-                     |
//                    |                      |
//                    V                      |
//              +-----------+                |
//              |           |                |
//              | Terminated|<---------------+
//              |           |
//              +-----------+

// Define the timer constants as per RFC
const t1 = 500  // Timer T1 duration (500ms)
const t2 = 4000 // Timer T2 duration (4000ms)
const t4 = 5000 // Timer T4 duration (5000ms)

// Define durations for specific timers used in the state machine
const tiprv_dur = 200   // Provisional response timer duration (200ms)
const tig_dur = t1      // Timer G duration (T1)
const tih_dur = 64 * t1 // Timer H duration (64 * T1)
const tii_dur = t4      // Timer I duration (T4)

type timer int

// Enum-like constants for different timer indices
const (
	timer_prv = iota
	timer_g
	timer_h
	timer_i
)

func (t timer) String() string {
	switch t {
	case timer_prv:
		return "Timer Provisional"
	case timer_g:
		return "Timer G"
	case timer_h:
		return "Timer H"
	case timer_i:
		return "Timer I"
	default:
		return "Unknown"
	}
}

// Define the states for the transaction
type state int

const (
	proceeding state = iota // In this state, the server is processing the request
	completed               // Transaction is completed, waiting for ACK or further responses
	confirmed               // ACK received, transaction confirmed
	terminated              // The transaction is finished (either successfully or failed)
)

// Sitrans represents the state machine for an INVITE transaction
type Sitrans struct {
	id        transaction.TransID                                  // Transaction ID
	state     state                                                // Current state of the transaction
	message   *message.SIPMessage                                  // The SIP message associated with the transaction
	transport *transport.Transport                                 // Transport layer for sending and receiving messages
	last_res  *message.SIPMessage                                  // The last response received
	timers    [4]transaction.Timer                                 // List of timers used for managing retransmissions and timeouts
	transc    chan *message.SIPMessage                             // Channel for receiving events like timeouts or messages
	trpt_cb   func(*transport.Transport, *message.SIPMessage) bool // Transport callback
	core_cb   func(*transport.Transport, *message.SIPMessage)      // Core callback
	term_cb   func(transaction.TransID, transaction.TERM_REASON)
}

// Make creates and initializes a new Sitrans instance with the given message and callbacks
func Make(
	id transaction.TransID, // Transaction ID
	msg *message.SIPMessage,
	transport *transport.Transport,
	core_callback func(*transport.Transport, *message.SIPMessage), // Core callback
	transport_callback func(*transport.Transport, *message.SIPMessage) bool, // Transport layer callback
	term_callback func(transaction.TransID, transaction.TERM_REASON), // Termination callback
) *Sitrans {
	// Create new timers for the transaction state machine
	timerprv := transaction.NewTimer()
	timerg := transaction.NewTimer()
	timerdh := transaction.NewTimer()
	timeri := transaction.NewTimer()

	log.Trace().Str("transaction_id", id.String()).Interface("message", msg).Interface("transport", transport).Msg("Creating new INVITE server transaction")
	// Return a new Sitrans instance with the provided parameters
	return &Sitrans{
		id:        id,                                                      // Set transaction ID
		message:   msg,                                                     // Set the SIP message
		transport: transport,                                               // Set the transport layer
		transc:    make(chan *message.SIPMessage, 5),                       // Channel for event communication
		timers:    [4]transaction.Timer{timerprv, timerg, timerdh, timeri}, // Initialize the timers array
		state:     proceeding,                                              // Initial state is "proceeding"
		trpt_cb:   transport_callback,                                      // Transport callback for message transmission
		core_cb:   core_callback,                                           // Core callback for message handling in the application logic
		term_cb:   term_callback,                                           // Termination callback for the transaction
	}
}

// Event is used to send events to the transaction, which are handled in the start() method
func (trans Sitrans) Event(msg *message.SIPMessage) {
	if trans.state == terminated {
		return
	}

	trans.transc <- msg // Push the event to the transc channel for processing
}

// Start initiates the transaction processing by running the start method in a goroutine
func (trans *Sitrans) Start() {
	log.Trace().Str("transaction_id", trans.id.String()).Msg("Starting INVITE server transaction")
	// Start the timer for provisional responses (Timer Prv)
	trans.timers[timer_prv].Start(tiprv_dur)

	// Call the core callback with the original message
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", trans.message).Msg("Initial action: Sending request to core")
	call_core_callback(trans, trans.message)

	// Main event loop for processing the transaction
	for {
		// Wait for events from the transc channel or from the timers
		select {
		case msg := <-trans.transc: // Event received from the transc channel
			trans.handle_msg(msg)
		case <-trans.timers[timer_prv].Chan(): // Timer Prv has expired
			trans.handle_timer(timer_prv)
		case <-trans.timers[timer_g].Chan(): // Timer G has expired
			trans.handle_timer(timer_g)
		case <-trans.timers[timer_h].Chan(): // Timer H has expired
			trans.handle_timer(timer_h)
		case <-trans.timers[timer_i].Chan(): // Timer I has expired
			trans.handle_timer(timer_i)
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
func (trans *Sitrans) handle_timer(timer timer) {
	log.Trace().Str("transaction_id", trans.id.String()).Str("timer", timer.String()).Msg("Handling timer event")
	// Handle the event based on which timer expired
	if timer == timer_h {
		// Timer H expired: Transaction is terminated due to timeout
		trans.state = terminated
		call_term_callback(trans, transaction.TIMEOUT)
	} else if timer == timer_prv && trans.state == proceeding {
		// Timer Prv expired: Send 100 TRYING response
		trying100 := message.MakeGenericResponse(100, "TRYING", trans.message)
		call_transport_callback(trans, trying100)
	} else if timer == timer_g && trans.state == completed {
		// Timer G expired: Retransmit the last response and restart Timer G with adjusted duration
		trans.timers[timer_g].Start(min(2*trans.timers[timer_g].Duration(), t2))
		call_transport_callback(trans, trans.last_res)
	} else if timer == timer_i && trans.state == confirmed {
		// Timer I expired: Move to the terminated state (final step)
		trans.state = terminated
		call_term_callback(trans, transaction.NORMAL)
	}
}

// handle_msg processes received SIP messages (requests or responses)
func (trans *Sitrans) handle_msg(msg *message.SIPMessage) {
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", msg).Msg("Handling message event")

	// Handle incoming SIP request (e.g., INVITE or ACK)
	if msg.Request != nil {
		if msg.Request.Method == "ACK" && trans.state == completed {
			// Received an ACK response in the "completed" state: Move to "confirmed"
			trans.timers[timer_g].Stop()
			trans.timers[timer_h].Stop()
			trans.timers[timer_i].Start(tii_dur)
			trans.state = confirmed
		}

		if msg.Request.Method == "INVITE" && trans.state == completed {
			// Received an INVITE request in the "completed" state: Retransmit the last response
			call_transport_callback(trans, trans.last_res)
		}

		return // Return early if the message is an ACK or INVITE
	}

	// Handle SIP responses based on their status code
	status_code := msg.Response.StatusCode
	if status_code >= 100 && status_code < 200 && trans.state == proceeding {
		// Provisional responses (1xx): Send them to the transport layer
		trans.timers[timer_prv].Stop() // Stop Timer Prv if a provisional response is received
		call_transport_callback(trans, msg)
	} else if status_code >= 200 && status_code <= 300 && trans.state == proceeding {
		// Final 2xx responses: Transition to terminated state
		trans.state = terminated
		call_term_callback(trans, transaction.NORMAL)
		call_transport_callback(trans, msg)
	} else if status_code > 300 {
		// Error responses (3xx-6xx): Transition to "completed" state
		trans.timers[timer_g].Start(tig_dur) // Start Timer G for retransmissions
		trans.timers[timer_h].Start(tih_dur) // Start Timer H for retransmissions
		trans.last_res = msg                 // Save the last response
		trans.state = completed
		call_transport_callback(trans, msg)
	}
}

// call_core_callback invokes the core callback to handle transaction-related events
func call_core_callback(sitrans *Sitrans, message *message.SIPMessage) {
	log.Trace().Str("transaction_id", sitrans.id.String()).Interface("message", message).Msg("Invoking core callback")
	sitrans.core_cb(sitrans.transport, message) // Call the core callback
}

// call_transport_callback invokes the transport callback to send or receive messages
func call_transport_callback(sitrans *Sitrans, message *message.SIPMessage) {
	log.Trace().Str("transaction_id", sitrans.id.String()).Interface("message", message).Msg("Invoking transport callback")
	if !sitrans.trpt_cb(sitrans.transport, message) { // Call the transport callback
		sitrans.state = terminated
		call_term_callback(sitrans, transaction.ERROR)
	}
}

func call_term_callback(sitrans *Sitrans, reason transaction.TERM_REASON) {
	log.Trace().Str("transaction_id", sitrans.id.String()).Interface("termination_reason", reason).Msg("Invoking termination callback")
	sitrans.term_cb(sitrans.id, reason)
}
