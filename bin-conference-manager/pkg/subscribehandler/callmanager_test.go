package subscribehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/conferencecallhandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/conferencehandler"
)

func Test_processEventCMConfbridgeJoined(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		responseConferencecall *conferencecall.Conferencecall

		expectConferencecallID uuid.UUID
		expectRes              *rabbitmqhandler.Response
	}{
		{
			"type conference",
			&rabbitmqhandler.Event{
				Type:      cmconfbridge.EventTypeConfbridgeJoined,
				Publisher: "call-manager",
				DataType:  "application/json",
				Data:      []byte(`{"id":"2a8739a2-9368-11ed-82dd-bfa0ae5f78fb","joined_call_id":"2abecb4c-9368-11ed-9130-b74b5a76b8d3"}`),
			},
			&conferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("18033654-9102-11ed-994e-4b9c733834a5"),
			},

			uuid.FromStringOrNil("2abecb4c-9368-11ed-9130-b74b5a76b8d3"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockConfcall := conferencecallhandler.NewMockConferencecallHandler(mc)

			h := &subscribeHandler{
				rabbitSock:            mockSock,
				conferencecallHandler: mockConfcall,
			}

			mockConfcall.EXPECT().GetByReferenceID(gomock.Any(), tt.expectConferencecallID).Return(tt.responseConferencecall, nil)
			mockConfcall.EXPECT().Joined(gomock.Any(), tt.responseConferencecall).Return(tt.responseConferencecall, nil)
			h.processEvent(tt.event)
		})
	}
}

func Test_processEventCMConfbridgeLeaved(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		responseConferencecall *conferencecall.Conferencecall

		expectConferenceID uuid.UUID
		expectCallID       uuid.UUID
		expectRes          *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Event{
				Type:      cmconfbridge.EventTypeConfbridgeLeaved,
				Publisher: "call-manager",
				DataType:  "application/json",
				Data:      []byte(`{"id":"3ea3ebe6-9369-11ed-b4e3-075af58c7edb","leaved_call_id":"3ec70a68-9369-11ed-bdfa-efc27d3a6df7"}`),
			},

			&conferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("417fb8ae-9369-11ed-aa39-5fa8ce8a29d4"),
			},

			uuid.FromStringOrNil("3ea3ebe6-9369-11ed-b4e3-075af58c7edb"),
			uuid.FromStringOrNil("3ec70a68-9369-11ed-bdfa-efc27d3a6df7"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)
			mockConfCall := conferencecallhandler.NewMockConferencecallHandler(mc)

			h := &subscribeHandler{
				rabbitSock:            mockSock,
				conferenceHandler:     mockConf,
				conferencecallHandler: mockConfCall,
			}

			mockConfCall.EXPECT().GetByReferenceID(gomock.Any(), tt.expectCallID).Return(tt.responseConferencecall, nil)
			mockConfCall.EXPECT().Terminated(gomock.Any(), tt.responseConferencecall).Return(tt.responseConferencecall, nil)
			h.processEvent(tt.event)
		})
	}
}
