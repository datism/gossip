package sitrans

import (
	"gossip/event"
	"gossip/message"
	"gossip/transaction"
	"reflect"
	"testing"
	"time"
)

func TestNormalScenario(t *testing.T) {
	// Channels to capture callback invocations
	transportCallbackChan := make(chan event.Event, 1)
	coreCallbackChan := make(chan event.Event, 1)

	// Create a mock transport callback
	mockTransportCallback := func(trans transaction.Transaction, ev event.Event) {
		transportCallbackChan <- ev
	}

	// Create a mock core callback
	mockCoreCallback := func(trans transaction.Transaction, ev event.Event) {
		coreCallbackChan <- ev
	}

	// Create a dummy SIP Message
	invite := &message.SIPMessage{
		Startline: message.Startline{
			Request: &message.Request{
				Method: "INVITE",
			},
		},
	}

	// Create a new Citrans instance
	trans := Make(invite, mockTransportCallback, mockCoreCallback)

	// 1. invite -> proceeding (send inv to core)
	trans.Start()
	assertCallback(t, coreCallbackChan, event.Event{Type: event.MESS, Data: invite})
	assertState(t, trans.state, proceeding)

	// 2. 100 -> proceeding (send 100 to transport)
	trying100 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 100,
			},
		},
	}
	trans.Event(event.Event{Type: event.MESS, Data: trying100})
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: trying100})
	assertState(t, trans.state, proceeding)

	// 3. 183 -> proceeding (send 183 to transport)
	proceeding183 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 183,
			},
		},
	}
	trans.Event(event.Event{Type: event.MESS, Data: proceeding183})
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: proceeding183})
	assertState(t, trans.state, proceeding)

	// 4. 180 -> proceeding (send 180 to transport)
	ringing180 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 180,
			},
		},
	}
	trans.Event(event.Event{Type: event.MESS, Data: ringing180})
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: ringing180})
	assertState(t, trans.state, proceeding)

	// 5. 200 -> terminated (send 200 to transport)
	ok200 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 200,
			},
		},
	}
	trans.Event(event.Event{Type: event.MESS, Data: ok200})
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: ok200})
	assertState(t, trans.state, terminated)
}

func TestErrorResponse(t *testing.T) {
	// Channels to capture callback invocations
	transportCallbackChan := make(chan event.Event, 1)
	coreCallbackChan := make(chan event.Event, 1)

	// Create a mock transport callback
	mockTransportCallback := func(trans transaction.Transaction, ev event.Event) {
		transportCallbackChan <- ev
	}

	// Create a mock core callback
	mockCoreCallback := func(trans transaction.Transaction, ev event.Event) {
		coreCallbackChan <- ev
	}

	// Create a dummy SIP Message
	inviteMessage := &message.SIPMessage{
		Startline: message.Startline{
			Request: &message.Request{
				Method: "INVITE",
			},
		},
	}

	// Create a new Citrans instance
	trans := Make(inviteMessage, mockTransportCallback, mockCoreCallback)

	// 1. invite -> proceeding (send inv to core)
	trans.Start()
	assertCallback(t, coreCallbackChan, event.Event{Type: event.MESS, Data: inviteMessage})
	assertState(t, trans.state, proceeding)

	// 2. 100 -> proceeding (send 100 to transport)
	trying100 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 100,
			},
		},
	}
	trans.Event(event.Event{Type: event.MESS, Data: trying100})
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: trying100})
	assertState(t, trans.state, proceeding)

	// 3. 3xx -> completed (send 3xx to transport)
	notfound404 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 400,
			},
		},
	}
	trans.Event(event.Event{Type: event.MESS, Data: notfound404})
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: notfound404})
	assertState(t, trans.state, completed)

	// 4. ACK -> confirmed
	ack := &message.SIPMessage{
		Startline: message.Startline{
			Request: &message.Request{
				Method: "ACK",
			},
		},
	}
	trans.Event(event.Event{Type: event.MESS, Data: ack})
	sleep(1)
	assertState(t, trans.state, confirmed)

	// 5. timer i -> terminated
	sleep(tii_dur)
	sleep(1)
	assertState(t, trans.state, terminated)
}

