package subscribehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/agenthandler"
	cmgroupdial "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupdial"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_processEvent_processEventCMGroupdialCreated(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		expectGroupdial *cmgroupdial.Groupdial
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
				Publisher: "call-manager",
				Type:      cmgroupdial.EventTypeGroupdialCreated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"1a7889cc-8493-4bad-90ee-b80f944349cb"}`),
			},

			expectGroupdial: &cmgroupdial.Groupdial{
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

			mockAgent.EXPECT().EventGroupdialCreated(gomock.Any(), tt.expectGroupdial).Return(nil)

			h.processEvent(tt.event)
		})
	}
}

func Test_processEvent_processEventCMGroupdialAnswered(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		expectGroupdial *cmgroupdial.Groupdial
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
				Publisher: "call-manager",
				Type:      cmgroupdial.EventTypeGroupdialAnswered,
				DataType:  "application/json",
				Data:      []byte(`{"id":"1a0d744a-c0c2-4a05-8a72-a508a62ce410"}`),
			},

			expectGroupdial: &cmgroupdial.Groupdial{
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

			mockAgent.EXPECT().EventGroupdialAnswered(gomock.Any(), tt.expectGroupdial).Return(nil)

			h.processEvent(tt.event)
		})
	}
}
