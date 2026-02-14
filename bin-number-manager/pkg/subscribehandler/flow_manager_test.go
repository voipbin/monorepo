package subscribehandler

import (
	"context"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/pkg/numberhandler"
)

func Test_processEvent_processEventFMFlowDeleted(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectFlow *fmflow.Flow
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "flow-manager",
				Type:      fmflow.EventTypeFlowDeleted,
				DataType:  "application/json",
				Data:      []byte(`{"id":"7d08051c-2d64-11ee-92d1-bf5dc689d1d5"}`),
			},

			expectFlow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7d08051c-2d64-11ee-92d1-bf5dc689d1d5"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockNumber := numberhandler.NewMockNumberHandler(mc)

			h := subscribeHandler{
				sockHandler:   mockSock,
				numberHandler: mockNumber,
			}

			mockNumber.EXPECT().EventFlowDeleted(gomock.Any(), tt.expectFlow).Return(nil)

			h.processEvent(tt.event)
		})
	}
}

func Test_processEventFMFlowDeleted_UnmarshalError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := subscribeHandler{
		sockHandler:   mockSock,
		numberHandler: mockNumber,
	}

	event := &sock.Event{
		Publisher: "flow-manager",
		Type:      fmflow.EventTypeFlowDeleted,
		DataType:  "application/json",
		Data:      []byte(`{invalid json`),
	}

	err := h.processEventFMFlowDeleted(context.Background(), event)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func Test_processEventFMFlowDeleted_HandlerError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := subscribeHandler{
		sockHandler:   mockSock,
		numberHandler: mockNumber,
	}

	flow := &fmflow.Flow{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("7d08051c-2d64-11ee-92d1-bf5dc689d1d5"),
		},
	}

	event := &sock.Event{
		Publisher: "flow-manager",
		Type:      fmflow.EventTypeFlowDeleted,
		DataType:  "application/json",
		Data:      []byte(`{"id":"7d08051c-2d64-11ee-92d1-bf5dc689d1d5"}`),
	}

	mockNumber.EXPECT().EventFlowDeleted(gomock.Any(), flow).Return(fmt.Errorf("handler error"))

	err := h.processEventFMFlowDeleted(context.Background(), event)
	if err == nil {
		t.Error("Expected error from handler, got nil")
	}
}
