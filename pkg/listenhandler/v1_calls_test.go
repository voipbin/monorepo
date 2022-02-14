package listenhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
)

func TestProcessV1CallsIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockCall := callhandler.NewMockCallHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
		callHandler: mockCall,
	}

	tests := []struct {
		name      string
		request   *rabbitmqhandler.Request
		call      *call.Call
		expectRes *rabbitmqhandler.Response
	}{
		{
			"basic",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls/638769c2-620d-11eb-bd1f-6b576e26b4e6",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("638769c2-620d-11eb-bd1f-6b576e26b4e6"),
				CustomerID: uuid.FromStringOrNil("ab0fb69e-7f50-11ec-b0d3-2b4311e649e0"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"638769c2-620d-11eb-bd1f-6b576e26b4e6","customer_id":"ab0fb69e-7f50-11ec-b0d3-2b4311e649e0","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockCall.EXPECT().Get(gomock.Any(), tt.call.ID).Return(tt.call, nil)

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

func TestProcessV1CallsGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockCall := callhandler.NewMockCallHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
		callHandler: mockCall,
	}

	tests := []struct {
		name       string
		request    *rabbitmqhandler.Request
		customerID uuid.UUID
		pageSize   uint64
		pageToken  string
		calls      []*call.Call
		expectRes  *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls?page_size=10&page_token=2020-05-03%2021:35:02.809&customer_id=ac03d4ea-7f50-11ec-908d-d39407ab524d",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			uuid.FromStringOrNil("ac03d4ea-7f50-11ec-908d-d39407ab524d"),
			10,
			"2020-05-03 21:35:02.809",
			[]*call.Call{
				{
					ID:         uuid.FromStringOrNil("866ad964-620e-11eb-9f09-9fab48a7edd3"),
					CustomerID: uuid.FromStringOrNil("ac03d4ea-7f50-11ec-908d-d39407ab524d"),
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"866ad964-620e-11eb-9f09-9fab48a7edd3","customer_id":"ac03d4ea-7f50-11ec-908d-d39407ab524d","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}]`),
			},
		},
		{
			"2 items",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls?page_size=10&page_token=2020-05-03%2021:35:02.809&customer_id=ac35aeb6-7f50-11ec-b7c5-abac92baf1fb",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			uuid.FromStringOrNil("ac35aeb6-7f50-11ec-b7c5-abac92baf1fb"),
			10,
			"2020-05-03 21:35:02.809",
			[]*call.Call{
				{
					ID:         uuid.FromStringOrNil("866ad964-620e-11eb-9f09-9fab48a7edd3"),
					CustomerID: uuid.FromStringOrNil("ac35aeb6-7f50-11ec-b7c5-abac92baf1fb"),
				},
				{
					ID:         uuid.FromStringOrNil("095e2ec4-5f6c-11ec-b64c-efe2fb8efcbc"),
					CustomerID: uuid.FromStringOrNil("ac35aeb6-7f50-11ec-b7c5-abac92baf1fb"),
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"866ad964-620e-11eb-9f09-9fab48a7edd3","customer_id":"ac35aeb6-7f50-11ec-b7c5-abac92baf1fb","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""},{"id":"095e2ec4-5f6c-11ec-b64c-efe2fb8efcbc","customer_id":"ac35aeb6-7f50-11ec-b7c5-abac92baf1fb","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockCall.EXPECT().Gets(gomock.Any(), tt.customerID, tt.pageSize, tt.pageToken).Return(tt.calls, nil)
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
	mockCall := callhandler.NewMockCallHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
		callHandler: mockCall,
	}

	type test struct {
		name string

		id         uuid.UUID
		retryCount int
		delay      int

		request *rabbitmqhandler.Request
	}

	tests := []test{
		{
			"normal test",

			uuid.FromStringOrNil("1a94c1e6-982e-11ea-9298-43412daaf0da"),
			0,
			10,

			&rabbitmqhandler.Request{
				URI:    "/v1/calls/1a94c1e6-982e-11ea-9298-43412daaf0da/health-check",
				Method: rabbitmqhandler.RequestMethodPost,
				Data:   []byte(`{"retry_count": 0, "delay": 10}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockCall.EXPECT().CallHealthCheck(gomock.Any(), tt.id, tt.retryCount, tt.delay)

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
	mockCall := callhandler.NewMockCallHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
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

			mockCall.EXPECT().ActionTimeout(gomock.Any(), tt.id, tt.action).Return(nil)

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
	mockCall := callhandler.NewMockCallHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
		callHandler: mockCall,
	}

	type test struct {
		name string

		callID       uuid.UUID
		customerID   uuid.UUID
		flowID       uuid.UUID
		masterCallID uuid.UUID
		source       address.Address
		destination  address.Address

		call      *call.Call
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"empty addresses",

			uuid.FromStringOrNil("47a468d4-ed66-11ea-be25-97f0d867d634"),
			uuid.FromStringOrNil("ff0a0722-7f50-11ec-a839-4be463701c2f"),
			uuid.FromStringOrNil("59518eae-ed66-11ea-85ef-b77bdbc74ccc"),
			uuid.FromStringOrNil("11b1b1fa-8c93-11ec-9597-2320d5458176"),
			address.Address{},
			address.Address{},

			&call.Call{
				ID:          uuid.FromStringOrNil("47a468d4-ed66-11ea-be25-97f0d867d634"),
				CustomerID:  uuid.FromStringOrNil("ff0a0722-7f50-11ec-a839-4be463701c2f"),
				FlowID:      uuid.FromStringOrNil("59518eae-ed66-11ea-85ef-b77bdbc74ccc"),
				Source:      address.Address{},
				Destination: address.Address{},
			},

			&rabbitmqhandler.Request{
				URI:      "/v1/calls/47a468d4-ed66-11ea-be25-97f0d867d634",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "ff0a0722-7f50-11ec-a839-4be463701c2f", "flow_id": "59518eae-ed66-11ea-85ef-b77bdbc74ccc", "master_call_id": "11b1b1fa-8c93-11ec-9597-2320d5458176", "source": {}, "destination": {}}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"47a468d4-ed66-11ea-be25-97f0d867d634","customer_id":"ff0a0722-7f50-11ec-a839-4be463701c2f","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"59518eae-ed66-11ea-85ef-b77bdbc74ccc","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
		},
		{
			"source address",

			uuid.FromStringOrNil("47a468d4-ed66-11ea-be25-97f0d867d634"),
			uuid.FromStringOrNil("ffeda266-7f50-11ec-8089-df3388aef0cc"),
			uuid.FromStringOrNil("59518eae-ed66-11ea-85ef-b77bdbc74ccc"),
			uuid.Nil,
			address.Address{
				Type:   address.TypeSIP,
				Target: "test_source@127.0.0.1:5061",
				Name:   "test_source",
			},
			address.Address{},

			&call.Call{
				ID:         uuid.FromStringOrNil("47a468d4-ed66-11ea-be25-97f0d867d634"),
				CustomerID: uuid.FromStringOrNil("ffeda266-7f50-11ec-8089-df3388aef0cc"),
				FlowID:     uuid.FromStringOrNil("59518eae-ed66-11ea-85ef-b77bdbc74ccc"),
				Source: address.Address{
					Type:   address.TypeSIP,
					Target: "test_source@127.0.0.1:5061",
					Name:   "test_source",
				},
				Destination: address.Address{},
			},

			&rabbitmqhandler.Request{
				URI:      "/v1/calls/47a468d4-ed66-11ea-be25-97f0d867d634",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "ffeda266-7f50-11ec-8089-df3388aef0cc", "flow_id": "59518eae-ed66-11ea-85ef-b77bdbc74ccc", "source": {"type": "sip", "target": "test_source@127.0.0.1:5061", "name": "test_source"}, "destination": {}}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"47a468d4-ed66-11ea-be25-97f0d867d634","customer_id":"ffeda266-7f50-11ec-8089-df3388aef0cc","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"59518eae-ed66-11ea-85ef-b77bdbc74ccc","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"sip","target":"test_source@127.0.0.1:5061","target_name":"","name":"test_source","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
		},
		{
			"flow_id null",

			uuid.FromStringOrNil("f93eef0c-ed79-11ea-85cb-b39596cdf7ff"),
			uuid.FromStringOrNil("0017a4bc-7f51-11ec-8407-2f0fd8f346ef"),
			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
			uuid.Nil,
			address.Address{
				Type:   address.TypeSIP,
				Target: "test_source@127.0.0.1:5061",
				Name:   "test_source",
			},
			address.Address{
				Type:   address.TypeSIP,
				Target: "test_destination@127.0.0.1:5061",
				Name:   "test_destination",
			},

			&call.Call{
				ID:         uuid.FromStringOrNil("f93eef0c-ed79-11ea-85cb-b39596cdf7ff"),
				CustomerID: uuid.FromStringOrNil("0017a4bc-7f51-11ec-8407-2f0fd8f346ef"),
				FlowID:     uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
				Source: address.Address{
					Type:   address.TypeSIP,
					Target: "test_source@127.0.0.1:5061",
					Name:   "test_source",
				},
				Destination: address.Address{
					Type:   address.TypeSIP,
					Target: "test_destination@127.0.0.1:5061",
					Name:   "test_destination",
				},
			},

			&rabbitmqhandler.Request{
				URI:      "/v1/calls/f93eef0c-ed79-11ea-85cb-b39596cdf7ff",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "0017a4bc-7f51-11ec-8407-2f0fd8f346ef", "flow_id": "00000000-0000-0000-0000-000000000000","source": {"type": "sip","target": "test_source@127.0.0.1:5061","name": "test_source"},"destination": {"type": "sip","target": "test_destination@127.0.0.1:5061","name": "test_destination"}}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f93eef0c-ed79-11ea-85cb-b39596cdf7ff","customer_id":"0017a4bc-7f51-11ec-8407-2f0fd8f346ef","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"sip","target":"test_source@127.0.0.1:5061","target_name":"","name":"test_source","detail":""},"destination":{"type":"sip","target":"test_destination@127.0.0.1:5061","target_name":"","name":"test_destination","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockCall.EXPECT().CreateCallOutgoing(gomock.Any(), tt.callID, tt.customerID, tt.flowID, tt.masterCallID, tt.source, tt.destination).Return(tt.call, nil)
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
	mockCall := callhandler.NewMockCallHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
		callHandler: mockCall,
	}

	type test struct {
		name string

		customerID   uuid.UUID
		flowID       uuid.UUID
		masterCallID uuid.UUID
		source       address.Address
		destinations []address.Address

		responseCall []*call.Call
		request      *rabbitmqhandler.Request
		expectRes    *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"empty addresses",

			uuid.FromStringOrNil("34e72f78-7f51-11ec-a83b-cfc69cd4a641"),
			uuid.FromStringOrNil("78fd1276-f3a8-11ea-9734-6735e73fd720"),
			uuid.FromStringOrNil("a1c63272-8c91-11ec-8ee7-8b50458d3214"),
			address.Address{},
			[]address.Address{},

			[]*call.Call{
				{
					ID:          uuid.FromStringOrNil("72d56d08-f3a8-11ea-9c0c-ef8258d54f42"),
					CustomerID:  uuid.FromStringOrNil("34e72f78-7f51-11ec-a83b-cfc69cd4a641"),
					FlowID:      uuid.FromStringOrNil("78fd1276-f3a8-11ea-9734-6735e73fd720"),
					Source:      address.Address{},
					Destination: address.Address{},
				},
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/calls",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "34e72f78-7f51-11ec-a83b-cfc69cd4a641", "flow_id": "78fd1276-f3a8-11ea-9734-6735e73fd720", "master_call_id": "a1c63272-8c91-11ec-8ee7-8b50458d3214", "source": {}, "destinations": []}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"72d56d08-f3a8-11ea-9c0c-ef8258d54f42","customer_id":"34e72f78-7f51-11ec-a83b-cfc69cd4a641","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"78fd1276-f3a8-11ea-9734-6735e73fd720","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}]`),
			},
		},
		{
			"source address",

			uuid.FromStringOrNil("351014ec-7f51-11ec-9e7c-2b6427f906b7"),
			uuid.FromStringOrNil("d4df6ed6-f3a8-11ea-bf19-6f8063fdcfa1"),
			uuid.Nil,
			address.Address{
				Type:   address.TypeSIP,
				Target: "test_source@127.0.0.1:5061",
				Name:   "test_source",
			},
			[]address.Address{},

			[]*call.Call{
				{
					ID:         uuid.FromStringOrNil("cd561ba6-f3a8-11ea-b7ac-57b19fa28e09"),
					CustomerID: uuid.FromStringOrNil("351014ec-7f51-11ec-9e7c-2b6427f906b7"),
					FlowID:     uuid.FromStringOrNil("d4df6ed6-f3a8-11ea-bf19-6f8063fdcfa1"),
					Source: address.Address{
						Type:   address.TypeSIP,
						Target: "test_source@127.0.0.1:5061",
						Name:   "test_source",
					},
					Destination: address.Address{},
				},
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/calls",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "351014ec-7f51-11ec-9e7c-2b6427f906b7", "flow_id": "d4df6ed6-f3a8-11ea-bf19-6f8063fdcfa1", "source": {"type": "sip", "target": "test_source@127.0.0.1:5061", "name": "test_source"}, "destinations": []}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"cd561ba6-f3a8-11ea-b7ac-57b19fa28e09","customer_id":"351014ec-7f51-11ec-9e7c-2b6427f906b7","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"d4df6ed6-f3a8-11ea-bf19-6f8063fdcfa1","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"sip","target":"test_source@127.0.0.1:5061","target_name":"","name":"test_source","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}]`),
			},
		},
		{
			"flow_id null",

			uuid.FromStringOrNil("3534d6e2-7f51-11ec-8a74-d70202efb516"),
			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
			uuid.Nil,
			address.Address{
				Type:   address.TypeSIP,
				Target: "test_source@127.0.0.1:5061",
				Name:   "test_source",
			},
			[]address.Address{
				{
					Type:   address.TypeSIP,
					Target: "test_destination@127.0.0.1:5061",
					Name:   "test_destination",
				},
			},

			[]*call.Call{
				{
					ID:         uuid.FromStringOrNil("09b84a24-f3a9-11ea-80f6-d7e6af125065"),
					CustomerID: uuid.FromStringOrNil("3534d6e2-7f51-11ec-8a74-d70202efb516"),
					FlowID:     uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
					Source: address.Address{
						Type:   address.TypeSIP,
						Target: "test_source@127.0.0.1:5061",
						Name:   "test_source",
					},
					Destination: address.Address{
						Type:   address.TypeSIP,
						Target: "test_destination@127.0.0.1:5061",
						Name:   "test_destination",
					},
				},
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/calls",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "3534d6e2-7f51-11ec-8a74-d70202efb516", "flow_id": "00000000-0000-0000-0000-000000000000","source": {"type": "sip","target": "test_source@127.0.0.1:5061","name": "test_source"},"destinations": [{"type": "sip","target": "test_destination@127.0.0.1:5061","name": "test_destination"}]}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"09b84a24-f3a9-11ea-80f6-d7e6af125065","customer_id":"3534d6e2-7f51-11ec-8a74-d70202efb516","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"sip","target":"test_source@127.0.0.1:5061","target_name":"","name":"test_source","detail":""},"destination":{"type":"sip","target":"test_destination@127.0.0.1:5061","target_name":"","name":"test_destination","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockCall.EXPECT().CreateCallsOutgoing(gomock.Any(), tt.customerID, tt.flowID, tt.masterCallID, tt.source, tt.destinations).Return(tt.responseCall, nil)
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
	mockCall := callhandler.NewMockCallHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
		callHandler: mockCall,
	}

	type test struct {
		name      string
		id        uuid.UUID
		call      *call.Call
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"empty addresses",
			uuid.FromStringOrNil("91a0b50e-f4ec-11ea-b64c-1bf53742d0d8"),
			&call.Call{
				ID:          uuid.FromStringOrNil("91a0b50e-f4ec-11ea-b64c-1bf53742d0d8"),
				CustomerID:  uuid.FromStringOrNil("6ed4431a-7f51-11ec-8855-73041a5777e8"),
				Source:      address.Address{},
				Destination: address.Address{},
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
				Data:       []byte(`{"id":"91a0b50e-f4ec-11ea-b64c-1bf53742d0d8","customer_id":"6ed4431a-7f51-11ec-8855-73041a5777e8","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockCall.EXPECT().HangingUp(context.Background(), tt.id, ari.ChannelCauseNormalClearing)
			mockCall.EXPECT().Get(gomock.Any(), tt.id).Return(tt.call, nil)

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
	mockCall := callhandler.NewMockCallHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
		callHandler: mockCall,
	}

	tests := []struct {
		name      string
		call      *call.Call
		force     bool
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&call.Call{
				ID:          uuid.FromStringOrNil("37b3a214-0afd-11eb-88ea-7bdd69288e90"),
				CustomerID:  uuid.FromStringOrNil("6efe9d54-7f51-11ec-95dc-4f25e8777704"),
				Source:      address.Address{},
				Destination: address.Address{},
			},
			false,
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/37b3a214-0afd-11eb-88ea-7bdd69288e90/action-next",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"force":false}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
		{
			"next force",
			&call.Call{
				ID:          uuid.FromStringOrNil("37b3a214-0afd-11eb-88ea-7bdd69288e90"),
				CustomerID:  uuid.FromStringOrNil("6f29a850-7f51-11ec-a44d-0714d04b7ff2"),
				Source:      address.Address{},
				Destination: address.Address{},
			},
			true,
			&rabbitmqhandler.Request{
				URI:       "/v1/calls/37b3a214-0afd-11eb-88ea-7bdd69288e90/action-next",
				Method:    rabbitmqhandler.RequestMethodPost,
				Publisher: "queue-manager",
				DataType:  "application/json",
				Data:      []byte(`{"force":true}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockCall.EXPECT().Get(gomock.Any(), tt.call.ID).Return(tt.call, nil)

			if !tt.force {
				mockCall.EXPECT().ActionNext(context.Background(), tt.call)
			} else {
				mockCall.EXPECT().ActionNextForce(context.Background(), tt.call)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			time.Sleep(100 * time.Millisecond)

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
	mockCall := callhandler.NewMockCallHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
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
				ID:         uuid.FromStringOrNil("bfcdc03a-25bf-11eb-a9b2-bba80a81835b"),
				CustomerID: uuid.FromStringOrNil("86711e1c-7f51-11ec-a807-0385a20de8ac"),
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

			mockCall.EXPECT().ChainedCallIDAdd(context.Background(), tt.call.ID, tt.chainedCallID).Return(nil)

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
	mockCall := callhandler.NewMockCallHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
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
				ID: uuid.FromStringOrNil("0eaa2942-25c4-11eb-90a3-63fb2b029bae"),
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

			mockCall.EXPECT().ChainedCallIDRemove(context.Background(), tt.call.ID, tt.chainedCallID).Return(nil)

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

func TestProcessV1CallsIDExternalMediaPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockCall := callhandler.NewMockCallHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
		callHandler: mockCall,
	}

	type test struct {
		name    string
		call    *call.Call
		request *rabbitmqhandler.Request

		expectExternalHost   string
		expectEncapsulation  string
		expectTransport      string
		expectConnectionType string
		expectFormat         string
		expectDirection      string

		extCh *channel.Channel

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				ID: uuid.FromStringOrNil("31255b7c-0a6b-11ec-87e2-afe5a545df76"),
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/31255b7c-0a6b-11ec-87e2-afe5a545df76/external-media",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"external_host": "127.0.0.1:5060", "encapsulation": "rtp", "transport": "udp", "connection_type": "client", "format": "ulaw", "direction": "both", "data": ""}`),
			},

			"127.0.0.1:5060",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",

			&channel.Channel{
				ID: "d9f3fc36-0a7a-11ec-8f20-eb8a7aa17176",

				Data: map[string]interface{}{
					callhandler.ChannelValiableExternalMediaLocalAddress: "127.0.0.1",
					callhandler.ChannelValiableExternalMediaLocalPort:    "9000",
				},
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"media_addr_ip":"127.0.0.1","media_addr_port":9000}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockCall.EXPECT().ExternalMediaStart(
				context.Background(),
				tt.call.ID,
				false,
				tt.expectExternalHost,
				tt.expectEncapsulation,
				tt.expectTransport,
				tt.expectConnectionType,
				tt.expectFormat,
				tt.expectDirection,
			).Return(tt.extCh, nil)

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
