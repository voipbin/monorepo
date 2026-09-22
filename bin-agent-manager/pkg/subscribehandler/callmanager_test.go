package subscribehandler

import (
	"testing"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/pkg/agenthandler"
)

func Test_processEvent_processEventCMGroupcallAnswered(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectGroupcall *cmgroupcall.Groupcall
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "call-manager",
				Type:      cmgroupcall.EventTypeGroupcallProgressing,
				DataType:  "application/json",
				Data:      []byte(`{"id":"1a0d744a-c0c2-4a05-8a72-a508a62ce410"}`),
			},

			expectGroupcall: &cmgroupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1a0d744a-c0c2-4a05-8a72-a508a62ce410"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := subscribeHandler{
				sockHandler:  mockSock,
				agentHandler: mockAgent,
			}

			mockAgent.EXPECT().EventGroupcallProgressing(gomock.Any(), tt.expectGroupcall).Return(nil)

			h.processEvent(tt.event)
		})
	}
}
