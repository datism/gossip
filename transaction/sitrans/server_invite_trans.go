package sitrans

import (
	"gossip/event"
	"gossip/message"
	"gossip/transaction"
	"gossip/util"
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
const t1 = 500    // Timer T1 duration (500ms)
const t2 = 4000   // Timer T2 duration (4000ms)
const t4 = 5000   // Timer T4 duration (5000ms)

// Define durations for specific timers used in the state machine
const tiprv_dur = 200    // Provisional response timer duration (200ms)
const tig_dur = t1       // Timer G duration (T1)
const tih_dur = 64 * t1  // Timer H duration (64 * T1)
const tii_dur = t4       // Timer I duration (T4)

// Enum-like constants for different timer indices
const (
	timer_prv = iota
	timer_g
	timer_h
	timer_i
)

// Define the states for the transaction
type state int

const (
	proceeding state = iota   // In this state, the server is processing the request
	completed                // Transaction is completed, waiting for ACK or further responses
	confirmed                // ACK received, transaction confirmed
	terminated               // The transaction is finished (either successfully or failed)
)

// Sitrans represents the state machine for an INVITE transaction
type Sitrans struct {
	state    state               // Current state of the transaction
	message  *message.SIPMessage // The SIP message associated with the transaction
	last_res *message.SIPMessage // The last response received
	timers   [4]util.Timer       // List of timers used for managing retransmissions and timeouts
	transc   chan event.Event    // Channel for receiving events like timeouts or messages
	trpt_cb  func(transaction.Transaction, event.Event) // Callback for transport layer
	core_cb  func(transaction.Transaction, event.Event) // Callback for core layer (application logic)
}

// Make creates and initializes a new Sitrans instance with the given message and callbacks
func Make(
	message *message.SIPMessage,
	transport_callback func(transaction.Transaction, event.Event),
	core_callback func(transaction.Transaction, event.Event),
) *Sitrans {
	// Create new timers for the transaction state machine
	timerprv := util.NewTimer()
	timerg := util.NewTimer()
	timerdh := util.NewTimer()
	timeri := util.NewTimer()

	// Return a new Sitrans instance with the provided parameters
	return &Sitrans{
		message: message,
		transc:  make(chan event.Event),  // Channel for event communication
		timers:  [4]util.Timer{timerprv, timerg, timerdh, timeri}, // Initialize the timers array
		state:   proceeding, // Initial state is "proceeding"
		trpt_cb: transport_callback, // Transport callback for message transmission
		core_cb: core_callback, // Core callback for message handling in the application logic
	}
}

// Event is used to send events to the transaction, which are handled in the start() method
func (trans Sitrans) Event(event event.Event) {
	trans.transc <- event // Push the event to the transc channel for processing
}

// Start initiates the transaction processing by running the start method in a goroutine
func (trans *Sitrans) Start() {
	go trans.start() // Start the transaction processing asynchronously
}

// start begins the transaction state machine, listening for events and handling state transitions
func (trans *Sitrans) start() {
	// Start the timer for provisional responses (Timer Prv)
	trans.timers[timer_prv].Start(tiprv_dur)
	
	// Call the core callback with the original message
	call_core_callback(trans, event.Event{Type: event.MESS, Data: trans.message})

	// Define a variable to hold incoming events
	var ev event.Event

	// Main event loop for processing the transaction
	for {
		// Wait for events from the transc channel or from the timers
		select {
		case ev = <-trans.transc: // Event received from the transc channel
		case <-trans.timers[timer_prv].Chan(): // Timer Prv has expired
			ev = event.Event{Type: event.TIMEOUT, Data: timer_prv}
		case <-trans.timers[timer_g].Chan(): // Timer G has expired
			ev = event.Event{Type: event.TIMEOUT, Data: timer_g}
		case <-trans.timers[timer_h].Chan(): // Timer H has expired
			ev = event.Event{Type: event.TIMEOUT, Data: timer_h}
		case <-trans.timers[timer_i].Chan(): // Timer I has expired
			ev = event.Event{Type: event.TIMEOUT, Data: timer_i}
		}

		// Handle the received event
		trans.handle_event(ev)

		// If the state is terminated, break the loop and stop the transaction
		if trans.state == terminated {
			break
		}
	}
}

// handle_event processes different types of events: timeouts and messages
func (ctx *Sitrans) handle_event(ev event.Event) {
	switch ev.Type {
	case event.TIMEOUT: // Handle timeout events (timer expirations)
		ctx.handle_timer(ev)
	case event.MESS: // Handle received messages (SIP responses)
		ctx.handle_msg(ev)
	default:
		return // If the event type is not recognized, do nothing
	}
}

// handle_timer processes events triggered by timer expirations
func (trans *Sitrans) handle_timer(ev event.Event) {
	// Handle the event based on which timer expired
	if ev.Data == timer_h {
		// Timer H expired: Transaction is terminated due to timeout
		call_core_callback(trans, event.Event{Type: event.TIMEOUT, Data: trans.message})
		trans.state = terminated
	} else if ev.Data == timer_prv && trans.state == proceeding {
		// Timer Prv expired: Send 100 TRYING response
		trying100 := message.MakeGenericResponse(100, "TRYING", trans.message)
		call_transport_callback(trans, event.Event{Type: event.MESS, Data: trying100})
	} else if ev.Data == timer_g && trans.state == completed {
		// Timer G expired: Retransmit the last response and restart Timer G with adjusted duration
		call_transport_callback(trans, event.Event{Type: event.MESS, Data: trans.last_res})
		trans.timers[timer_g].Start(min(2*trans.timers[timer_g].Duration(), t2))
	} else if ev.Data == timer_i && trans.state == confirmed {
		// Timer I expired: Move to the terminated state (final step)
		trans.state = terminated
	}
}

// handle_msg processes received SIP messages (requests or responses)
func (trans *Sitrans) handle_msg(ev event.Event) {
	// Extract the SIP message from the event
	msg, ok := ev.Data.(*message.SIPMessage)
	if !ok {
		return // If the message is not valid, do nothing
	}

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
			call_transport_callback(trans, event.Event{Type: event.MESS, Data: trans.last_res})
		}

		return // Return early if the message is an ACK or INVITE
	}

	// Handle SIP responses based on their status code
	status_code := msg.Response.StatusCode
	if status_code >= 100 && status_code < 200 && trans.state == proceeding {
		// Provisional responses (1xx): Send them to the transport layer
		trans.timers[timer_prv].Stop() // Stop Timer Prv if a provisional response is received
		call_transport_callback(trans, event.Event{Type: event.MESS, Data: msg})
	} else if status_code >= 200 && status_code <= 300 && trans.state == proceeding {
		// Final 2xx responses: Transition to terminated state
		call_transport_callback(trans, event.Event{Type: event.MESS, Data: msg})
		trans.state = terminated
	} else if status_code > 300 {
		// Error responses (3xx-6xx): Transition to "completed" state
		call_transport_callback(trans, event.Event{Type: event.MESS, Data: msg})
		trans.timers[timer_g].Start(tig_dur) // Start Timer G for retransmissions
		trans.timers[timer_h].Start(tih_dur) // Start Timer H for retransmissions
		trans.last_res = msg // Save the last response
		trans.state = completed
	}
}

// call_core_callback invokes the core callback with the provided event
func call_core_callback(sitrans *Sitrans, ev event.Event) {
	go sitrans.core_cb(sitrans, ev) // Call the core callback asynchronously
}

// call_transport_callback invokes the transport callback with the provided event
func call_transport_callback(sitrans *Sitrans, ev event.Event) {
	go sitrans.trpt_cb(sitrans, ev) // Call the transport callback asynchronously
}
