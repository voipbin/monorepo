package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

func TestProcessV1CallsIDHealthPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &listenHandler{
		rabbitSock: mockSock,
		db:         mockDB,
		reqHandler: mockReq,
	}

	type test struct {
		name    string
		call    *call.Call
		request *rabbitmq.Request
	}

	tests := []test{
		{
			"normal test",
			&call.Call{
				ID:         uuid.FromStringOrNil("1a94c1e6-982e-11ea-9298-43412daaf0da"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "94490ad8-982e-11ea-959d-b3d42fe73e00",
			},
			&rabbitmq.Request{
				URI:    "/v1/calls/1a94c1e6-982e-11ea-9298-43412daaf0da/health-check",
				Method: rabbitmq.RequestMethodPost,
				Data:   `{"retry_count": 0, "delay": 10}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
			mockReq.EXPECT().AstChannelGet(tt.call.AsteriskID, tt.call.ChannelID).Return(&channel.Channel{}, nil)
			mockReq.EXPECT().CallCallHealth(tt.call.ID, 10, 0).Return(nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			} else if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}

		})
	}
}

func TestProcessV1CallsIDActionTimeoutPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmq.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
		db:          mockDB,
		reqHandler:  mockReq,
		callHandler: mockCall,
	}

	type test struct {
		name      string
		id        uuid.UUID
		request   *rabbitmq.Request
		action    *action.Action
		expectRes *rabbitmq.Response
	}

	tests := []test{
		{
			"normal test",
			uuid.FromStringOrNil("1a94c1e6-982e-11ea-9298-43412daaf0da"),
			&rabbitmq.Request{
				URI:      "/v1/calls/1a94c1e6-982e-11ea-9298-43412daaf0da/action-timeout",
				Method:   rabbitmq.RequestMethodPost,
				DataType: "application/json",
				Data:     `{"id": "ec4c8192-994b-11ea-ab64-9b63b984b7c4", "type": "echo", "tm_execute": "2020-05-03T21:35:02.809"}`,
			},
			&action.Action{
				ID:        uuid.FromStringOrNil("ec4c8192-994b-11ea-ab64-9b63b984b7c4"),
				Type:      action.TypeEcho,
				Next:      action.IDEnd,
				TMExecute: "2020-05-03T21:35:02.809",
			},
			&rabbitmq.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockCall.EXPECT().ActionTimeout(tt.id, tt.action)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match. exepct: 200, got: %v", res)
			}
		})
	}
}
