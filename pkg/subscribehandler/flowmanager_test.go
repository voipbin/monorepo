package subscribehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/arieventhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
)

func Test_processEvent_processEventActiveflowStop(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		expectCallID uuid.UUID
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
				Publisher: "flow-manager",
				Type:      "activeflow_updated",
				DataType:  "application/json",
				Data:      []byte(`{"status":"ended","reference_type":"call","reference_id":"376c0f2e-b5f7-11ed-ad49-13b40b99b414"}`),
			},

			expectCallID: uuid.FromStringOrNil("376c0f2e-b5f7-11ed-ad49-13b40b99b414"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := subscribeHandler{
				rabbitSock:      mockSock,
				ariEventHandler: mockARIEvent,
				callHandler:     mockCall,
			}

			mockCall.EXPECT().HangingUp(gomock.Any(), tt.expectCallID, call.HangupReasonNormal).Return(&call.Call{}, nil)

			if errProcess := h.processEvent(tt.event); errProcess != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errProcess)
			}
		})
	}
}