func TestTimeoutTimer(t *testing.T) {
	// Channels to capture callback invocations
	transportCallbackChan := make(chan event.Event, 1)
	coreCallbackChan := make(chan event.Event, 1)

	// Create a mock transport callback
	mockTransportCallback := func(trans transaction.Transaction, ev event.Event) {
		transportCallbackChan <- ev
	}

	// Create a mock core callback
	mockCoreCallback := func(trans transaction.Transaction, ev event.Event) {
		coreCallbackChan <- ev
	}

	// Create a dummy SIP Message
	invite := &message.SIPMessage{
		Startline: message.Startline{
			Request: &message.Request{
				Method: "INVITE",
			},
		},
	}

	// Create a new Citrans instance
	trans := Make(invite, mockTransportCallback, mockCoreCallback)

	// 1. invite -> calling (send invite to core)
	trans.Start()
	assertCallback(t, coreCallbackChan, event.Event{Type: event.MESS, Data: invite})
	assertState(t, trans.state, proceeding)

	// 2. timer prv -> proceeding (send 100 to transport)
	trying100 := message.MakeGenericResponse(100, "TRYING", invite)
	sleep(tiprv_dur)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: trying100})
	assertState(t, trans.state, proceeding)

	// 3. 3xx -> completed (send 3xx to transport)
	notfound404 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 400,
			},
		},
	}
	trans.Event(event.Event{Type: event.MESS, Data: notfound404})
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: notfound404})
	assertState(t, trans.state, completed)

	// 4. invite -> completed (send 300 to transport)
	trans.Event(event.Event{Type: event.MESS, Data: invite})
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: notfound404})
	assertState(t, trans.state, completed)

	// 5. timer g -> completed (send 300 to transport)
	sleep(tig_dur)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: notfound404})
	assertState(t, trans.state, completed)

	// 6. timer g -> completed (send 300 to transport)
	sleep(2 * tig_dur)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: notfound404})
	assertState(t, trans.state, completed)

	// 7. timer g -> completed (send 300 to transport)
	sleep(4 * tig_dur)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: notfound404})
	assertState(t, trans.state, completed)

	// 8. timer g -> completed (send 300 to transport)
	sleep(t2)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: notfound404})
	assertState(t, trans.state, completed)

	// 9. timer g -> completed (send 300 to transport)
	sleep(t2)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: notfound404})
	assertState(t, trans.state, completed)

	// 10. timer g -> completed (send 300 to transport)
	sleep(t2)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: notfound404})
	assertState(t, trans.state, completed)

	// 11. timer g -> completed (send 300 to transport)
	sleep(t2)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: notfound404})
	assertState(t, trans.state, completed)

	// 12. timer g -> completed (send 300 to transport)
	sleep(t2)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: notfound404})
	assertState(t, trans.state, completed)

	// 13. timer g -> completed (send 300 to transport)
	sleep(t2)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: notfound404})
	assertState(t, trans.state, completed)

	// 14. timer g -> completed (send 300 to transport)
	sleep(t2)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: notfound404})
	assertState(t, trans.state, completed)

	// 15. timer h -> termiated (send timeout to core)
	sleep(500)
	assertCallback(t, coreCallbackChan, event.Event{Type: event.TIMEOUT, Data: invite})
	assertState(t, trans.state, terminated)
}

func assertState(t *testing.T, actual, expected state) {
	if actual != expected {
		t.Errorf("Expected state %v, got %v", expected, actual)
	}
}

func assertCallback(t *testing.T, callbackChan <-chan event.Event, expected_event event.Event) {
	select {
	case ev := <-callbackChan:
		if !reflect.DeepEqual(ev, expected_event) {
			t.Errorf("Expected callback with event %v, got %v", expected_event, ev)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Expected callback but none received with event %v", expected_event)
	}
}

func sleep(duration int) {
	time.Sleep(time.Duration(duration) * time.Millisecond)
}
