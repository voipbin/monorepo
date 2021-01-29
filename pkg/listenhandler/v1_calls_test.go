package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/action"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestProcessV1CallsIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &listenHandler{
		rabbitSock: mockSock,
		db:         mockDB,
		reqHandler: mockReq,
	}

	type test struct {
		name      string
		request   *rabbitmqhandler.Request
		call      *call.Call
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"basic",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls/638769c2-620d-11eb-bd1f-6b576e26b4e6",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&call.Call{
				ID:     uuid.FromStringOrNil("638769c2-620d-11eb-bd1f-6b576e26b4e6"),
				UserID: 0,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"638769c2-620d-11eb-bd1f-6b576e26b4e6","user_id":0,"asterisk_id":"","channel_id":"","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":"","tm_execute":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)

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

func TestProcessV1CallsGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &listenHandler{
		rabbitSock: mockSock,
		db:         mockDB,
		reqHandler: mockReq,
	}

	type test struct {
		name      string
		request   *rabbitmqhandler.Request
		userID    uint64
		pageSize  uint64
		pageToken string
		calls     []*call.Call
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"basic",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls?page_size=10&page_token=2020-05-03%2021:35:02.809&user_id=1",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			1,
			10,
			"2020-05-03 21:35:02.809",
			[]*call.Call{
				&call.Call{
					ID:     uuid.FromStringOrNil("866ad964-620e-11eb-9f09-9fab48a7edd3"),
					UserID: 1,
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"866ad964-620e-11eb-9f09-9fab48a7edd3","user_id":1,"asterisk_id":"","channel_id":"","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":"","tm_execute":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallGets(gomock.Any(), tt.userID, tt.pageSize, tt.pageToken).Return(tt.calls, nil)
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

func TestProcessV1CallsIDHealthPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
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
		request *rabbitmqhandler.Request
	}

	tests := []test{
		{
			"normal test",
			&call.Call{
				ID:         uuid.FromStringOrNil("1a94c1e6-982e-11ea-9298-43412daaf0da"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "94490ad8-982e-11ea-959d-b3d42fe73e00",
			},
			&rabbitmqhandler.Request{
				URI:    "/v1/calls/1a94c1e6-982e-11ea-9298-43412daaf0da/health-check",
				Method: rabbitmqhandler.RequestMethodPost,
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

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
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
		request   *rabbitmqhandler.Request
		action    *action.Action
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal test",
			uuid.FromStringOrNil("1a94c1e6-982e-11ea-9298-43412daaf0da"),
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/1a94c1e6-982e-11ea-9298-43412daaf0da/action-timeout",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"action_id": "ec4c8192-994b-11ea-ab64-9b63b984b7c4", "action_type": "echo", "tm_execute": "2020-05-03T21:35:02.809"}`),
			},
			&action.Action{
				ID:        uuid.FromStringOrNil("ec4c8192-994b-11ea-ab64-9b63b984b7c4"),
				Type:      action.TypeEcho,
				TMExecute: "2020-05-03T21:35:02.809",
			},
			&rabbitmqhandler.Response{
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

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
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
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
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

			&rabbitmqhandler.Request{
				URI:      "/v1/calls/47a468d4-ed66-11ea-be25-97f0d867d634",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "flow_id": "59518eae-ed66-11ea-85ef-b77bdbc74ccc", "source": {}, "destination": {}}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"47a468d4-ed66-11ea-be25-97f0d867d634","user_id":1,"asterisk_id":"","channel_id":"","flow_id":"59518eae-ed66-11ea-85ef-b77bdbc74ccc","conf_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":"","tm_execute":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
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

			&rabbitmqhandler.Request{
				URI:      "/v1/calls/47a468d4-ed66-11ea-be25-97f0d867d634",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "flow_id": "59518eae-ed66-11ea-85ef-b77bdbc74ccc", "source": {"type": "sip", "target": "test_source@127.0.0.1:5061", "name": "test_source"}, "destination": {}}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"47a468d4-ed66-11ea-be25-97f0d867d634","user_id":1,"asterisk_id":"","channel_id":"","flow_id":"59518eae-ed66-11ea-85ef-b77bdbc74ccc","conf_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"sip","target":"test_source@127.0.0.1:5061","name":"test_source"},"destination":{"type":"","target":"","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":"","tm_execute":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
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

			&rabbitmqhandler.Request{
				URI:      "/v1/calls/f93eef0c-ed79-11ea-85cb-b39596cdf7ff",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "flow_id": "00000000-0000-0000-0000-000000000000","source": {"type": "sip","target": "test_source@127.0.0.1:5061","name": "test_source"},"destination": {"type": "sip","target": "test_destination@127.0.0.1:5061","name": "test_destination"}}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f93eef0c-ed79-11ea-85cb-b39596cdf7ff","user_id":1,"asterisk_id":"","channel_id":"","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"sip","target":"test_source@127.0.0.1:5061","name":"test_source"},"destination":{"type":"sip","target":"test_destination@127.0.0.1:5061","name":"test_destination"},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":"","tm_execute":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
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

func TestProcessV1CallsPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
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
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"empty addresses",
			&call.Call{
				ID:          uuid.FromStringOrNil("72d56d08-f3a8-11ea-9c0c-ef8258d54f42"),
				UserID:      1,
				FlowID:      uuid.FromStringOrNil("78fd1276-f3a8-11ea-9734-6735e73fd720"),
				Source:      call.Address{},
				Destination: call.Address{},
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/calls",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "flow_id": "78fd1276-f3a8-11ea-9734-6735e73fd720", "source": {}, "destination": {}}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"72d56d08-f3a8-11ea-9c0c-ef8258d54f42","user_id":1,"asterisk_id":"","channel_id":"","flow_id":"78fd1276-f3a8-11ea-9734-6735e73fd720","conf_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":"","tm_execute":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
		},
		{
			"source address",
			&call.Call{
				ID:     uuid.FromStringOrNil("cd561ba6-f3a8-11ea-b7ac-57b19fa28e09"),
				UserID: 1,
				FlowID: uuid.FromStringOrNil("d4df6ed6-f3a8-11ea-bf19-6f8063fdcfa1"),
				Source: call.Address{
					Type:   call.AddressTypeSIP,
					Target: "test_source@127.0.0.1:5061",
					Name:   "test_source",
				},
				Destination: call.Address{},
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/calls",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "flow_id": "d4df6ed6-f3a8-11ea-bf19-6f8063fdcfa1", "source": {"type": "sip", "target": "test_source@127.0.0.1:5061", "name": "test_source"}, "destination": {}}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"cd561ba6-f3a8-11ea-b7ac-57b19fa28e09","user_id":1,"asterisk_id":"","channel_id":"","flow_id":"d4df6ed6-f3a8-11ea-bf19-6f8063fdcfa1","conf_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"sip","target":"test_source@127.0.0.1:5061","name":"test_source"},"destination":{"type":"","target":"","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":"","tm_execute":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
		},
		{
			"flow_id null",
			&call.Call{
				ID:     uuid.FromStringOrNil("09b84a24-f3a9-11ea-80f6-d7e6af125065"),
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
			&rabbitmqhandler.Request{
				URI:      "/v1/calls",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "flow_id": "00000000-0000-0000-0000-000000000000","source": {"type": "sip","target": "test_source@127.0.0.1:5061","name": "test_source"},"destination": {"type": "sip","target": "test_destination@127.0.0.1:5061","name": "test_destination"}}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"09b84a24-f3a9-11ea-80f6-d7e6af125065","user_id":1,"asterisk_id":"","channel_id":"","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"sip","target":"test_source@127.0.0.1:5061","name":"test_source"},"destination":{"type":"sip","target":"test_destination@127.0.0.1:5061","name":"test_destination"},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":"","tm_execute":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockCall.EXPECT().CreateCallOutgoing(gomock.Any(), tt.call.UserID, tt.call.FlowID, tt.call.Source, tt.call.Destination).Return(tt.call, nil)
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

func TestProcessV1CallsIDDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
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
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"empty addresses",
			&call.Call{
				ID:          uuid.FromStringOrNil("91a0b50e-f4ec-11ea-b64c-1bf53742d0d8"),
				UserID:      1,
				Source:      call.Address{},
				Destination: call.Address{},
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/91a0b50e-f4ec-11ea-b64c-1bf53742d0d8",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
				Data:     nil,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"91a0b50e-f4ec-11ea-b64c-1bf53742d0d8","user_id":1,"asterisk_id":"","channel_id":"","flow_id":"00000000-0000-0000-0000-000000000000","conf_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":"","tm_execute":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
			mockCall.EXPECT().HangingUp(tt.call, ari.ChannelCauseNormalClearing)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)

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

func TestProcessV1CallsIDActionNextPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
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
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"empty addresses",
			&call.Call{
				ID:          uuid.FromStringOrNil("37b3a214-0afd-11eb-88ea-7bdd69288e90"),
				UserID:      1,
				Source:      call.Address{},
				Destination: call.Address{},
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/37b3a214-0afd-11eb-88ea-7bdd69288e90/action-next",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     nil,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
			mockCall.EXPECT().ActionNext(tt.call).Return(nil)

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

func TestProcessV1CallsIDChainedCallIDsPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
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
		name          string
		call          *call.Call
		chainedCallID uuid.UUID
		request       *rabbitmqhandler.Request
		expectRes     *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				ID:     uuid.FromStringOrNil("bfcdc03a-25bf-11eb-a9b2-bba80a81835b"),
				UserID: 1,
			},
			uuid.FromStringOrNil("76490d6a-25c0-11eb-970b-3bf9ae938f41"),
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/bfcdc03a-25bf-11eb-a9b2-bba80a81835b/chained-call-ids",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"chained_call_id": "76490d6a-25c0-11eb-970b-3bf9ae938f41"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockCall.EXPECT().ChainedCallIDAdd(tt.call.ID, tt.chainedCallID).Return(nil)

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

func TestProcessV1CallsIDChainedCallIDsDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
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
		name          string
		call          *call.Call
		chainedCallID uuid.UUID
		request       *rabbitmqhandler.Request
		expectRes     *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				ID:     uuid.FromStringOrNil("0eaa2942-25c4-11eb-90a3-63fb2b029bae"),
				UserID: 1,
			},
			uuid.FromStringOrNil("0ee268f2-25c4-11eb-917c-07eef32616dc"),
			&rabbitmqhandler.Request{
				URI:    "/v1/calls/0eaa2942-25c4-11eb-90a3-63fb2b029bae/chained-call-ids/0ee268f2-25c4-11eb-917c-07eef32616dc",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockCall.EXPECT().ChainedCallIDRemove(tt.call.ID, tt.chainedCallID).Return(nil)

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
