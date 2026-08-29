package notifyhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	wmwebhook "monorepo/bin-webhook-manager/models/webhook"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_PublishWebhookEvent(t *testing.T) {

	tests := []struct {
		name       string
		customerID uuid.UUID
		eventType  string
		event      *testEvent

		expectEvent   *sock.Event
		expectWebhook []byte
	}{
		{
			"normal",
			uuid.FromStringOrNil("419841c6-825d-11ec-823f-13ee3d677a1b"),
			"test_created",
			&testEvent{
				Name:   "test name",
				Detail: "test detail",
			},
			&sock.Event{
				Type:      "test_created",
				Publisher: testPublisher,
				DataType:  dataTypeJSON,
			},
			[]byte(`{"name":"test name","detail":"test detail"}`),
		},
		{
			"customer id is empty",
			uuid.Nil,
			"test_created",
			&testEvent{
				Name:   "test name",
				Detail: "test detail",
			},
			&sock.Event{
				Type:      "test_created",
				Publisher: testPublisher,
				DataType:  dataTypeJSON,
			},
			[]byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &notifyHandler{
				sockHandler: mockSock,
				reqHandler:  mockReq,
				queueNotify: commonoutline.QueueNameCallEvent,
				publisher:   testPublisher,
			}

			ctx := context.Background()

			tt.expectEvent.Data, _ = json.Marshal(tt.event)
			mockSock.EXPECT().EventPublish(string(h.queueNotify), "", tt.expectEvent)
			if tt.customerID != uuid.Nil {
				mockReq.EXPECT().WebhookV1WebhookSend(gomock.Any(), tt.customerID, wmwebhook.DataTypeJSON, string(tt.eventType), tt.expectWebhook)
			}

			h.PublishWebhookEvent(ctx, tt.customerID, tt.eventType, tt.event)

			time.Sleep(time.Millisecond * 1000)
		})
	}
}

func Test_PublishWebhook(t *testing.T) {

	tests := []struct {
		name       string
		customerID uuid.UUID
		eventType  string
		event      *testEvent

		expectEvent   *sock.Event
		expectWebhook []byte
	}{

		{
			"normal",
			uuid.FromStringOrNil("8225c952-825d-11ec-a03a-afa5f50337e1"),
			"test_created",
			&testEvent{
				Name:   "test name",
				Detail: "test detail",
			},
			&sock.Event{
				Type:      "test_created",
				Publisher: testPublisher,
				DataType:  dataTypeJSON,
			},
			[]byte(`{"name":"test name","detail":"test detail"}`),
		},
		{
			"customer id is empty",
			uuid.Nil,
			"test_created",
			&testEvent{
				Name:   "test name",
				Detail: "test detail",
			},
			&sock.Event{
				Type:      "test_created",
				Publisher: testPublisher,
				DataType:  dataTypeJSON,
			},
			[]byte(``),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &notifyHandler{
				sockHandler: mockSock,
				reqHandler:  mockReq,
				queueNotify: commonoutline.QueueNameCallEvent,
				publisher:   testPublisher,
			}

			ctx := context.Background()

			tt.expectEvent.Data, _ = json.Marshal(tt.event)
			if tt.customerID != uuid.Nil {
				mockReq.EXPECT().WebhookV1WebhookSend(gomock.Any(), tt.customerID, wmwebhook.DataTypeJSON, string(tt.eventType), tt.expectWebhook)
			}
			h.PublishWebhook(ctx, tt.customerID, tt.eventType, tt.event)

			time.Sleep(time.Millisecond * 1000)
		})
	}
}

func Test_PublishEvent(t *testing.T) {

	tests := []struct {
		name      string
		eventType string
		event     *testEvent

		expectEvent *sock.Event
	}{

		{
			"normal",
			"test_created",
			&testEvent{
				Name:   "test name",
				Detail: "test detail",
			},
			&sock.Event{
				Type:      "test_created",
				Publisher: testPublisher,
				DataType:  dataTypeJSON,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &notifyHandler{
				sockHandler: mockSock,
				reqHandler:  mockReq,
				queueNotify: commonoutline.QueueNameCallEvent,
				publisher:   testPublisher,
			}

			tt.expectEvent.Data, _ = json.Marshal(tt.event)
			mockSock.EXPECT().EventPublish(string(h.queueNotify), "", tt.expectEvent)

			h.PublishEvent(context.Background(), tt.eventType, tt.event)

			time.Sleep(time.Millisecond * 1000)
		})
	}
}

