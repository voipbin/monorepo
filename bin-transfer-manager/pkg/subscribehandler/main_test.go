package subscribehandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-transfer-manager/pkg/transferhandler"
)

func TestNewSubscribeHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTransfer := transferhandler.NewMockTransferHandler(mc)

	tests := []struct {
		name             string
		serviceName      string
		subscribeQueue   string
		subscribeTargets []string
	}{
		{
			name:             "creates_handler_successfully",
			serviceName:      "transfer-manager",
			subscribeQueue:   "test-queue",
			subscribeTargets: []string{"target1", "target2"},
		},
		{
			name:             "creates_handler_with_empty_targets",
			serviceName:      "transfer-manager",
			subscribeQueue:   "test-queue",
			subscribeTargets: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewSubscribeHandler(
				tt.serviceName,
				mockSock,
				tt.subscribeQueue,
				tt.subscribeTargets,
				mockTransfer,
			)

			if h == nil {
				t.Error("Expected handler but got nil")
			}
		})
	}
}

func TestProcessEventRun(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTransfer := transferhandler.NewMockTransferHandler(mc)

	h := &subscribeHandler{
		serviceName:      "transfer-manager",
		sockHandler:      mockSock,
		subscribeQueue:   "test-queue",
		subscribeTargets: []string{},
		transferHandler:  mockTransfer,
	}

	tests := []struct {
		name  string
		event *sock.Event
	}{
		{
			name: "processes_event_without_error",
			event: &sock.Event{
				Publisher: "unknown-publisher",
				Type:      "unknown-type",
				Data:      []byte(`{}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := h.processEventRun(tt.event)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
