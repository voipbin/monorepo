package notifyhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	wmwebhook "monorepo/bin-webhook-manager/models/webhook"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

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

// Test_PublishEvent_globalTopicPublish verifies the dual publish (VOIP-1404): the fanout publish
// runs first and unchanged, then the very same sock.Event is published to the global topic
// exchange with the generated routing key.
func Test_PublishEvent_globalTopicPublish(t *testing.T) {

	tests := []struct {
		name string

		publisher commonoutline.ServiceName
		eventType string
		event     interface{}

		expectEvent      *sock.Event
		expectRoutingKey string
	}{
		{
			name: "subscription id resolved from the top level json id",

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
			name: "subscription id resolved by the subscription identifier override",

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
			name: "no subscription id at all",

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

			// the fanout publish must happen first -- it is the system of record.
			gomock.InOrder(
				mockSock.EXPECT().EventPublish(string(h.queueNotify), "", tt.expectEvent).Return(nil),
				mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), tt.expectRoutingKey, tt.expectEvent).Return(nil),
			)

			beforeOK := topicPublishCount(tt.eventType, topicPublishResultOK)
			beforeNotify := notifyTotalCount(tt.eventType)
			beforeProcessTime := notifyProcessTimeCount(tt.eventType)

			h.PublishEvent(context.Background(), tt.eventType, tt.event)

			if afterOK := topicPublishCount(tt.eventType, topicPublishResultOK); afterOK != beforeOK+1 {
				t.Errorf("Wrong match. expect: %f, got: %f", beforeOK+1, afterOK)
			}

			// metric isolation: the dual publish issues TWO broker publishes but only ONE of them
			// is the fanout publish, so promNotifyTotal must grow by exactly 1. A topic publish
			// that reused publishDirectEvent would double this and silently corrupt the existing
			// fanout metrics (VOIP-1404 code-review round 1, F6).
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
		event     interface{}

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

			mockSock.EXPECT().EventPublish(string(h.queueNotify), "", gomock.Any()).Return(nil)
			mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), gomock.Any(), gomock.Any()).Return(nil)

			before := topicPlaceholderCount(tt.eventType)

			h.PublishEvent(context.Background(), tt.eventType, tt.event)

			if after := topicPlaceholderCount(tt.eventType); after != before+tt.expectPlaceholderDelta {
				t.Errorf("Wrong match. expect: %f, got: %f", before+tt.expectPlaceholderDelta, after)
			}
		})
	}
}

// Test_publishEvent_globalTopicPublishFailureIsolated verifies the failure-isolation contract: a
// topic publish failure must not affect the fanout publish, must not reach the caller, and must
// be counted as an error.
func Test_publishEvent_globalTopicPublishFailureIsolated(t *testing.T) {
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

	gomock.InOrder(
		mockSock.EXPECT().EventPublish(string(h.queueNotify), "", gomock.Any()).Return(nil),
		mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), expectRoutingKey, gomock.Any()).Return(fmt.Errorf("no route")),
	)

	beforeError := topicPublishCount(eventType, topicPublishResultError)
	beforeOK := topicPublishCount(eventType, topicPublishResultOK)

	m, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Could not marshal the event. err: %v", err)
	}

	// the private path is used here so the caller-visible error can be asserted directly --
	// PublishEvent itself swallows it.
	if errPublish := h.publishEvent(eventType, dataTypeJSON, m, requestTimeoutDefault, 0, "", false); errPublish != nil {
		t.Errorf("Wrong match. expected no error from the caller path. err: %v", errPublish)
	}

	if afterError := topicPublishCount(eventType, topicPublishResultError); afterError != beforeError+1 {
		t.Errorf("Wrong match. expect: %f, got: %f", beforeError+1, afterError)
	}
	if afterOK := topicPublishCount(eventType, topicPublishResultOK); afterOK != beforeOK {
		t.Errorf("Wrong match. expect: %f, got: %f", beforeOK, afterOK)
	}
}

// Test_publishEvent_fanoutFailureSkipsTopic verifies that a fanout publish failure skips the topic
// publish entirely -- publishing an event the fanout consumers never saw would diverge state.
func Test_publishEvent_fanoutFailureSkipsTopic(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &notifyHandler{
		sockHandler:  mockSock,
		queueNotify:  commonoutline.QueueNameTranscribeEvent,
		publisher:    "transcribe-manager",
		topicEnabled: true,
	}

	mockSock.EXPECT().EventPublish(string(h.queueNotify), "", gomock.Any()).Return(fmt.Errorf("connection closed"))
	mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), gomock.Any(), gomock.Any()).Times(0)

	err := h.publishEvent("test_fanoutfailed", dataTypeJSON, []byte(`{"id":"d3454367-a1bc-11f1-92ef-60452e5e40a2"}`), requestTimeoutDefault, 0, "", false)
	if err == nil {
		t.Error("Wrong match. expected an error from the fanout publish.")
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

	if err := h.publishEvent("test_delayed", dataTypeJSON, []byte(`{"id":"f5676589-a1bc-11f1-92ef-60452e5e40a2"}`), requestTimeoutDefault, DelaySecond, "", false); err != nil {
		t.Errorf("Wrong match. expected no error. err: %v", err)
	}
}

// Test_PublishEventRaw_globalTopicPublish verifies the raw path: a []byte payload cannot satisfy
// the SubscriptionIdentifier assertion, so the key comes from the payload's top-level "id"
// fallback, and a non-JSON payload lands on the placeholder.
func Test_PublishEventRaw_globalTopicPublish(t *testing.T) {

	tests := []struct {
		name string

		eventType string
		dataType  string
		data      []byte

		expectRoutingKey string
	}{
		{
			name: "json payload with a top level id",

			eventType: "call_created",
			dataType:  "application/json",
			data:      []byte(`{"id":"06787690-a1bc-11f1-92ef-60452e5e40a2","name":"test"}`),

			expectRoutingKey: "call-manager.call.06787690-a1bc-11f1-92ef-60452e5e40a2.created",
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

			gomock.InOrder(
				mockSock.EXPECT().EventPublish(string(h.queueNotify), "", gomock.Any()).Return(nil),
				mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), tt.expectRoutingKey, gomock.Any()).Return(nil),
			)

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

			mockSock.EXPECT().EventPublish(string(h.queueNotify), "", gomock.Any()).Return(nil)
			if tt.expectTopicPublish {
				mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), tt.expectRoutingKey, gomock.Any()).Return(nil)
			} else {
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

// Test_PublishEvent_overrideSuppressesJSONFallback pins design §4.2 (VOIP-1404 code-review round
// 1, F2): when the event data implements SubscriptionIdentifier, that override is authoritative
// even if it produces an empty or uuid.Nil value. The payload here deliberately carries a valid
// top-level "id" -- if the JSON fallback leaked through, the key would carry that id and the
// placeholder counter would stay flat.
func Test_PublishEvent_overrideSuppressesJSONFallback(t *testing.T) {

	tests := []struct {
		name string

		eventType string
		event     interface{}

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

			gomock.InOrder(
				mockSock.EXPECT().EventPublish(string(h.queueNotify), "", gomock.Any()).Return(nil),
				mockSock.EXPECT().EventPublish(string(commonoutline.QueueNameEvent), tt.expectRoutingKey, gomock.Any()).Return(nil),
			)

			before := topicPlaceholderCount(tt.eventType)

			h.PublishEvent(context.Background(), tt.eventType, tt.event)

			if after := topicPlaceholderCount(tt.eventType); after != before+1 {
				t.Errorf("Wrong match. expect: %f, got: %f", before+1, after)
			}
		})
	}
}
