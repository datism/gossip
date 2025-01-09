package ictrans

import (
	"gossip/message"
	"gossip/transaction"
	"gossip/util"
)

//                     |INVITE from TU
//              Timer A fires     |INVITE sent
//              Reset A,          V                      Timer B fires
//              INVITE sent +-----------+                or Transport Err.
//                +---------|           |---------------+inform TU
//                |         |  Calling  |               |
//                +-------->|           |-------------->|
//                          +-----------+ 2xx           |
//                             |  |       2xx to TU     |
//                             |  |1xx                  |
//     300-699 +---------------+  |1xx to TU            |
//    ACK sent |                  |                     |
// resp. to TU |  1xx             V                     |
//             |  1xx to TU  -----------+               |
//             |  +---------|           |               |
//             |  |         |Proceeding |-------------->|
//             |  +-------->|           | 2xx           |
//             |            +-----------+ 2xx to TU     |
//             |       300-699    |                     |
//             |       ACK sent,  |                     |
//             |       resp. to TU|                     |
//             |                  |                     |      NOTE:
//             |  300-699         V                     |
//             |  ACK sent  +-----------+Transport Err. |  transitions
//             |  +---------|           |Inform TU      |  labeled with
//             |  |         | Completed |-------------->|  the event
//             |  +-------->|           |               |  over the action
//             |            +-----------+               |  to take
//             |              ^   |                     |
//             |              |   | Timer D fires       |
//             +--------------+   | -                   |
//                                |                     |
//                                V                     |
//                          +-----------+               |
//                          |           |               |
//                          | Terminated|<--------------+
//                          |           |
//                          +-----------+

// Constants representing timer durations in milliseconds
const t1 = 500  // Default value for T1 (Round Trip Time estimate between client and server)
const tia_dur = t1         // Timer A duration, starts with T1 duration
const tib_dur = 64 * t1    // Timer B duration, starts with 64*T1
const tid_dur = 32000      // Timer D duration (in case of transport reliability issues), typically 32 seconds

// Constants representing timer types
const (
    timer_a = iota   // Timer A: Retransmit the request on timeout
    timer_b           // Timer B: Transaction timeout (in the calling state)
    timer_d           // Timer D: Completion timeout after receiving final response
)

// Define the states of the INVITE client transaction
type state int

const (
    calling = iota    // Initial state after INVITE is sent, waiting for provisional or final response
    proceeding        // State after receiving a provisional (1xx) response
    completed         // State after receiving a final (2xx-6xx) response
    terminated        // Final state, transaction is completed and destroyed
)

// Ictrans represents a SIP INVITE client transaction
type Ictrans struct {
    state   state          // Current state of the transaction
    message *message.SIPMessage // The SIP message being processed (INVITE or response)
    timers  [3]util.Timer  // The timers used to manage retransmissions and timeouts
    transc  chan util.Event  // Channel for receiving events and processing them
    trpt_cb func(transaction.Transaction, util.Event)  // Transport callback
    core_cb func(transaction.Transaction, util.Event)   // Core callback
}

// Make creates a new instance of a client transaction, initializing timers and setting initial state
func Make(
    message *message.SIPMessage,  // The INVITE message to be processed
    transport_callback func(transaction.Transaction, util.Event),  // Transport layer callback
    core_callback func(transaction.Transaction, util.Event),  // Core callback
) *Ictrans {
    timera := util.NewTimer()   // Timer A for retransmissions
    timerb := util.NewTimer()   // Timer B for transaction timeout
    timerd := util.NewTimer()   // Timer D for completion timeout

    return &Ictrans{
        message: message.DeepCopy(),        // The initial SIP message (INVITE)
        transc:  make(chan util.Event), // Channel to communicate events
        timers:  [3]util.Timer{timera, timerb, timerd}, // Initialize timers
        state:   calling,         // Start with the calling state
        trpt_cb: transport_callback, // Set transport callback
        core_cb: core_callback,   // Set core callback
    }
}

// Event triggers an event in the transaction. The event can be a SIP message or timeout.
func (trans Ictrans) Event(event util.Event) {
    trans.transc <- event
}

// Start begins the processing of the client transaction, invoking the transport callback and starting timers.
func (trans *Ictrans) Start() {
    go trans.start()  // Start processing in a separate goroutine to handle events asynchronously
}

