package subscribehandler

import (
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-flow-manager/pkg/activeflowhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_processEvent_processEventCMCallHangup(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectedCall *cmcall.Call
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "call-manager",
				Type:      cmcall.EventTypeCallHangup,
				DataType:  "application/json",
				Data:      []byte(`{"id":"561df120-ecd9-11ee-9355-b7deef352acc"}`),
			},

			expectedCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("561df120-ecd9-11ee-9355-b7deef352acc"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockActive := activeflowhandler.NewMockActiveflowHandler(mc)

			h := subscribeHandler{
				sockHandler:       mockSock,
				activeflowHandler: mockActive,
			}

			mockActive.EXPECT().EventCallHangup(gomock.Any(), tt.expectedCall).Return(nil)

			h.processEvent(tt.event)
		})
	}
}
