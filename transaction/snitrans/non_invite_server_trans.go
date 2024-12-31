package ctnoninviteserver

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

const timerJDuration = 64 * T1 // Timer J duration (64*T1)

// Timer constants (Indexes)
const (
	timerJ = iota
)

// Define the states for the Non-Invite Server Transaction
type state int

const (
	trying state = iota   // The transaction is waiting for a response
	proceeding           // The transaction has sent a provisional response
	completed            // The transaction has sent a final response
	terminated           // The transaction has been terminated
)

// Ctnoninviteserver represents the state machine for a Non-Invite Server Transaction
type Ctnoninviteserver struct {
	state    state               // Current state of the transaction
	message  *message.SIPMessage // The SIP message associated with the transaction
	timers   [1]util.Timer       // Timer J for retransmission
	transc   chan event.Event    // Channel for receiving events like timeouts or messages
	trpt_cb  func(transaction.Transaction, event.Event) // Callback for transport layer
	core_cb  func(transaction.Transaction, event.Event) // Callback for core layer
}

// Make creates and initializes a new Ctnoninviteserver instance with the given message and callbacks
func Make(
	message *message.SIPMessage,
	transport_callback func(transaction.Transaction, event.Event),
	core_callback func(transaction.Transaction, event.Event),
) *Ctnoninviteserver {
	// Create new timers for the transaction state machine
	timerJ := util.NewTimer() // Timer J for the Completed state

	// Return a new Ctnoninviteserver instance with the provided parameters
	return &Ctnoninviteserver{
		message: message,
		transc:  make(chan event.Event),  // Channel for event communication
		timers:  [1]util.Timer{timerJ},    // Initialize Timer J
		state:   trying, // Initial state is "trying"
		trpt_cb: transport_callback, // Transport callback for message transmission
		core_cb: core_callback, // Core callback for message handling in the application logic
	}
}

// Event is used to send events to the transaction, which are handled in the start() method
func (trans Ctnoninviteserver) Event(event event.Event) {
	trans.transc <- event // Push the event to the transc channel for processing
}

// Start initiates the transaction processing by running the start method in a goroutine
func (trans *Ctnoninviteserver) Start() {
	go trans.start() // Start the transaction processing asynchronously
}

// start begins the transaction state machine, listening for events and handling state transitions
func (trans *Ctnoninviteserver) start() {
	// Send the request to the core for handling
	call_core_callback(trans, event.Event{Type: event.MESS, Data: trans.message})

	// Listen for events (e.g., timeouts, received messages)
	var ev event.Event
	for {
		// Wait for events (e.g., timeouts, received messages)
		select {
		case ev = <-trans.transc:
		case <-trans.timers[timerJ].Chan(): // Timer J expires
			ev = event.Event{Type: event.TIMEOUT, Data: timerJ}
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
func (trans *Ctnoninviteserver) handle_event(ev event.Event) {
	switch ev.Type {
	case event.TIMEOUT:
		trans.handle_timeout(ev)
	case event.MESS:
		trans.handle_message(ev)
	default:
		return
	}
}

// handle_timeout processes timeout events (Timer J)
func (trans *Ctnoninviteserver) handle_timeout(ev event.Event) {
	switch ev.Data {
	case timerJ:
		// Timer J expired: Move to Terminated state
		trans.state = terminated
	}
}

// handle_message processes received SIP messages (requests and responses)
func (trans *Ctnoninviteserver) handle_message(ev event.Event) {
	msg, ok := ev.Data.(*message.SIPMessage)
	if !ok {
		return
	}

	status_code := msg.Response.StatusCode
	if status_code >= 100 && status_code < 200 {
		// Provisional response (1xx): Move to Proceeding state
		call_core_callback(trans, ev)
		trans.state = proceeding

		// Send provisional response to transport layer
		call_transport_callback(trans, event.Event{Type: event.MESS, Data: msg})

	} else if status_code >= 200 && status_code <= 699 {
		// Final response (2xx-6xx): Move to Completed state
		call_core_callback(trans, ev)
		trans.state = completed

		// Send final response to transport layer
		call_transport_callback(trans, event.Event{Type: event.MESS, Data: msg})

		// Set Timer J for retransmissions (if unreliable transport)
		trans.timers[timerJ].Start(timerJDuration)
	}
}

// call_core_callback invokes the core callback with the provided event
func call_core_callback(ctnoninviteserver *Ctnoninviteserver, ev event.Event) {
	go ctnoninviteserver.core_cb(ctnoninviteserver, ev)
}

// call_transport_callback invokes the transport callback with the provided event
func call_transport_callback(ctnoninviteserver *Ctnoninviteserver, ev event.Event) {
	go ctnoninviteserver.trpt_cb(ctnoninviteserver, ev)
}
