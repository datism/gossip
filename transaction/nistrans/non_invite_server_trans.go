package nistrans

import (
	"gossip/message"
	"gossip/transaction"
	"gossip/util"
)

// Timer constants
const t1 = 500

const tij_dur = 64 * t1 // Timer J duration (64*T1)

// Timer constants (Indexes)
const (
	timer_j = iota
)

// Define the states for the Non-Invite Server Transaction
type state int

const (
	trying     state = iota // The transaction is waiting for a response
	proceeding              // The transaction has sent a provisional response
	completed               // The transaction has sent a final response
	terminated              // The transaction has been terminated
)

// NIstrans represents the state machine for a Non-Invite Server Transaction
type NIstrans struct {
	state    state                                     // Current state of the transaction
	message  *message.SIPMessage                       // The SIP message associated with the transaction
	last_res *message.SIPMessage                       // The last response received
	timers   [1]util.Timer                             // Timer J for retransmission
	transc   chan util.Event                           // Channel for receiving events like timeouts or messages
	trpt_cb  func(transaction.Transaction, util.Event) // Callback for transport layer
	core_cb  func(transaction.Transaction, util.Event) // Callback for core layer
}

// Make creates and initializes a new NIstrans instance with the given message and callbacks
func Make(
	message *message.SIPMessage,
	transport_callback func(transaction.Transaction, util.Event),
	core_callback func(transaction.Transaction, util.Event),
) *NIstrans {
	// Create new timers for the transaction state machine
	timerJ := util.NewTimer() // Timer J for the Completed state

	// Return a new NIstrans instance with the provided parameters
	return &NIstrans{
		message: message.DeepCopy(),
		transc:  make(chan util.Event), // Channel for event communication
		timers:  [1]util.Timer{timerJ}, // Initialize Timer J
		state:   trying,                // Initial state is "trying"
		trpt_cb: transport_callback,    // Transport callback for message transmission
		core_cb: core_callback,         // Core callback for message handling in the application logic
	}
}

// Event is used to send events to the transaction, which are handled in the start() method
func (trans NIstrans) Event(event util.Event) {
	trans.transc <- event // Push the event to the transc channel for processing
}

// Start initiates the transaction processing by running the start method in a goroutine
func (trans *NIstrans) Start() {
	trans.start() // Start the transaction processing asynchronously
}

// start begins the transaction state machine, listening for events and handling state transitions
func (trans *NIstrans) start() {
	// Send the request to the core for handling
	call_core_callback(trans, util.Event{Type: util.MESS, Data: trans.message.DeepCopy()})

	trans.timers[timer_j].Start(tij_dur)

	// Listen for events (e.g., timeouts, received messages)
	var ev util.Event
	for {
		// Wait for events (e.g., timeouts, received messages)
		select {
		case ev = <-trans.transc:
		case <-trans.timers[timer_j].Chan(): // Timer J expires
			ev = util.Event{Type: util.TIMEOUT, Data: timer_j}
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
func (trans *NIstrans) handle_event(ev util.Event) {
	switch ev.Type {
	case util.TIMEOUT:
		trans.handle_timeout(ev)
	case util.MESS:
		trans.handle_message(ev)
	default:
		return
	}
}

// handle_timeout processes timeout events (Timer J)
func (trans *NIstrans) handle_timeout(ev util.Event) {
	if ev.Data == timer_j && trans.state == completed {
		trans.state = terminated
	}
}

// handle_message processes received SIP messages (requests and responses)
func (trans *NIstrans) handle_message(ev util.Event) {
	msg, ok := ev.Data.(*message.SIPMessage)
	if !ok {
		return
	}

	if msg.Request != nil {
		if trans.state == proceeding || trans.state == completed {
			call_transport_callback(trans, util.Event{Type: util.MESS, Data: trans.last_res.DeepCopy()})
		}

		return
	}

	msg.Transport = trans.message.Transport
	status_code := msg.Response.StatusCode
	if status_code >= 100 && status_code < 200 {
		// Provisional response (1xx): Move to Proceeding state
		call_core_callback(trans, util.Event{Type: util.MESS, Data: msg.DeepCopy()})
		trans.last_res = msg
		trans.state = proceeding

		// Send provisional response to transport layer
		call_transport_callback(trans, util.Event{Type: util.MESS, Data: msg.DeepCopy()})

	} else if status_code >= 200 && status_code <= 699 {
		// Final response (2xx-6xx): Move to Completed state
		call_core_callback(trans, util.Event{Type: util.MESS, Data: msg.DeepCopy()})
		trans.last_res = msg
		trans.state = completed

		// Send final response to transport layer
		call_transport_callback(trans, util.Event{Type: util.MESS, Data: msg.DeepCopy()})

		// Set Timer J for retransmissions (if unreliable transport)
		trans.timers[timer_j].Start(tij_dur)
	}
}

// call_core_callback invokes the core callback with the provided event
func call_core_callback(trans *NIstrans, ev util.Event) {
	trans.core_cb(trans, ev)
}

// call_transport_callback invokes the transport callback with the provided event
func call_transport_callback(trans *NIstrans, ev util.Event) {
	trans.trpt_cb(trans, ev)
}