// start is the main loop that processes events in the client transaction.
func (trans *Ictrans) start() {
    // Initial action: Call transport callback to send INVITE message
    call_transport_callback(trans, util.Event{Type: util.MESS, Data: trans.message.DeepCopy()})
    // Start Timer A (T1) for retransmissions and Timer B (64*T1) for transaction timeout
    trans.timers[timer_a].Start(tia_dur)
    trans.timers[timer_b].Start(tib_dur)

    var ev util.Event

    // Event loop that listens for events (SIP messages or timer expirations)
    for {
        select {
        case ev = <-trans.transc:  // Message event (SIP response)
        case <-trans.timers[timer_a].Chan():  // Timer A expired, triggering a retransmission
            ev = util.Event{Type: util.TIMEOUT, Data: timer_a}
        case <-trans.timers[timer_b].Chan():  // Timer B expired, transaction timed out
            ev = util.Event{Type: util.TIMEOUT, Data: timer_b}
        case <-trans.timers[timer_d].Chan():  // Timer D expired, termination after final response
            ev = util.Event{Type: util.TIMEOUT, Data: timer_d}
        }

        // Handle the received event (message or timeout)
        trans.handle_event(ev)

        // If the transaction is terminated, exit the loop
        if trans.state == terminated {
            close(trans.transc)  // Close the event channel when the transaction ends
            break
        }
    }
}

// handle_event processes events, which can be timeouts or received messages
func (trans *Ictrans) handle_event(ev util.Event) {
    switch ev.Type {
    case util.TIMEOUT:
        trans.handle_timer(ev)  // Handle timeout events (timer expirations)
    case util.MESS:
        trans.handle_msg(ev)  // Handle received SIP messages (responses)
    default:
        return
    }
}

// handle_timer processes timeout events, which can trigger retransmissions or state transitions
func (trans *Ictrans) handle_timer(ev util.Event) {
    if ev.Data == timer_b {  // Timer B expired, inform TU of timeout and terminate transaction
        call_core_callback(trans, util.Event{Type: util.TIMEOUT, Data: trans.message.DeepCopy()})
        trans.state = terminated
    } else if ev.Data == timer_a && trans.state == calling {  // Timer A expired in calling state, retransmit INVITE
        call_transport_callback(trans, util.Event{Type: util.MESS, Data: trans.message.DeepCopy()})
        trans.timers[timer_a].Start(trans.timers[timer_a].Duration() * 2)  // Double Timer A duration
    } else if ev.Data == timer_d && trans.state == completed {  // Timer D expired in completed state, terminate transaction
        trans.state = terminated
    }
}

// handle_msg processes received SIP messages, transitioning states based on response codes
func (trans *Ictrans) handle_msg(ev util.Event) {
    response, ok := ev.Data.(*message.SIPMessage)  // Extract the SIP message from the event
    if !ok || response.Response == nil {  // Invalid or missing response, ignore the event
        return
    }

    status_code := response.Response.StatusCode  // Get the response's status code

    if status_code >= 100 && status_code < 200 {  // Provisional response (1xx)
        if trans.state == calling {  // If in calling state, transition to proceeding
            trans.timers[timer_a].Stop()  // Stop Timer A as no more retransmissions are needed
            call_core_callback(trans, ev)  // Pass 1xx response to the core callback
            trans.state = proceeding  // Transition to proceeding state
        } else if trans.state == proceeding {  // In proceeding state, pass 1xx to the TU
            call_core_callback(trans, ev)
        }
    } else if status_code >= 200 && status_code <= 300 && trans.state < completed {  // Final success response (2xx)
        call_core_callback(trans, ev)  // Pass the final response to the core
        trans.state = terminated  // Transition to terminated state
    } else if status_code > 300 {  // Error response (3xx-6xx)
        if trans.state < completed {  // If in calling or proceeding state, generate ACK and stop Timer B
            call_core_callback(trans, ev)
            ack := message.MakeGenericAck(trans.message, response)  // Create an ACK for the response
            call_transport_callback(trans, util.Event{Type: util.MESS, Data: ack.DeepCopy()})  // Send the ACK

            trans.timers[timer_b].Stop()  // Stop Timer B (transaction timeout)
            trans.timers[timer_d].Start(tid_dur)  // Start Timer D (completion timeout)
            trans.state = completed  // Transition to completed state
        } else if trans.state == completed {  // In completed state, just retransmit the ACK
            ack := message.MakeGenericAck(trans.message, response)
            call_transport_callback(trans, util.Event{Type: util.MESS, Data: ack.DeepCopy()})
        }
    }
}

// call_core_callback invokes the core callback to handle transaction-related events
func call_core_callback(citrans *Ictrans, ev util.Event) {
    citrans.core_cb(citrans, ev)  // Call the core callback in a new goroutine
}

// call_transport_callback invokes the transport callback to send or receive messages
func call_transport_callback(citrans *Ictrans, ev util.Event) {
    citrans.trpt_cb(citrans, ev)  // Call the transport callback in a new goroutine
}
