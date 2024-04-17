package subscribehandler

import (
	"testing"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-agent-manager/pkg/agenthandler"
)

func Test_processEvent_processEventCMGroupcallCreated(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		expectGroupcall *cmgroupcall.Groupcall
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
				Publisher: "call-manager",
				Type:      cmgroupcall.EventTypeGroupcallCreated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"1a7889cc-8493-4bad-90ee-b80f944349cb"}`),
			},

			expectGroupcall: &cmgroupcall.Groupcall{
				ID: uuid.FromStringOrNil("1a7889cc-8493-4bad-90ee-b80f944349cb"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := subscribeHandler{
				rabbitSock:   mockSock,
				agentHandler: mockAgent,
			}

			mockAgent.EXPECT().EventGroupcallCreated(gomock.Any(), tt.expectGroupcall).Return(nil)

			h.processEvent(tt.event)
		})
	}
}

func Test_processEvent_processEventCMGroupcallAnswered(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		expectGroupcall *cmgroupcall.Groupcall
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
				Publisher: "call-manager",
				Type:      cmgroupcall.EventTypeGroupcallProgressing,
				DataType:  "application/json",
				Data:      []byte(`{"id":"1a0d744a-c0c2-4a05-8a72-a508a62ce410"}`),
			},

			expectGroupcall: &cmgroupcall.Groupcall{
				ID: uuid.FromStringOrNil("1a0d744a-c0c2-4a05-8a72-a508a62ce410"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := subscribeHandler{
				rabbitSock:   mockSock,
				agentHandler: mockAgent,
			}

			mockAgent.EXPECT().EventGroupcallProgressing(gomock.Any(), tt.expectGroupcall).Return(nil)

			h.processEvent(tt.event)
		})
	}
}
