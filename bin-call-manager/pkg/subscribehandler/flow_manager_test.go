package subscribehandler

import (
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/pkg/arieventhandler"
	"monorepo/bin-call-manager/pkg/callhandler"
)

func Test_processEvent_processEventActiveflowStop(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectedActiveflow *fmactiveflow.Activeflow
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "flow-manager",
				Type:      "activeflow_updated",
				DataType:  "application/json",
				Data:      []byte(`{"id":"e739a280-f161-11ee-8444-2385d7cef78a"}`),
			},

			expectedActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e739a280-f161-11ee-8444-2385d7cef78a"),
				},
			},
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

			mockCall.EXPECT().EventFMActiveflowUpdated(gomock.Any(), tt.expectedActiveflow.Return(nil)

			if errProcess := h.processEvent(tt.event); errProcess != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errProcess)
			}
		})
	}
}
