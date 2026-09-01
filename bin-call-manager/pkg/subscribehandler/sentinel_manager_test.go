package subscribehandler

import (
	"testing"

	"monorepo/bin-call-manager/pkg/arieventhandler"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	smcontainer "monorepo/bin-sentinel-manager/models/container"

	gomock "go.uber.org/mock/gomock"
)

func Test_processEvent_processEventSMContainerDied(t *testing.T) {
	tests := []struct {
		name string

		event *sock.Event

		expectedContainer *smcontainer.Event
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "sentinel-manager",
				Type:      smcontainer.EventTypeContainerDied,
				DataType:  "application/json",
				Data:      []byte(`{"container_name":"voip-asterisk-call-docker-1","service":"asterisk-call","asterisk_id":"3e:50:6b:43:bb:32"}`),
			},

			expectedContainer: &smcontainer.Event{
				ContainerName: "voip-asterisk-call-docker-1",
				Service:       smcontainer.ServiceAsteriskCall,
				AsteriskID:    "3e:50:6b:43:bb:32",
			},
		},
		{
			name: "unresolved asterisk id is forwarded to the handler, which guards it",

			event: &sock.Event{
				Publisher: "sentinel-manager",
				Type:      smcontainer.EventTypeContainerDied,
				DataType:  "application/json",
				Data:      []byte(`{"container_name":"voip-asterisk-call-docker-2","service":"asterisk-call","asterisk_id":""}`),
			},

			expectedContainer: &smcontainer.Event{
				ContainerName: "voip-asterisk-call-docker-2",
				Service:       smcontainer.ServiceAsteriskCall,
				AsteriskID:    "",
			},
		},
		{
			name: "non call service is forwarded to the handler, which filters it",

			event: &sock.Event{
				Publisher: "sentinel-manager",
				Type:      smcontainer.EventTypeContainerDied,
				DataType:  "application/json",
				Data:      []byte(`{"container_name":"voip-asterisk-registrar-docker-1","service":"asterisk-registrar","asterisk_id":"aa:bb:cc:dd:ee:ff"}`),
			},

			expectedContainer: &smcontainer.Event{
				ContainerName: "voip-asterisk-registrar-docker-1",
				Service:       smcontainer.ServiceAsteriskRegistrar,
				AsteriskID:    "aa:bb:cc:dd:ee:ff",
			},
		},
		{
			// This is the real path to a nil payload: `json.Unmarshal` of a literal `null` into a
			// **Event SUCCEEDS and leaves the pointer nil. callhandler's EventSMContainerDied
			// carries the nil guard that keeps this from panicking the subscribe loop.
			name: "null payload unmarshals to a nil event",

			event: &sock.Event{
				Publisher: "sentinel-manager",
				Type:      smcontainer.EventTypeContainerDied,
				DataType:  "application/json",
				Data:      []byte(`null`),
			},

			expectedContainer: nil,
		},
		{
			name: "missing fields unmarshal to their zero values",

			event: &sock.Event{
				Publisher: "sentinel-manager",
				Type:      smcontainer.EventTypeContainerDied,
				DataType:  "application/json",
				Data:      []byte(`{}`),
			},

			expectedContainer: &smcontainer.Event{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := subscribeHandler{
				sockHandler:     mockSock,
				ariEventHandler: mockARIEvent,
				callHandler:     mockCall,
			}

			mockCall.EXPECT().EventSMContainerDied(gomock.Any(), tt.expectedContainer).Return(nil)

			if errProcess := h.processEvent(tt.event); errProcess != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errProcess)
			}
		})
	}
}

// Test_processEvent_processEventSMContainerDied_ignoresOtherEventTypes pins the dispatch guard:
// the old `pod_deleted` wire string must no longer route anywhere. The strict mock (no
// EventSMContainerDied expectation) is the assertion.
func Test_processEvent_processEventSMContainerDied_ignoresOtherEventTypes(t *testing.T) {
	tests := []struct {
		name string

		event *sock.Event
	}{
		{
			name: "retired pod_deleted event type",

			event: &sock.Event{
				Publisher: "sentinel-manager",
				Type:      "pod_deleted",
				DataType:  "application/json",
				Data:      []byte(`{"metadata":{"annotations":{"asterisk-id":"3e:50:6b:43:bb:32"}}}`),
			},
		},
		{
			name: "container_started is not consumed",

			event: &sock.Event{
				Publisher: "sentinel-manager",
				Type:      smcontainer.EventTypeContainerStarted,
				DataType:  "application/json",
				Data:      []byte(`{"container_name":"voip-asterisk-call-docker-1","service":"asterisk-call","asterisk_id":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := subscribeHandler{
				sockHandler:     sockhandler.NewMockSockHandler(mc),
				ariEventHandler: arieventhandler.NewMockARIEventHandler(mc),
				callHandler:     callhandler.NewMockCallHandler(mc),
			}

			if errProcess := h.processEvent(tt.event); errProcess != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errProcess)
			}
		})
	}
}

// Test_processEventSMContainerDied_malformedPayload pins that a malformed payload surfaces as an
// error rather than being silently dispatched with a zero-valued event.
func Test_processEventSMContainerDied_malformedPayload(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := subscribeHandler{
		sockHandler:     sockhandler.NewMockSockHandler(mc),
		ariEventHandler: arieventhandler.NewMockARIEventHandler(mc),
		callHandler:     callhandler.NewMockCallHandler(mc),
	}

	event := &sock.Event{
		Publisher: "sentinel-manager",
		Type:      smcontainer.EventTypeContainerDied,
		DataType:  "application/json",
		Data:      []byte(`{"container_name":`),
	}

	if errProcess := h.processEvent(event); errProcess == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}
