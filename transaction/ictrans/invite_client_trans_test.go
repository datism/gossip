package ictrans

import (
	"gossip/util"
	"gossip/message"
	"gossip/message/cseq"
	"gossip/transaction"
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
	invite := &message.SIPMessage{
		Startline: message.Startline{
			Request: &message.Request{
				Method: "INVITE",
			},
		},
	}

	// Create a new Citrans instance
	trans := Make(invite, mockTransportCallback, mockCoreCallback)

	// 1. invite -> calling (send invite to transport)
	trans.Start()
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: invite})
	assertState(t, trans.state, calling)

	// 2. 100 -> proceeding (send 100 to core)
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

	// 3. 183 -> proceeding (send 183 to core)
	proceeding183 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 183,
			},
		},
	}
	trans.Event(util.Event{Type: util.MESS, Data: proceeding183})
	assertCallback(t, coreCallbackChan, util.Event{Type: util.MESS, Data: proceeding183})
	assertState(t, trans.state, proceeding)

	// 4. 180 -> proceeding (send 180 to core)
	ringing180 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 180,
			},
		},
	}
	trans.Event(util.Event{Type: util.MESS, Data: ringing180})
	assertCallback(t, coreCallbackChan, util.Event{Type: util.MESS, Data: ringing180})
	assertState(t, trans.state, proceeding)

	// 5. 200 -> terminated (send 200 to core)
	ok200 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 200,
			},
		},
	}
	trans.Event(util.Event{Type: util.MESS, Data: ok200})
	assertCallback(t, coreCallbackChan, util.Event{Type: util.MESS, Data: ok200})
	assertState(t, trans.state, terminated)
}

func TestErrorResponse(t *testing.T) {
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
	invite := &message.SIPMessage{
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
	trans := Make(invite, mockTransportCallback, mockCoreCallback)

	// 1. invite -> calling (send invite to transport)
	trans.Start()
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: invite})
	assertState(t, trans.state, calling)

	// 2. 100 -> proceeding (send 100 to core)
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

	// 3. 3xx -> completed (send ack to transport + send 3xx to core)
	notfound404 := &message.SIPMessage{
		Startline: message.Startline{
			Response: &message.Response{
				StatusCode: 400,
			},
		},
	}
	trans.Event(util.Event{Type: util.MESS, Data: notfound404})
	assertCallback(t, coreCallbackChan, util.Event{Type: util.MESS, Data: notfound404})
	ack404 := message.MakeGenericAck(invite, notfound404)
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: ack404})
	assertState(t, trans.state, completed)

	// 4. 3xx -> completed (send ack to transport)
	trans.Event(util.Event{Type: util.MESS, Data: notfound404})
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: ack404})
	assertState(t, trans.state, completed)

	// 5. timer d -> terminated
	sleep(tid_dur)
	sleep(100)
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
	invite := &message.SIPMessage{
		Startline: message.Startline{
			Request: &message.Request{
				Method: "INVITE",
			},
		},
	}

	// Create a new Citrans instance
	trans := Make(invite, mockTransportCallback, mockCoreCallback)

	// 1. invite -> calling (send invite to transport)
	trans.Start()
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: invite})
	assertState(t, trans.state, calling)

	// 2. timer a  (send invite to transport)
	sleep(tia_dur)
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: invite})
	assertState(t, trans.state, calling)

	// 3. timer a  (send invite to transport)
	sleep(2 * tia_dur)
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: invite})
	assertState(t, trans.state, calling)

	// 4. timer a  (send invite to transport)
	sleep(4 * tia_dur)
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: invite})
	assertState(t, trans.state, calling)

	// 5. timer a (send invite to transport)
	sleep(8 * tia_dur)
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: invite})
	assertState(t, trans.state, calling)

	// 6. timer a (send invite to transport)
	sleep(16 * tia_dur)
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: invite})
	assertState(t, trans.state, calling)

	// 7. timer a - 6 (send invite to transport)
	sleep(32 * tia_dur)
	assertCallback(t, transportCallbackChan, util.Event{Type: util.MESS, Data: invite})
	assertState(t, trans.state, calling)

	// 8. timer b -> terminated (send timeout to core)
	sleep(500)
	assertCallback(t, coreCallbackChan, util.Event{Type: util.TIMEOUT, Data: invite})
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
