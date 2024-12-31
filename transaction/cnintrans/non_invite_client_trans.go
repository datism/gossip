package ctnoninvite

import (
	"gossip/event"
	"gossip/message"
	"gossip/transaction"
	"gossip/util"
)

// Timer constants
const (
	T1 = 500     // Timer T1 duration (500ms)
	T2 = 4000    // Timer T2 duration (4000ms)
	T4 = 5000    // Timer T4 duration (5000ms)
)

const timerFDuration = 64 * T1 // Timer F duration (64*T1)

// Timer constants (Indexes)
const (
	timerE = iota
	timerF
	timerK
)

// Define the states for the Non-Invite Client Transaction
type state int

const (
	trying state = iota   // The transaction is waiting for a response
	proceeding           // The transaction has received a provisional response
	completed            // The transaction has received a final response
	terminated           // The transaction has been terminated
)

// Ctnoninvite represents the state machine for a Non-Invite Client Transaction
type Ctnoninvite struct {
	state    state               // Current state of the transaction
	message  *message.SIPMessage // The SIP message associated with the transaction
	timers   [3]util.Timer       // Timers used for retransmission and timeout
	transc   chan event.Event    // Channel for receiving events like timeouts or messages
	trpt_cb  func(transaction.Transaction, event.Event) // Callback for transport layer
	core_cb  func(transaction.Transaction, event.Event) // Callback for core layer
}

// Make creates and initializes a new Ctnoninvite instance with the given message and callbacks
func Make(
	message *message.SIPMessage,
	transport_callback func(transaction.Transaction, event.Event),
	core_callback func(transaction.Transaction, event.Event),
) *Ctnoninvite {
	// Create new timers for the transaction state machine
	timerE := util.NewTimer() // Retransmission Timer
	timerF := util.NewTimer() // Timeout Timer
	timerK := util.NewTimer() // Timer for completed state

	// Return a new Ctnoninvite instance with the provided parameters
	return &Ctnoninvite{
		message: message,
		transc:  make(chan event.Event),  // Channel for event communication
		timers:  [3]util.Timer{timerE, timerF, timerK}, // Initialize the timers array
		state:   trying, // Initial state is "trying"
		trpt_cb: transport_callback, // Transport callback for message transmission
		core_cb: core_callback, // Core callback for message handling in the application logic
	}
}

// Event is used to send events to the transaction, which are handled in the start() method
func (trans Ctnoninvite) Event(event event.Event) {
	trans.transc <- event // Push the event to the transc channel for processing
}

// Start initiates the transaction processing by running the start method in a goroutine
func (trans *Ctnoninvite) Start() {
	go trans.start() // Start the transaction processing asynchronously
}

// start begins the transaction state machine, listening for events and handling state transitions
func (trans *Ctnoninvite) start() {
	// Start Timer F (64*T1)
	trans.timers[timerF].Start(timerFDuration)

	// Send the request to the transport layer
	call_transport_callback(trans, event.Event{Type: event.MESS, Data: trans.message})

	// Set Timer E for retransmission to fire at T1
	trans.timers[timerE].Start(T1)

	// Call the core callback with the original message
	call_core_callback(trans, event.Event{Type: event.MESS, Data: trans.message})

	var ev event.Event
	for {
		// Wait for events (e.g., timeouts, received messages)
		select {
		case ev = <-trans.transc:
		case <-trans.timers[timerE].Chan(): // Timer E expires
			ev = event.Event{Type: event.TIMEOUT, Data: timerE}
		case <-trans.timers[timerF].Chan(): // Timer F expires
			ev = event.Event{Type: event.TIMEOUT, Data: timerF}
		case <-trans.timers[timerK].Chan(): // Timer K expires
			ev = event.Event{Type: event.TIMEOUT, Data: timerK}
		}

		// Handle events (timeouts or SIP messages)
		trans.handle_event(ev)

		// If the state is terminated, exit the loop
		if trans.state == terminated {
			break
		}
	}
}

// handle_event processes different types of events: timeouts and messages
func (trans *Ctnoninvite) handle_event(ev event.Event) {
	switch ev.Type {
	case event.TIMEOUT:
		trans.handle_timeout(ev)
	case event.MESS:
		trans.handle_message(ev)
	default:
		return
	}
}

// handle_timeout processes timeout events (Timer E, F, K)
func (trans *Ctnoninvite) handle_timeout(ev event.Event) {
	switch ev.Data {
	case timerE:
		// Timer E expired: Retransmit the request, reset Timer E
		call_transport_callback(trans, event.Event{Type: event.MESS, Data: trans.message})
		trans.timers[timerE].Start(min(2*T1, T2))

	case timerF:
		// Timer F expired: Timeout, transition to Terminated
		call_core_callback(trans, event.Event{Type: event.TIMEOUT, Data: trans.message})
		trans.state = terminated

	case timerK:
		// Timer K expired: Move to Terminated state (for unreliable transport)
		trans.state = terminated
	}
}

// handle_message processes received SIP messages (responses)
func (trans *Ctnoninvite) handle_message(ev event.Event) {
	msg, ok := ev.Data.(*message.SIPMessage)
	if !ok {
		return
	}

	status_code := msg.Response.StatusCode
	if status_code >= 100 && status_code < 200 {
		// Provisional response (1xx): Move to Proceeding state
		call_core_callback(trans, ev)
		trans.state = proceeding
	} else if status_code >= 200 && status_code <= 699 {
		// Final response (2xx-6xx): Move to Completed state
		call_core_callback(trans, ev)
		trans.state = completed

		// Set Timer K for retransmissions (if unreliable transport)
		if is_unreliable_transport() {
			trans.timers[timerK].Start(T4)
		}
	}
}

// call_core_callback invokes the core callback with the provided event
func call_core_callback(ctnoninvite *Ctnoninvite, ev event.Event) {
	go ctnoninvite.core_cb(ctnoninvite, ev)
}

// call_transport_callback invokes the transport callback with the provided event
func call_transport_callback(ctnoninvite *Ctnoninvite, ev event.Event) {
	go ctnoninvite.trpt_cb(ctnoninvite, ev)
}
