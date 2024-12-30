package citrans

import (
	"gossip/event"
	"gossip/message"
	"gossip/message/cseq"
	"gossip/transaction"
	"reflect"
	"testing"
	"time"
)

func TestNormalScenario(t *testing.T) {
	// Channels to capture callback invocations
	transportCallbackChan := make(chan event.Event, 3)
	coreCallbackChan := make(chan event.Event, 3)

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

	// 1. invite -> calling (send invite to transport)
	trans.Start()
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: inviteMessage})
	assertState(t, trans.state, calling)

	// 2. 100 -> proceeding (send 100 to core)
	trying100 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 100,
			},
		},
	}
	trans.Event(event.Event{Type: event.MESS, Data: trying100})
	assertCallback(t, coreCallbackChan, event.Event{Type: event.MESS, Data: trying100})
	assertState(t, trans.state, proceeding)

	// 3. 183 -> proceeding (send 183 to core)
	proceeding183 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 183,
			},
		},
	}
	trans.Event(event.Event{Type: event.MESS, Data: proceeding183})
	assertCallback(t, coreCallbackChan, event.Event{Type: event.MESS, Data: proceeding183})
	assertState(t, trans.state, proceeding)

	// 4. 180 -> proceeding (send 180 to core)
	ringing180 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 180,
			},
		},
	}
	trans.Event(event.Event{Type: event.MESS, Data: ringing180})
	assertCallback(t, coreCallbackChan, event.Event{Type: event.MESS, Data: ringing180})
	assertState(t, trans.state, proceeding)

	// 5. 200 -> terminated (send 200 to core)
	ok200 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 200,
			},
		},
	}
	trans.Event(event.Event{Type: event.MESS, Data: ok200})
	assertCallback(t, coreCallbackChan, event.Event{Type: event.MESS, Data: ok200})
	assertState(t, trans.state, terminated)
}

func TestTimeoutTimer(t *testing.T) {
	// Channels to capture callback invocations
	transportCallbackChan := make(chan event.Event, 3)
	coreCallbackChan := make(chan event.Event, 3)

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

	// 1. invite -> calling (send invite to transport)
	trans.Start()
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: inviteMessage})
	assertState(t, trans.state, calling)

	// 2. timer a  (send invite to transport)
	sleep(tia_dur)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: inviteMessage})
	assertState(t, trans.state, calling)

	// 3. timer a  (send invite to transport)
	sleep(2 * tia_dur)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: inviteMessage})
	assertState(t, trans.state, calling)

	// 4. timer a  (send invite to transport)
	sleep(4 * tia_dur)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: inviteMessage})
	assertState(t, trans.state, calling)

	// 5. timer a (send invite to transport)
	sleep(8 * tia_dur)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: inviteMessage})
	assertState(t, trans.state, calling)

	// 6. timer a (send invite to transport)
	sleep(16 * tia_dur)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: inviteMessage})
	assertState(t, trans.state, calling)

	// 7. timer a - 6 (send invite to transport)
	sleep(32 * tia_dur)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: inviteMessage})
	assertState(t, trans.state, calling)

	// 8. timer b -> terminated (send timeout to core)
	sleep(500)
	assertCallback(t, coreCallbackChan, event.Event{Type: event.TIMEOUT, Data: TIMERB})
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
		CSeq: &cseq.SIPCseq{
			Method: "INVITE",
			Seq:    1,
		},
	}

	// Create a new Citrans instance
	trans := Make(inviteMessage, mockTransportCallback, mockCoreCallback)

	// 1. invite -> calling (send invite to transport)
	trans.Start()
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: inviteMessage})
	assertState(t, trans.state, calling)

	// 2. 100 -> proceeding (send 100 to core)
	trying100 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 100,
			},
		},
	}
	trans.Event(event.Event{Type: event.MESS, Data: trying100})
	assertCallback(t, coreCallbackChan, event.Event{Type: event.MESS, Data: trying100})
	assertState(t, trans.state, proceeding)

	// 3. 3xx -> completed (send ack to transport + send 3xx to core)
	notfound404 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 400,
			},
		},
	}
	trans.Event(event.Event{Type: event.MESS, Data: notfound404})
	assertCallback(t, coreCallbackChan, event.Event{Type: event.MESS, Data: notfound404})
	ack404 := message.MakeGenericAck(inviteMessage, notfound404)
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: ack404})
	assertState(t, trans.state, completed)

	// 4. 3xx -> completed (send ack to transport)
	trans.Event(event.Event{Type: event.MESS, Data: notfound404})
	assertCallback(t, transportCallbackChan, event.Event{Type: event.MESS, Data: ack404})
	assertState(t, trans.state, completed)

	// 5. timer d -> terminated
	sleep(tid_dur)
	sleep(100)
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
