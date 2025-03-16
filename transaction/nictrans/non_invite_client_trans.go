package nictrans

import (
	"gossip/message"
	"gossip/transaction"
	"gossip/transport"

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

// Timer constants
const (
	t1      = 500     // Timer T1 duration (500ms)
	t2      = 4000    // Timer T2 duration (4000ms)
	t4      = 5000    // Timer T4 duration (5000ms)
	tif_dur = 64 * t1 // Timer F duration (64*T1)
	tie_dur = t1
	tik_dur = t4
)

type timer int

// Timer constants (Indexes)
const (
	timer_e = iota
	timer_f
	timer_k
)

func (t timer) String() string {
	switch t {
	case timer_e:
		return "Timer E"
	case timer_f:
		return "Timer F"
	case timer_k:
		return "Timer K"
	default:
		return "Unknown"
	}
}

// Define the states for the Non-Invite Client Transaction
type state int

const (
	trying     state = iota // The transaction is waiting for a response
	proceeding              // The transaction has received a provisional response
	completed               // The transaction has received a final response
	terminated              // The transaction has been terminated
)

// NIctrans represents the state machine for a Non-Invite Client Transaction
type NIctrans struct {
	id        transaction.TransID                                  // Transaction ID
	state     state                                                // Current state of the transaction
	message   *message.SIPMessage                                  // The SIP message associated with the transaction
	transport *transport.Transport                                 // Transport layer for sending and receiving messages
	timers    [3]transaction.Timer                                 // Timers used for retransmission and timeout
	transc    chan *message.SIPMessage                             // Channel for receiving events like timeouts or messages
	trpt_cb   func(*transport.Transport, *message.SIPMessage) bool // Callback for transport layer
	core_cb   func(*transport.Transport, *message.SIPMessage)      // Callback for core layer
	term_cb   func(transaction.TransID, transaction.TERM_REASON)
}

// Make creates and initializes a new NIctrans instance with the given message and callbacks
func Make(
	id transaction.TransID, // Transaction ID
	msg *message.SIPMessage,
	transport *transport.Transport,
	core_callback func(*transport.Transport, *message.SIPMessage),
	transport_callback func(*transport.Transport, *message.SIPMessage) bool,
	term_callback func(transaction.TransID, transaction.TERM_REASON),
) *NIctrans {
	// Create new timers for the transaction state machine
	timerE := transaction.NewTimer() // Retransmission Timer
	timerF := transaction.NewTimer() // Timeout Timer
	timerK := transaction.NewTimer() // Timer for completed state

	log.Trace().Str("transaction_id", id.String()).Interface("message", msg).Interface("transport", transport).Msg("Creating new Non-Invite client transaction")
	// Return a new NIctrans instance with the provided parameters
	return &NIctrans{
		id:        id, // Set transaction ID
		message:   msg,
		transport: transport,
		transc:    make(chan *message.SIPMessage, 5),            // Channel for event communication
		timers:    [3]transaction.Timer{timerE, timerF, timerK}, // Initialize the timers array
		state:     trying,                                       // Initial state is "trying"
		trpt_cb:   transport_callback,                           // Transport callback for message transmission
		core_cb:   core_callback,                                // Core callback for message handling in the application logic
		term_cb:   term_callback,                                // Termination callback for the transaction
	}
}

// Event is used to send events to the transaction, which are handled in the start() method
func (trans NIctrans) Event(msg *message.SIPMessage) {
	if trans.state == terminated {
		return
	}

	trans.transc <- msg // Push the event to the transc channel for processing
}

// Start initiates the transaction processing by running the start method in a goroutine
func (trans *NIctrans) Start() {
	log.Trace().Str("transaction_id", trans.id.String()).Msg("Starting Non-Invite client transaction")
	// Start Timer F (64*T1)
	trans.timers[timer_f].Start(tif_dur)

	// Send the request to the transport layer
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", trans.message).Msg("Initial action: Sending request")
	call_transport_callback(trans, trans.message)

	// Set Timer E for retransmission to fire at T1
	trans.timers[timer_e].Start(tie_dur)

	for {
		// Wait for events (e.g., timeouts, received messages)
		select {
		case msg := <-trans.transc:
			trans.handle_message(msg)
		case <-trans.timers[timer_e].Chan(): // Timer E expires
			trans.handle_timer(timer_e)
		case <-trans.timers[timer_f].Chan(): // Timer F expires
			trans.handle_timer(timer_f)
		case <-trans.timers[timer_k].Chan(): // Timer K expires
			trans.handle_timer(timer_k)
		}

		// If the state is terminated, exit the loop
		if trans.state == terminated {
			log.Trace().Str("transaction_id", trans.id.String()).Msg("Transaction terminated")
			close(trans.transc) // Close the event channel when the transaction ends
			break
		}
	}
}

// handle_timeout processes timeout events (Timer E, F, K)
func (trans *NIctrans) handle_timer(timer timer) {
	log.Trace().Str("transaction_id", trans.id.String()).Str("timer", timer.String()).Msg("Handling timer event")
	if timer == timer_f && trans.state < completed { // Timer F expired, inform TU of timeout and terminate transaction
		trans.state = terminated
		call_term_callback(trans, transaction.TIMEOUT)
	} else if timer == timer_e && trans.state < completed { // Timer E expired in trying, proceeding state, retransmit request
		trans.timers[timer_e].Start(min(trans.timers[timer_e].Duration()*2, t2)) // Double Timer E duration
		call_transport_callback(trans, trans.message)
	} else if timer == timer_k && trans.state == completed { // Timer D expired in completed state, terminate transaction
		trans.state = terminated
		call_term_callback(trans, transaction.NORMAL)
	}
}

// handle_message processes received SIP messages (responses)
func (trans *NIctrans) handle_message(msg *message.SIPMessage) {
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", msg).Msg("Handling message event")
	if msg.Response == nil {
		return
	}

	status_code := msg.Response.StatusCode
	if status_code >= 100 && status_code < 200 {
		// Provisional response (1xx): Move to Proceeding state
		trans.state = proceeding
		call_core_callback(trans, msg)
	} else if status_code >= 200 && status_code <= 699 {
		// Final response (2xx-6xx): Move to Completed state
		trans.timers[timer_k].Start(tik_dur)
		trans.state = completed
		call_core_callback(trans, msg)
	}
}

// call_core_callback invokes the core callback with the provided event
func call_core_callback(trans *NIctrans, msg *message.SIPMessage) {
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", msg).Msg("Invoking core callback")
	trans.core_cb(trans.transport, msg)
}

// call_transport_callback invokes the transport callback with the provided event
func call_transport_callback(trans *NIctrans, msg *message.SIPMessage) {
	log.Trace().Str("transaction_id", trans.id.String()).Interface("message", msg).Msg("Invoking transport callback")
	if !trans.trpt_cb(trans.transport, msg) {
		call_term_callback(trans, transaction.ERROR)
		trans.state = terminated
	}
}

// call_term_callback invokes the termination callback with the provided reason
func call_term_callback(trans *NIctrans, reason transaction.TERM_REASON) {
	log.Trace().Str("transaction_id", trans.id.String()).Interface("termination_reason", reason).Msg("Invoking termination callback")
	trans.term_cb(trans.id, reason)
}
