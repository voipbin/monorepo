package subscribehandler

import (
	"fmt"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/pkg/billinghandler"
	"monorepo/bin-billing-manager/pkg/failedeventhandler"
)

func Test_processEventRun_success(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event
	}{
		{
			name: "successful event processing does not save",
			event: &sock.Event{
				Publisher: "call-manager",
				Type:      cmcall.EventTypeCallProgressing,
				DataType:  "application/json",
				Data:      []byte(`{"id":"aa000001-0000-0000-0000-000000000001"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockBilling := billinghandler.NewMockBillingHandler(mc)
			mockFailed := failedeventhandler.NewMockFailedEventHandler(mc)

			h := subscribeHandler{
				sockHandler:        mockSock,
				billingHandler:     mockBilling,
				failedEventHandler: mockFailed,
			}

			mockBilling.EXPECT().EventCMCallProgressing(gomock.Any(), gomock.Any()).Return(nil)
			// failedEventHandler.Save should NOT be called

			if err := h.processEventRun(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEventRun_failure_saves_event(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event
	}{
		{
			name: "failed event processing saves to failed events",
			event: &sock.Event{
				Publisher: "call-manager",
				Type:      cmcall.EventTypeCallProgressing,
				DataType:  "application/json",
				Data:      []byte(`{"id":"bb000001-0000-0000-0000-000000000001"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockBilling := billinghandler.NewMockBillingHandler(mc)
			mockFailed := failedeventhandler.NewMockFailedEventHandler(mc)

			h := subscribeHandler{
				sockHandler:        mockSock,
				billingHandler:     mockBilling,
				failedEventHandler: mockFailed,
			}

			processErr := fmt.Errorf("processing failed")
			mockBilling.EXPECT().EventCMCallProgressing(gomock.Any(), gomock.Any()).Return(processErr)
			mockFailed.EXPECT().Save(gomock.Any(), tt.event, gomock.Any()).Return(nil)

			if err := h.processEventRun(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEventRun_failure_save_also_fails(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event
	}{
		{
			name: "both processing and save fail - still returns nil",
			event: &sock.Event{
				Publisher: "call-manager",
				Type:      cmcall.EventTypeCallHangup,
				DataType:  "application/json",
				Data:      []byte(`{"id":"cc000001-0000-0000-0000-000000000001"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockBilling := billinghandler.NewMockBillingHandler(mc)
			mockFailed := failedeventhandler.NewMockFailedEventHandler(mc)

			h := subscribeHandler{
				sockHandler:        mockSock,
				billingHandler:     mockBilling,
				failedEventHandler: mockFailed,
			}

			processErr := fmt.Errorf("processing failed")
			mockBilling.EXPECT().EventCMCallHangup(gomock.Any(), gomock.Any()).Return(processErr)
			mockFailed.EXPECT().Save(gomock.Any(), tt.event, gomock.Any()).Return(fmt.Errorf("save failed"))

			// processEventRun always returns nil to avoid nacking the message
			if err := h.processEventRun(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEventRun_unknown_event(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockBilling := billinghandler.NewMockBillingHandler(mc)
	mockFailed := failedeventhandler.NewMockFailedEventHandler(mc)

	h := subscribeHandler{
		sockHandler:        mockSock,
		billingHandler:     mockBilling,
		failedEventHandler: mockFailed,
	}

	event := &sock.Event{
		Publisher: "unknown-service",
		Type:      "unknown-type",
		DataType:  "application/json",
		Data:      []byte(`{}`),
	}

	// unknown events are silently ignored - no Save, no error
	if err := h.processEventRun(event); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
