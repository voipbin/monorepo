package subscribehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/activeflowhandler"
)

func Test_processEvent_processEventCMCallHangup(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		expectCall *cmcall.Call
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
				Publisher: "call-manager",
				Type:      cmcall.EventTypeCallHangup,
				DataType:  "application/json",
				Data:      []byte(`{"id":"561df120-ecd9-11ee-9355-b7deef352acc"}`),
			},

			expectCall: &cmcall.Call{
				ID: uuid.FromStringOrNil("561df120-ecd9-11ee-9355-b7deef352acc"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockActive := activeflowhandler.NewMockActiveflowHandler(mc)

			h := subscribeHandler{
				rabbitSock:        mockSock,
				activeflowHandler: mockActive,
			}

			mockActive.EXPECT().EventCallHangup(gomock.Any(), tt.expectCall).Return(nil)

			h.processEvent(tt.event)
		})
	}
}
