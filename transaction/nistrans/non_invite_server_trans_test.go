package nistrans

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

	// 1. invite -> proceeding (send inv to core)
	trans.Start()
	assertCallback(t, coreCallbackChan, util.Event{Type: util.MESS, Data: update})
	assertState(t, trans.state, trying)

	// 2. 200 -> terminated (send 200 to transport)
	ok200 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 200,
			},
		},
	}
	trans.Event(util.Event{Type: util.MESS, Data: ok200})
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: ok200})
	assertState(t, trans.state, completed)

	// 3. timer j -> terminated
	sleep(tij_dur)
	sleep(1)
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
