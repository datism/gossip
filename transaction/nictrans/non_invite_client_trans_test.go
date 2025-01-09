package nictrans

import (
	"gossip/message"
	"gossip/transaction"
	"gossip/util"
	"reflect"
	"testing"
	"time"
)

func TestNormalScenario(t *testing.T) {
	// Channels to capture callback invocations
	transportCallbackChan := make(chan util.Event, 1)
	coreCallbackChan := make(chan util.Event, 1)

	// Create a mock transport callback
	mockTransportCallback := func(trans transaction.Transaction, ev util.Event) {
		transportCallbackChan <- ev
	}

	// Create a mock core callback
	mockCoreCallback := func(trans transaction.Transaction, ev util.Event) {
		coreCallbackChan <- ev
	}

	// Create a dummy SIP Message
	update := &message.SIPMessage{
		Startline: message.Startline{
			Request: &message.Request{
				Method: "UPDATE",
			},
		},
	}

	// Create a new Citrans instance
	trans := Make(update, mockTransportCallback, mockCoreCallback)

	// 1. update -> calling (send invite to transport)
	trans.Start()
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: update})
	assertState(t, trans.state, trying)

	// 2. 200 -> terminated (send 200 to core)
	ok200 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 200,
			},
		},
	}
	trans.Event(util.Event{Type: util.MESS, Data: ok200})
	assertCallback(t, coreCallbackChan, util.Event{Type: util.MESS, Data: ok200})
	assertState(t, trans.state, completed)

	sleep(tik_dur)
	sleep(1)
	assertState(t, trans.state, terminated)
}

func TestTimeoutTimer(t *testing.T) {
	// Channels to capture callback invocations
	transportCallbackChan := make(chan util.Event, 1)
	coreCallbackChan := make(chan util.Event, 1)

	// Create a mock transport callback
	mockTransportCallback := func(trans transaction.Transaction, ev util.Event) {
		transportCallbackChan <- ev
	}

	// Create a mock core callback
	mockCoreCallback := func(trans transaction.Transaction, ev util.Event) {
		coreCallbackChan <- ev
	}

	// Create a dummy SIP Message
	update := &message.SIPMessage{
		Startline: message.Startline{
			Request: &message.Request{
				Method: "UPDATE",
			},
		},
	}

	// Create a new Citrans instance
	trans := Make(update, mockTransportCallback, mockCoreCallback)

	// 1. invite -> calling (send invite to transport)
	trans.Start()
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: update})
	assertState(t, trans.state, trying)

	// 2. timer e  (send update to transport)
	sleep(tie_dur)
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: update})
	assertState(t, trans.state, trying)

	// 3. timer e  (send update to transport)
	sleep(2 * tie_dur)
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: update})
	assertState(t, trans.state, trying)

	// 4. timer e  (send update to transport)
	sleep(4 * tie_dur)
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: update})
	assertState(t, trans.state, trying)

	// 5. timer e  (send update to transport)
	sleep(8 * tie_dur)
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: update})
	assertState(t, trans.state, trying)

	// 6. timer e  (send update to transport)
	sleep(t2)
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: update})
	assertState(t, trans.state, trying)

	// 7. 100 -> proceeding (send 100 to core)
	trying100 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 100,
			},
		},
	}
	trans.Event(util.Event{Type: util.MESS, Data: trying100})
	assertCallback(t, coreCallbackChan, util.Event{Type: util.MESS, Data: trying100})
	assertState(t, trans.state, proceeding)

	// 8. timer e  (send update to transport)
	sleep(t2)
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: update})
	assertState(t, trans.state, proceeding)

	// 9. timer e  (send update to transport)
	sleep(t2)
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: update})
	assertState(t, trans.state, proceeding)

	// 10. timer e  (send update to transport)
	sleep(t2)
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: update})
	assertState(t, trans.state, proceeding)

	// 11. timer e  (send update to transport)
	sleep(t2)
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: update})
	assertState(t, trans.state, proceeding)

	// 12. timer e  (send update to transport)
	sleep(t2)
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: update})
	assertState(t, trans.state, proceeding)

	// 13. timer f -> terminated (send timeout to core)
	sleep(500)
	assertCallback(t, coreCallbackChan, util.Event{Type: util.TIMEOUT, Data: update})
	assertState(t, trans.state, terminated)
}

func assertState(t *testing.T, actual, expected state) {
	if actual != expected {
		t.Errorf("Expected state %v, got %v", expected, actual)
	}
}

func assertCallback(t *testing.T, callbackChan <-chan util.Event, expected_event util.Event) {
	select {
	case ev := <-callbackChan:
		if !reflect.DeepEqual(ev, expected_event) {
			t.Errorf("Expected callback with util %v, got %v", expected_event, ev)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Expected callback but none received with util %v", expected_event)
	}
}

func sleep(duration int) {
	time.Sleep(time.Duration(duration) * time.Millisecond)
}