func Test_PublishEventRaw(t *testing.T) {

	tests := []struct {
		name string

		eventType string
		dataType  string
		data      []byte

		expectEvent *sock.Event
	}{
		{
			"normal",

			"test_created",
			"application/json",
			[]byte(`{"type":"ChannelCreated"}`),

			&sock.Event{
				Type:      "test_created",
				Publisher: testPublisher,
				DataType:  "application/json",
				Data:      []byte(`{"type":"ChannelCreated"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &notifyHandler{
				sockHandler: mockSock,
				reqHandler:  mockReq,
				queueNotify: commonoutline.QueueNameCallEvent,
				publisher:   testPublisher,
			}

			ctx := context.Background()

			mockSock.EXPECT().EventPublish(string(h.queueNotify), "", tt.expectEvent)

			h.PublishEventRaw(ctx, tt.eventType, tt.dataType, tt.data)

			time.Sleep(time.Millisecond * 1000)
		})
	}
}

func TestPublishEventWithRoutingKey(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	h := &notifyHandler{
		sockHandler: mockSock,
		queueNotify: "test.queue",
		publisher:   "test-service",
	}

	data := map[string]string{"foo": "bar"}
	routingKey := "customer_id.abc123.call.call_updated.xyz789"

	mockSock.EXPECT().EventPublish("test.queue", routingKey, gomock.Any()).Return(nil)

	h.PublishEventWithRoutingKey(context.Background(), "call_updated", routingKey, data)

	// PublishEventWithRoutingKey is fire-and-forget like PublishEvent; assert via mock call above.
}

// TestPublishEventWithRoutingKey_MarshalError verifies the marshal-error path returns early
// without ever calling EventPublish (no mock expectation set -- gomock's strict-by-default
// behavior fails the test if EventPublish is called unexpectedly).
func TestPublishEventWithRoutingKey_MarshalError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	h := &notifyHandler{
		sockHandler: mockSock,
		queueNotify: "test.queue",
		publisher:   "test-service",
	}

	// json.Marshal cannot serialize a bare func value -- forces the marshal-error branch.
	unmarshalable := func() {}

	h.PublishEventWithRoutingKey(context.Background(), "call_updated", "customer_id.abc123.#", unmarshalable)

	// No mockSock.EXPECT() set for EventPublish -- if PublishEventWithRoutingKey called it
	// despite the marshal failure, gomock would fail this test with an unexpected-call error.
}

// Test_PublishEvent_globalTopicPublish verifies the topic-only publish (VOIP-1407): with
// topicEnabled=true, the event is published exclusively to the global topic exchange with the
// generated routing key -- no fanout publish. promNotifyTotal/promNotifyProcessTime are the same
// shared counters the removed fanout-only path used to observe; with only one active publish path
// per instance, the topic-only publish growing them cannot double-count anything.
func Test_PublishEvent_globalTopicPublish(t *testing.T) {

	tests := []struct {
		name string

		publisher commonoutline.ServiceName
		eventType string
		event     eventtopic.SubscriptionIdentifier

		expectEvent      *sock.Event
		expectRoutingKey string
	}{
		{
			name: "subscription id resolved from the own-id method",

			publisher: "transcribe-manager",
			eventType: "transcribe_created",
			event: &testIDEvent{
				ID:   "9f01c3d2-a1bc-11f1-92ef-60452e5e40a2",
				Name: "test name",
			},

			expectEvent: &sock.Event{
				Type:      "transcribe_created",
				Publisher: "transcribe-manager",
				DataType:  dataTypeJSON,
			},
			expectRoutingKey: "transcribe-manager.transcribe.9f01c3d2-a1bc-11f1-92ef-60452e5e40a2.created",
		},
		{
			name: "subscription id resolved by the parent-stream address",

			publisher: "transcribe-manager",
			eventType: "transcribe_speech_interim",
			event: &testStreamEvent{
				ID:           "a0121e34-a1bc-11f1-92ef-60452e5e40a2",
				TranscribeID: "9f01c3d2-a1bc-11f1-92ef-60452e5e40a2",
			},

			expectEvent: &sock.Event{
				Type:      "transcribe_speech_interim",
				Publisher: "transcribe-manager",
				DataType:  dataTypeJSON,
			},
			expectRoutingKey: "transcribe-manager.transcribe.9f01c3d2-a1bc-11f1-92ef-60452e5e40a2.speech_interim",
		},
		{
			name: "explicit empty address degrades to the placeholder",

			publisher: "transcribe-manager",
			eventType: "transcribe_created",
			event: &testEvent{
				Name:   "test name",
				Detail: "test detail",
			},

			expectEvent: &sock.Event{
				Type:      "transcribe_created",
				Publisher: "transcribe-manager",
				DataType:  dataTypeJSON,
			},
			expectRoutingKey: "transcribe-manager.transcribe.-.created",
		},
		{
			name: "subscription id is nil uuid",

			publisher: "transcribe-manager",
			eventType: "transcribe_created",
			event: &testIDEvent{
				ID:   uuid.Nil.String(),
				Name: "test name",
			},

			expectEvent: &sock.Event{
				Type:      "transcribe_created",
				Publisher: "transcribe-manager",
				DataType:  dataTypeJSON,
			},
			expectRoutingKey: "transcribe-manager.transcribe.-.created",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &notifyHandler{
				sockHandler:  mockSock,
				reqHandler:   mockReq,
				queueNotify:  commonoutline.QueueNameTranscribeEvent,
				publisher:    tt.publisher,
				topicEnabled: true,
			}

			tt.expectEvent.Data, _ = json.Marshal(tt.event)

			// no fanout publish at all -- the topic exchange is the sole target.
			mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), tt.expectRoutingKey, tt.expectEvent).Return(nil)

			beforeOK := topicPublishCount(tt.eventType, topicPublishResultOK)
			beforeNotify := notifyTotalCount(tt.eventType)
			beforeProcessTime := notifyProcessTimeCount(tt.eventType)

			h.PublishEvent(context.Background(), tt.eventType, tt.event)

			if afterOK := topicPublishCount(tt.eventType, topicPublishResultOK); afterOK != beforeOK+1 {
				t.Errorf("Wrong match. expect: %f, got: %f", beforeOK+1, afterOK)
			}

			// VOIP-1407: promNotifyTotal/promNotifyProcessTime are shared across both the
			// fanout-only and topic-only paths now -- exactly one publish happened, so both grow
			// by exactly 1 via the single active path.
			if afterNotify := notifyTotalCount(tt.eventType); afterNotify != beforeNotify+1 {
				t.Errorf("Wrong match. expect: %f, got: %f", beforeNotify+1, afterNotify)
			}
			if afterProcessTime := notifyProcessTimeCount(tt.eventType); afterProcessTime != beforeProcessTime+1 {
				t.Errorf("Wrong match. expect: %d, got: %d", beforeProcessTime+1, afterProcessTime)
			}
		})
	}
}

// Test_PublishEvent_globalTopicPublishPlaceholderMetric verifies the placeholder counter only
// grows when no valid subscription address exists.
func Test_PublishEvent_globalTopicPublishPlaceholderMetric(t *testing.T) {

	tests := []struct {
		name string

		eventType string
		event     eventtopic.SubscriptionIdentifier

		expectPlaceholderDelta float64
	}{
		{
			name: "valid subscription id",

			eventType: "test_placeholdervalid",
			event: &testIDEvent{
				ID: "b1232145-a1bc-11f1-92ef-60452e5e40a2",
			},

			expectPlaceholderDelta: 0,
		},
		{
			name: "no subscription id",

			eventType: "test_placeholdermissing",
			event: &testEvent{
				Name: "test name",
			},

			expectPlaceholderDelta: 1,
		},
		{
			name: "nil uuid subscription id",

			eventType: "test_placeholdernil",
			event: &testIDEvent{
				ID: uuid.Nil.String(),
			},

			expectPlaceholderDelta: 1,
		},
		{
			// pins that the publish path meters an oversized address as a placeholder via
			// eventtopic.IsPlaceholderSubscriptionID -- an inline empty/nil check would let an
			// oversized id produce a "-" key while the placeholder counter stays flat.
			name: "oversized subscription id",

			eventType: "test_placeholderoversized",
			event: &testIDEvent{
				ID: strings.Repeat("a", 65),
			},

			expectPlaceholderDelta: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			h := &notifyHandler{
				sockHandler:  mockSock,
				queueNotify:  commonoutline.QueueNameTranscribeEvent,
				publisher:    "transcribe-manager",
				topicEnabled: true,
			}

			mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), gomock.Any(), gomock.Any()).Return(nil)

			before := topicPlaceholderCount(tt.eventType)

			h.PublishEvent(context.Background(), tt.eventType, tt.event)

			if after := topicPlaceholderCount(tt.eventType); after != before+tt.expectPlaceholderDelta {
				t.Errorf("Wrong match. expect: %f, got: %f", before+tt.expectPlaceholderDelta, after)
			}
		})
	}
}

// Test_publishEvent_globalTopicPublishFailurePropagates verifies the VOIP-1407 inversion: with no
// fanout fallback left to degrade to, a topic publish failure is now returned to the caller
// (previously logged and swallowed) in addition to being counted as an error. The returned error
// carries the routing key, so the diagnostic that used to be log-only survives to the caller.
func Test_publishEvent_globalTopicPublishFailurePropagates(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &notifyHandler{
		sockHandler:  mockSock,
		queueNotify:  commonoutline.QueueNameTranscribeEvent,
		publisher:    "transcribe-manager",
		topicEnabled: true,
	}

	eventType := "test_topicfailed"
	event := &testIDEvent{ID: "c2343256-a1bc-11f1-92ef-60452e5e40a2"}
	expectRoutingKey := "transcribe-manager.test.c2343256-a1bc-11f1-92ef-60452e5e40a2.topicfailed"

	mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), expectRoutingKey, gomock.Any()).Return(fmt.Errorf("no route"))

	beforeError := topicPublishCount(eventType, topicPublishResultError)
	beforeOK := topicPublishCount(eventType, topicPublishResultOK)

	m, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Could not marshal the event. err: %v", err)
	}

	// the private path is used here so the caller-visible error can be asserted directly. The
	// subscription id is passed pre-resolved, the way PublishEvent hands it over after calling the
	// mandatory method (VOIP-1419).
	errPublish := h.publishEvent(eventType, dataTypeJSON, m, requestTimeoutDefault, 0, event.EventSubscriptionID())
	if errPublish == nil {
		t.Fatal("Wrong match. expected an error from the topic publish to reach the caller.")
	}
	if !strings.Contains(errPublish.Error(), expectRoutingKey) {
		t.Errorf("Wrong match. expected the routing key in the returned error. err: %v", errPublish)
	}

	if afterError := topicPublishCount(eventType, topicPublishResultError); afterError != beforeError+1 {
		t.Errorf("Wrong match. expect: %f, got: %f", beforeError+1, afterError)
	}
	if afterOK := topicPublishCount(eventType, topicPublishResultOK); afterOK != beforeOK {
		t.Errorf("Wrong match. expect: %f, got: %f", beforeOK, afterOK)
	}
}

// Test_publishEvent_optionOffSkipsTopic verifies the default behavior stays byte-identical: no
// topic publish at all when the option is off.
func Test_publishEvent_optionOffSkipsTopic(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &notifyHandler{
		sockHandler: mockSock,
		queueNotify: commonoutline.QueueNameTranscribeEvent,
		publisher:   "transcribe-manager",
	}

	mockSock.EXPECT().EventPublish(string(h.queueNotify), "", gomock.Any()).Return(nil)
	mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), gomock.Any(), gomock.Any()).Times(0)

	h.PublishEvent(context.Background(), "test_optionoff", &testIDEvent{ID: "e4565478-a1bc-11f1-92ef-60452e5e40a2"})
}

// Test_publishEvent_delayedSkipsTopic verifies the delay > 0 branch never reaches the topic
// publish. No public API produces delay > 0 today, so the private function is targeted directly --
// the guard is defensive, delayed-event topic semantics are deferred to the follow-up.
func Test_publishEvent_delayedSkipsTopic(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &notifyHandler{
		sockHandler:  mockSock,
		queueNotify:  commonoutline.QueueNameTranscribeEvent,
		publisher:    "transcribe-manager",
		topicEnabled: true,
	}

	mockSock.EXPECT().EventPublishWithDelay(string(commonoutline.QueueNameDelay), string(h.queueNotify), gomock.Any(), DelaySecond).Return(nil)
	mockSock.EXPECT().EventPublish(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	if err := h.publishEvent("test_delayed", dataTypeJSON, []byte(`{"id":"f5676589-a1bc-11f1-92ef-60452e5e40a2"}`), requestTimeoutDefault, DelaySecond, "f5676589-a1bc-11f1-92ef-60452e5e40a2"); err != nil {
		t.Errorf("Wrong match. expected no error. err: %v", err)
	}
}

// Test_PublishEventRaw_globalTopicPublish verifies the raw path's placeholder contract
// (VOIP-1419): a []byte payload cannot carry an EventSubscriptionID and there is no JSON
// fallback, so on a topic-enabled handler EVERY Raw publish lands under the `-` placeholder --
// including a payload whose JSON carries a perfectly valid top-level "id". The first row is the
// mutation lock for the fallback's deletion: resurrecting it would put the payload id back into
// the key and fail this row.
func Test_PublishEventRaw_globalTopicPublish(t *testing.T) {

	tests := []struct {
		name string

		eventType string
		dataType  string
		data      []byte

		expectRoutingKey string
	}{
		{
			name: "json payload with a top level id still lands on the placeholder",

			eventType: "call_created",
			dataType:  "application/json",
			data:      []byte(`{"id":"06787690-a1bc-11f1-92ef-60452e5e40a2","name":"test"}`),

			expectRoutingKey: "call-manager.call.-.created",
		},
		{
			name: "json payload without an id",

			eventType: "call_created",
			dataType:  "application/json",
			data:      []byte(`{"name":"test"}`),

			expectRoutingKey: "call-manager.call.-.created",
		},
		{
			name: "non json payload",

			eventType: "ari_event",
			dataType:  "application/json",
			data:      []byte(`not a json payload`),

			expectRoutingKey: "call-manager.ari.-.event",
		},
		{
			name: "empty payload",

			eventType: "call_created",
			dataType:  "application/json",
			data:      []byte{},

			expectRoutingKey: "call-manager.call.-.created",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			h := &notifyHandler{
				sockHandler:  mockSock,
				queueNotify:  commonoutline.QueueNameCallEvent,
				publisher:    "call-manager",
				topicEnabled: true,
			}

			mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), tt.expectRoutingKey, gomock.Any()).Return(nil)

			h.PublishEventRaw(context.Background(), tt.eventType, tt.dataType, tt.data)
		})
	}
}

// Test_PublishEvent_typedNilSubscriptionIdentifier pins the typed-nil contract (VOIP-1404
// code-review round 1, F1). A nil pointer whose type implements SubscriptionIdentifier satisfies
// the type assertion, and calling the method on it dereferences a nil receiver. Neither the
// option-off path (which must not assert at all) nor the option-on path (which guards the nil
// before calling) may panic; with the option on the payload marshals to `null`, so the key falls
// on the `-` placeholder.
func Test_PublishEvent_typedNilSubscriptionIdentifier(t *testing.T) {

	tests := []struct {
		name string

		topicEnabled bool
		eventType    string

		expectTopicPublish     bool
		expectRoutingKey       string
		expectPlaceholderDelta float64
	}{
		{
			name: "option off",

			topicEnabled: false,
			eventType:    "transcribe_typedniloff",

			expectTopicPublish:     false,
			expectPlaceholderDelta: 0,
		},
		{
			name: "option on",

			topicEnabled: true,
			eventType:    "transcribe_typednilon",

			expectTopicPublish:     true,
			expectRoutingKey:       "transcribe-manager.transcribe.-.typednilon",
			expectPlaceholderDelta: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			h := &notifyHandler{
				sockHandler:  mockSock,
				queueNotify:  commonoutline.QueueNameTranscribeEvent,
				publisher:    "transcribe-manager",
				topicEnabled: tt.topicEnabled,
			}

			// a typed nil: the interface value is non-nil, but the pointer inside it is.
			var event *testStreamEvent

			// VOIP-1407: post-cutover the two arms publish to different exchanges -- the
			// option-off arm still fanout-publishes (topicEnabled=false is unchanged), the
			// option-on arm publishes topic-only (no fanout at all).
			if tt.expectTopicPublish {
				mockSock.EXPECT().EventPublish(string(h.queueNotify), "", gomock.Any()).Times(0)
				mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), tt.expectRoutingKey, gomock.Any()).Return(nil)
			} else {
				mockSock.EXPECT().EventPublish(string(h.queueNotify), "", gomock.Any()).Return(nil)
				mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), gomock.Any(), gomock.Any()).Times(0)
			}

			before := topicPlaceholderCount(tt.eventType)

			// a panic here fails the test -- the fanout publish would never happen either.
			h.PublishEvent(context.Background(), tt.eventType, event)

			if after := topicPlaceholderCount(tt.eventType); after != before+tt.expectPlaceholderDelta {
				t.Errorf("Wrong match. expect: %f, got: %f", before+tt.expectPlaceholderDelta, after)
			}
		})
	}
}

// Test_PublishEvent_nilInterface pins guard branch 1 of resolveSubscriptionID (VOIP-1419 design
// D4): an untyped nil argument still compiles against the interface parameter, and for a nil
// interface reflect reports Kind Invalid -- the typed-nil branch alone would miss it and the
// method call would panic. Pre-VOIP-1419 this input was safe only because the type assertion
// failed first; this test is the mutation lock for the explicit `data == nil` check. The payload
// marshals to `null`, so the key lands on the `-` placeholder.
func Test_PublishEvent_nilInterface(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &notifyHandler{
		sockHandler:  mockSock,
		queueNotify:  commonoutline.QueueNameTranscribeEvent,
		publisher:    "transcribe-manager",
		topicEnabled: true,
	}

	eventType := "transcribe_nilinterface"
	expectRoutingKey := "transcribe-manager.transcribe.-.nilinterface"

	mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), expectRoutingKey, gomock.Any()).Return(nil)

	before := topicPlaceholderCount(eventType)

	// a panic here fails the test.
	h.PublishEvent(context.Background(), eventType, nil)

	if after := topicPlaceholderCount(eventType); after != before+1 {
		t.Errorf("Wrong match. expect: %f, got: %f", before+1, after)
	}
}

// Test_PublishEvent_emptyAddressIgnoresPayloadID pins the method's authority (VOIP-1419): an
// explicit empty or uuid.Nil address degrades to the `-` placeholder even though the marshaled
// payload deliberately carries a valid top-level "id". If any payload-derived resolution ever
// leaked back in, the key would carry that id and the placeholder counter would stay flat.
func Test_PublishEvent_emptyAddressIgnoresPayloadID(t *testing.T) {

	tests := []struct {
		name string

		eventType string
		event     eventtopic.SubscriptionIdentifier

		expectRoutingKey string
	}{
		{
			name: "override returns an empty value",

			eventType: "transcribe_overrideempty",
			event: &testEmptyOverrideEvent{
				ID: "17898701-a1bc-11f1-92ef-60452e5e40a2",
			},

			expectRoutingKey: "transcribe-manager.transcribe.-.overrideempty",
		},
		{
			name: "override returns the nil uuid",

			eventType: "transcribe_overridenil",
			event: &testNilOverrideEvent{
				ID: "289a9812-a1bc-11f1-92ef-60452e5e40a2",
			},

			expectRoutingKey: "transcribe-manager.transcribe.-.overridenil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			h := &notifyHandler{
				sockHandler:  mockSock,
				queueNotify:  commonoutline.QueueNameTranscribeEvent,
				publisher:    "transcribe-manager",
				topicEnabled: true,
			}

			mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), tt.expectRoutingKey, gomock.Any()).Return(nil)

			before := topicPlaceholderCount(tt.eventType)

			h.PublishEvent(context.Background(), tt.eventType, tt.event)

			if after := topicPlaceholderCount(tt.eventType); after != before+1 {
				t.Errorf("Wrong match. expect: %f, got: %f", before+1, after)
			}
		})
	}
}

// Test_PublishEvent_optionOffSkipsSubscriptionIdentifier pins the topicEnabled gate itself
// (VOIP-1404 code-review round 2, F11). With the option off, PublishEvent must not touch the
// caller's data at all -- no type assertion, no EventSubscriptionID call. Asserting only on the
// resulting routing key cannot detect a missing gate, because the resolution succeeds harmlessly
// either way; the spy observes the call directly, so removing the gate fails this test.
func Test_PublishEvent_optionOffSkipsSubscriptionIdentifier(t *testing.T) {

	tests := []struct {
		name string

		topicEnabled bool
		eventType    string

		expectTopicPublish bool
		expectRoutingKey   string
		expectCalled       bool
	}{
		{
			name: "option off",

			topicEnabled: false,
			eventType:    "transcribe_spyoff",

			expectTopicPublish: false,
			expectCalled:       false,
		},
		{
			// the positive control: without it, a spy that never fires would pass the case above
			// even if EventSubscriptionID had been silently unwired everywhere.
			name: "option on",

			topicEnabled: true,
			eventType:    "transcribe_spyon",

			expectTopicPublish: true,
			expectRoutingKey:   "transcribe-manager.transcribe.9f01c3d2-a1bc-11f1-92ef-60452e5e40a2.spyon",
			expectCalled:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			h := &notifyHandler{
				sockHandler:  mockSock,
				queueNotify:  commonoutline.QueueNameTranscribeEvent,
				publisher:    "transcribe-manager",
				topicEnabled: tt.topicEnabled,
			}

			event := &testSpyEvent{
				ID:           "a0121e34-a1bc-11f1-92ef-60452e5e40a2",
				TranscribeID: "9f01c3d2-a1bc-11f1-92ef-60452e5e40a2",
			}

			// VOIP-1407: post-cutover the two arms publish to different exchanges -- the
			// option-off arm still fanout-publishes (topicEnabled=false is unchanged), the
			// option-on arm publishes topic-only (no fanout at all).
			if tt.expectTopicPublish {
				mockSock.EXPECT().EventPublish(string(h.queueNotify), "", gomock.Any()).Times(0)
				mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), tt.expectRoutingKey, gomock.Any()).Return(nil)
			} else {
				mockSock.EXPECT().EventPublish(string(h.queueNotify), "", gomock.Any()).Return(nil)
				mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), gomock.Any(), gomock.Any()).Times(0)
			}

			h.PublishEvent(context.Background(), tt.eventType, event)

			if event.called != tt.expectCalled {
				t.Errorf("Wrong match. expect: %t, got: %t", tt.expectCalled, event.called)
			}
		})
	}
}

// Test_publishEvent_branchSplit is table-driven coverage of publishEvent()'s three-case switch
// (VOIP-1407 design §2.2): `delay > 0` is evaluated first regardless of topicEnabled -- the
// delayed-publish branch never reaches either the fanout or the topic path -- and topicEnabled
// then selects exclusively between the topic-only and fanout-only paths (never both, never
// neither).
func Test_publishEvent_branchSplit(t *testing.T) {
	tests := []struct {
		name string

		topicEnabled bool
		delay        int

		expectFanout  bool
		expectTopic   bool
		expectDelayed bool
	}{
		{
			name:         "topic disabled, no delay -> fanout only",
			topicEnabled: false,
			delay:        0,
			expectFanout: true,
		},
		{
			name:          "topic disabled, delayed -> delayed publish only",
			topicEnabled:  false,
			delay:         DelaySecond,
			expectDelayed: true,
		},
		{
			name:         "topic enabled, no delay -> topic only",
			topicEnabled: true,
			delay:        0,
			expectTopic:  true,
		},
		{
			name:          "topic enabled, delayed -> delayed publish only, delay wins over topicEnabled",
			topicEnabled:  true,
			delay:         DelaySecond,
			expectDelayed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			h := &notifyHandler{
				sockHandler:  mockSock,
				queueNotify:  commonoutline.QueueNameTranscribeEvent,
				publisher:    "transcribe-manager",
				topicEnabled: tt.topicEnabled,
			}

			if tt.expectFanout {
				mockSock.EXPECT().EventPublish(string(h.queueNotify), "", gomock.Any()).Return(nil)
			} else {
				mockSock.EXPECT().EventPublish(string(h.queueNotify), "", gomock.Any()).Times(0)
			}
			if tt.expectTopic {
				mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), gomock.Any(), gomock.Any()).Return(nil)
			} else {
				mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), gomock.Any(), gomock.Any()).Times(0)
			}
			if tt.expectDelayed {
				mockSock.EXPECT().EventPublishWithDelay(string(commonoutline.QueueNameDelay), string(h.queueNotify), gomock.Any(), tt.delay).Return(nil)
			} else {
				mockSock.EXPECT().EventPublishWithDelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}

			event := &testIDEvent{ID: "b6787790-a1bc-11f1-92ef-60452e5e40a2"}
			m, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("Could not marshal the event. err: %v", err)
			}

			if errPublish := h.publishEvent("test_branchsplit", dataTypeJSON, m, requestTimeoutDefault, tt.delay, event.EventSubscriptionID()); errPublish != nil {
				t.Errorf("Wrong match. expected no error. err: %v", errPublish)
			}
		})
	}
}

// Test_publishTopicEventOrErr_observesProcessTime verifies VOIP-1407 design §2.5's decision
// directly: publishTopicEventOrErr observes promNotifyProcessTime under the same metric name and
// `type` label the removed fanout leg used to, on both the success and the failure path.
func Test_publishTopicEventOrErr_observesProcessTime(t *testing.T) {
	tests := []struct {
		name string

		publishErr error
	}{
		{
			name:       "success",
			publishErr: nil,
		},
		{
			name:       "failure",
			publishErr: fmt.Errorf("no route"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			h := &notifyHandler{
				sockHandler:  mockSock,
				queueNotify:  commonoutline.QueueNameTranscribeEvent,
				publisher:    "transcribe-manager",
				topicEnabled: true,
			}

			eventType := "test_processtime_" + tt.name
			evt := &sock.Event{Type: eventType, Publisher: "transcribe-manager", DataType: dataTypeJSON}

			mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), gomock.Any(), evt).Return(tt.publishErr)

			beforeProcessTime := notifyProcessTimeCount(eventType)
			beforeResult := topicPublishCount(eventType, topicPublishResultOK)
			if tt.publishErr != nil {
				beforeResult = topicPublishCount(eventType, topicPublishResultError)
			}

			errPublish := h.publishTopicEventOrErr(context.Background(), evt, "some-subscription-id")
			if tt.publishErr == nil && errPublish != nil {
				t.Errorf("Wrong match. expected no error. err: %v", errPublish)
			}
			if tt.publishErr != nil && errPublish == nil {
				t.Error("Wrong match. expected an error.")
			}

			if afterProcessTime := notifyProcessTimeCount(eventType); afterProcessTime != beforeProcessTime+1 {
				t.Errorf("Wrong match. expect: %d, got: %d", beforeProcessTime+1, afterProcessTime)
			}

			result := topicPublishResultOK
			if tt.publishErr != nil {
				result = topicPublishResultError
			}
			if afterResult := topicPublishCount(eventType, result); afterResult != beforeResult+1 {
				t.Errorf("Wrong match. expect: %f, got: %f", beforeResult+1, afterResult)
			}
		})
	}
}
