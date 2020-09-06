package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/channel"
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
				Data:   []byte(`{"retry_count": 0, "delay": 10}`),
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
				Data:     []byte(`{"action_id": "ec4c8192-994b-11ea-ab64-9b63b984b7c4", "action_type": "echo", "tm_execute": "2020-05-03T21:35:02.809"}`),
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

			mockCall.EXPECT().ActionTimeout(tt.id, tt.action).Return(nil)

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

func TestProcessV1CallsIDPost(t *testing.T) {
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
		call      *call.Call
		request   *rabbitmq.Request
		expectRes *rabbitmq.Response
	}

	tests := []test{
		{
			"empty addresses",
			&call.Call{
				ID:          uuid.FromStringOrNil("47a468d4-ed66-11ea-be25-97f0d867d634"),
				UserID:      1,
				FlowID:      uuid.FromStringOrNil("59518eae-ed66-11ea-85ef-b77bdbc74ccc"),
				Source:      call.Address{},
				Destination: call.Address{},
			},

			&rabbitmq.Request{
				URI:      "/v1/calls/47a468d4-ed66-11ea-be25-97f0d867d634",
				Method:   rabbitmq.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "flow_id": "59518eae-ed66-11ea-85ef-b77bdbc74ccc", "source": {}, "destination": {}}`),
			},
			&rabbitmq.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"47a468d4-ed66-11ea-be25-97f0d867d634","user_id":1,"asterisk_id":"","channel_id":"","flow_id":"59518eae-ed66-11ea-85ef-b77bdbc74ccc","conf_id":"00000000-0000-0000-0000-000000000000","type":"","source":{"type":"","target":"","name":""},"destination":{"type":"","target":"","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":"","next":"00000000-0000-0000-0000-000000000000","tm_execute":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
		},
		{
			"source address",
			&call.Call{
				ID:     uuid.FromStringOrNil("47a468d4-ed66-11ea-be25-97f0d867d634"),
				UserID: 1,
				FlowID: uuid.FromStringOrNil("59518eae-ed66-11ea-85ef-b77bdbc74ccc"),
				Source: call.Address{
					Type:   call.AddressTypeSIP,
					Target: "test_source@127.0.0.1:5061",
					Name:   "test_source",
				},
				Destination: call.Address{},
			},

			&rabbitmq.Request{
				URI:      "/v1/calls/47a468d4-ed66-11ea-be25-97f0d867d634",
				Method:   rabbitmq.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "flow_id": "59518eae-ed66-11ea-85ef-b77bdbc74ccc", "source": {"type": "sip", "target": "test_source@127.0.0.1:5061", "name": "test_source"}, "destination": {}}`),
			},
			&rabbitmq.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"47a468d4-ed66-11ea-be25-97f0d867d634","user_id":1,"asterisk_id":"","channel_id":"","flow_id":"59518eae-ed66-11ea-85ef-b77bdbc74ccc","conf_id":"00000000-0000-0000-0000-000000000000","type":"","source":{"type":"sip","target":"test_source@127.0.0.1:5061","name":"test_source"},"destination":{"type":"","target":"","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":"","next":"00000000-0000-0000-0000-000000000000","tm_execute":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
		},
		{
			"flow_id null",
			&call.Call{
				ID:     uuid.FromStringOrNil("f93eef0c-ed79-11ea-85cb-b39596cdf7ff"),
				UserID: 1,
				FlowID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
				Source: call.Address{
					Type:   call.AddressTypeSIP,
					Target: "test_source@127.0.0.1:5061",
					Name:   "test_source",
				},
				Destination: call.Address{
					Type:   call.AddressTypeSIP,
					Target: "test_destination@127.0.0.1:5061",
					Name:   "test_destination",
				},
			},

			&rabbitmq.Request{
				URI:      "/v1/calls/f93eef0c-ed79-11ea-85cb-b39596cdf7ff",
				Method:   rabbitmq.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "flow_id": "00000000-0000-0000-0000-000000000000","source": {"type": "sip","target": "test_source@127.0.0.1:5061","name": "test_source"},"destination": {"type": "sip","target": "test_destination@127.0.0.1:5061","name": "test_destination"}}`),
			},
			&rabbitmq.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f93eef0c-ed79-11ea-85cb-b39596cdf7ff","user_id":1,"asterisk_id":"","channel_id":"","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"","source":{"type":"sip","target":"test_source@127.0.0.1:5061","name":"test_source"},"destination":{"type":"sip","target":"test_destination@127.0.0.1:5061","name":"test_destination"},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":"","next":"00000000-0000-0000-0000-000000000000","tm_execute":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockCall.EXPECT().CreateCallOutgoing(tt.call.ID, tt.call.UserID, tt.call.FlowID, tt.call.Source, tt.call.Destination).Return(tt.call, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
