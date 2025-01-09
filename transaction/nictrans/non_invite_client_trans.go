package nictrans

import (
	"gossip/message"
	"gossip/transaction"
	"gossip/util"
)

// Timer constants
const (
	t1 = 500  // Timer T1 duration (500ms)
	t2 = 4000 // Timer T2 duration (4000ms)
	t4 = 5000 // Timer T4 duration (5000ms)
	tif_dur = 64 * t1 // Timer F duration (64*T1)
	tie_dur = t1
	tik_dur = t4
)

// Timer constants (Indexes)
const (
	timer_e = iota
	timer_f
	timer_k
)

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
	state   state                                     // Current state of the transaction
	message *message.SIPMessage                       // The SIP message associated with the transaction
	timers  [3]util.Timer                             // Timers used for retransmission and timeout
	transc  chan util.Event                           // Channel for receiving events like timeouts or messages
	trpt_cb func(transaction.Transaction, util.Event) // Callback for transport layer
	core_cb func(transaction.Transaction, util.Event) // Callback for core layer
}

// Make creates and initializes a new NIctrans instance with the given message and callbacks
func Make(
	message *message.SIPMessage,
	transport_callback func(transaction.Transaction, util.Event),
	core_callback func(transaction.Transaction, util.Event),
) *NIctrans {
	// Create new timers for the transaction state machine
	timerE := util.NewTimer() // Retransmission Timer
	timerF := util.NewTimer() // Timeout Timer
	timerK := util.NewTimer() // Timer for completed state

	// Return a new NIctrans instance with the provided parameters
	return &NIctrans{
		message: message.DeepCopy(),
		transc:  make(chan util.Event),                 // Channel for event communication
		timers:  [3]util.Timer{timerE, timerF, timerK}, // Initialize the timers array
		state:   trying,                                // Initial state is "trying"
		trpt_cb: transport_callback,                    // Transport callback for message transmission
		core_cb: core_callback,                         // Core callback for message handling in the application logic
	}
}

// Event is used to send events to the transaction, which are handled in the start() method
func (trans NIctrans) Event(event util.Event) {
	trans.transc <- event // Push the event to the transc channel for processing
}

// Start initiates the transaction processing by running the start method in a goroutine
func (trans *NIctrans) Start() {
	go trans.start() // Start the transaction processing asynchronously
}

// start begins the transaction state machine, listening for events and handling state transitions
func (trans *NIctrans) start() {
	// Start Timer F (64*T1)
	trans.timers[timer_f].Start(tif_dur)

	// Send the request to the transport layer
	call_transport_callback(trans, util.Event{Type: util.MESS, Data: trans.message.DeepCopy()})

	// Set Timer E for retransmission to fire at T1
	trans.timers[timer_e].Start(tie_dur)

	var ev util.Event
	for {
		// Wait for events (e.g., timeouts, received messages)
		select {
		case ev = <-trans.transc:
		case <-trans.timers[timer_e].Chan(): // Timer E expires
			ev = util.Event{Type: util.TIMEOUT, Data: timer_e}
		case <-trans.timers[timer_f].Chan(): // Timer F expires
			ev = util.Event{Type: util.TIMEOUT, Data: timer_f}
		case <-trans.timers[timer_k].Chan(): // Timer K expires
			ev = util.Event{Type: util.TIMEOUT, Data: timer_k}
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
func (trans *NIctrans) handle_event(ev util.Event) {
	switch ev.Type {
	case util.TIMEOUT:
		trans.handle_timeout(ev)
	case util.MESS:
		trans.handle_message(ev)
	default:
		return
	}
}

// handle_timeout processes timeout events (Timer E, F, K)
func (trans *NIctrans) handle_timeout(ev util.Event) {
	if ev.Data == timer_f && trans.state < completed {  // Timer F expired, inform TU of timeout and terminate transaction
        call_core_callback(trans, util.Event{Type: util.TIMEOUT, Data: trans.message.DeepCopy()})
        trans.state = terminated
    } else if ev.Data == timer_e && trans.state < completed {  // Timer E expired in trying, proceeding state, retransmit request
        call_transport_callback(trans, util.Event{Type: util.MESS, Data: trans.message.DeepCopy()})
        trans.timers[timer_e].Start(min(trans.timers[timer_e].Duration() * 2, t2))  // Double Timer E duration
    } else if ev.Data == timer_k && trans.state == completed {  // Timer D expired in completed state, terminate transaction
        trans.state = terminated
    }
}

// handle_message processes received SIP messages (responses)
func (trans *NIctrans) handle_message(ev util.Event) {
	msg, ok := ev.Data.(*message.SIPMessage)
	if !ok || msg.Response == nil {
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
		trans.timers[timer_k].Start(tik_dur)
		trans.state = completed
	}
}

// call_core_callback invokes the core callback with the provided event
func call_core_callback(trans *NIctrans, ev util.Event) {
	trans.core_cb(trans, ev)
}

// call_transport_callback invokes the transport callback with the provided event
func call_transport_callback(trans *NIctrans, ev util.Event) {
	trans.trpt_cb(trans, ev)
}
