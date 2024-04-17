package subscribehandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/pkg/arieventhandler"
	"monorepo/bin-call-manager/pkg/callhandler"
)

func Test_processEvent_processEventActiveflowStop(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		exectActiveflow *fmactiveflow.Activeflow
		// expectCallID uuid.UUID
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
				Publisher: "flow-manager",
				Type:      "activeflow_updated",
				DataType:  "application/json",
				Data:      []byte(`{"id":"e739a280-f161-11ee-8444-2385d7cef78a"}`),
			},

			exectActiveflow: &fmactiveflow.Activeflow{
				ID: uuid.FromStringOrNil("e739a280-f161-11ee-8444-2385d7cef78a"),
			},
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

			mockCall.EXPECT().EventFMActiveflowUpdated(gomock.Any(), tt.exectActiveflow).Return(nil)

			if errProcess := h.processEvent(tt.event); errProcess != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errProcess)
			}
		})
	}
}
