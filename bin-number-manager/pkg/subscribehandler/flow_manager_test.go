package subscribehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandler"
)

func Test_processEvent_processEventFMFlowDeleted(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		expectFlow *fmflow.Flow
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
				Publisher: "flow-manager",
				Type:      fmflow.EventTypeFlowDeleted,
				DataType:  "application/json",
				Data:      []byte(`{"id":"7d08051c-2d64-11ee-92d1-bf5dc689d1d5"}`),
			},

			expectFlow: &fmflow.Flow{
				ID: uuid.FromStringOrNil("7d08051c-2d64-11ee-92d1-bf5dc689d1d5"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockNumber := numberhandler.NewMockNumberHandler(mc)

			h := subscribeHandler{
				rabbitSock:    mockSock,
				numberHandler: mockNumber,
			}

			mockNumber.EXPECT().EventFlowDeleted(gomock.Any(), tt.expectFlow).Return(nil)

			h.processEvent(tt.event)
		})
	}
}
