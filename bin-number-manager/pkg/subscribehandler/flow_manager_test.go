package subscribehandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

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
				sockHandler:   mockSock,
				numberHandler: mockNumber,
			}

			mockNumber.EXPECT().EventFlowDeleted(gomock.Any(), tt.expectFlow).Return(nil)

			h.processEvent(tt.event)
		})
	}
}
