package listenhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/externalmediahandler"
)

func Test_processV1CallsIDGet(t *testing.T) {

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
				Data:       []byte(`{"id":"638769c2-620d-11eb-bd1f-6b576e26b4e6","customer_id":"ab0fb69e-7f50-11ec-b0d3-2b4311e649e0","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				rabbitSock:  mockSock,
				callHandler: mockCall,
			}

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

func Test_processV1CallsGet(t *testing.T) {
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
				Data:       []byte(`[{"id":"866ad964-620e-11eb-9f09-9fab48a7edd3","customer_id":"ac03d4ea-7f50-11ec-908d-d39407ab524d","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}]`),
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
				Data:       []byte(`[{"id":"866ad964-620e-11eb-9f09-9fab48a7edd3","customer_id":"ac35aeb6-7f50-11ec-b7c5-abac92baf1fb","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"095e2ec4-5f6c-11ec-b64c-efe2fb8efcbc","customer_id":"ac35aeb6-7f50-11ec-b7c5-abac92baf1fb","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				rabbitSock:  mockSock,
				callHandler: mockCall,
			}

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
		action    *fmaction.Action
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
			&fmaction.Action{
				ID:        uuid.FromStringOrNil("ec4c8192-994b-11ea-ab64-9b63b984b7c4"),
				Type:      fmaction.TypeEcho,
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

func Test_processV1CallsIDPost(t *testing.T) {
	tests := []struct {
		name string

		callID       uuid.UUID
		customerID   uuid.UUID
		flowID       uuid.UUID
		activeflowID uuid.UUID
		masterCallID uuid.UUID

		source      commonaddress.Address
		destination commonaddress.Address

		call      *call.Call
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
	}{
		{
			"empty addresses",

			uuid.FromStringOrNil("47a468d4-ed66-11ea-be25-97f0d867d634"),
			uuid.FromStringOrNil("ff0a0722-7f50-11ec-a839-4be463701c2f"),
			uuid.FromStringOrNil("59518eae-ed66-11ea-85ef-b77bdbc74ccc"),
			uuid.FromStringOrNil("bf88a888-ddab-435b-8ae1-1eb8a3072230"),
			uuid.FromStringOrNil("11b1b1fa-8c93-11ec-9597-2320d5458176"),

			commonaddress.Address{},
			commonaddress.Address{},

			&call.Call{
				ID:          uuid.FromStringOrNil("47a468d4-ed66-11ea-be25-97f0d867d634"),
				CustomerID:  uuid.FromStringOrNil("ff0a0722-7f50-11ec-a839-4be463701c2f"),
				FlowID:      uuid.FromStringOrNil("59518eae-ed66-11ea-85ef-b77bdbc74ccc"),
				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},
			},

			&rabbitmqhandler.Request{
				URI:      "/v1/calls/47a468d4-ed66-11ea-be25-97f0d867d634",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "ff0a0722-7f50-11ec-a839-4be463701c2f", "flow_id": "59518eae-ed66-11ea-85ef-b77bdbc74ccc", "activeflow_id": "bf88a888-ddab-435b-8ae1-1eb8a3072230", "master_call_id": "11b1b1fa-8c93-11ec-9597-2320d5458176", "source": {}, "destination": {}}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"47a468d4-ed66-11ea-be25-97f0d867d634","customer_id":"ff0a0722-7f50-11ec-a839-4be463701c2f","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"59518eae-ed66-11ea-85ef-b77bdbc74ccc","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"source address",

			uuid.FromStringOrNil("47a468d4-ed66-11ea-be25-97f0d867d634"),
			uuid.FromStringOrNil("ffeda266-7f50-11ec-8089-df3388aef0cc"),
			uuid.FromStringOrNil("59518eae-ed66-11ea-85ef-b77bdbc74ccc"),
			uuid.FromStringOrNil("2e9f9862-9803-47f0-8f40-66f1522ef7f3"),
			uuid.Nil,

			commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "test_source@127.0.0.1:5061",
				Name:   "test_source",
			},
			commonaddress.Address{},

			&call.Call{
				ID:         uuid.FromStringOrNil("47a468d4-ed66-11ea-be25-97f0d867d634"),
				CustomerID: uuid.FromStringOrNil("ffeda266-7f50-11ec-8089-df3388aef0cc"),
				FlowID:     uuid.FromStringOrNil("59518eae-ed66-11ea-85ef-b77bdbc74ccc"),
				Source: commonaddress.Address{
					Type:   commonaddress.TypeSIP,
					Target: "test_source@127.0.0.1:5061",
					Name:   "test_source",
				},
				Destination: commonaddress.Address{},
			},

			&rabbitmqhandler.Request{
				URI:      "/v1/calls/47a468d4-ed66-11ea-be25-97f0d867d634",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "ffeda266-7f50-11ec-8089-df3388aef0cc", "flow_id": "59518eae-ed66-11ea-85ef-b77bdbc74ccc", "activeflow_id": "2e9f9862-9803-47f0-8f40-66f1522ef7f3", "source": {"type": "sip", "target": "test_source@127.0.0.1:5061", "name": "test_source"}, "destination": {}}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"47a468d4-ed66-11ea-be25-97f0d867d634","customer_id":"ffeda266-7f50-11ec-8089-df3388aef0cc","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"59518eae-ed66-11ea-85ef-b77bdbc74ccc","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"sip","target":"test_source@127.0.0.1:5061","target_name":"","name":"test_source","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"flow_id null",

			uuid.FromStringOrNil("f93eef0c-ed79-11ea-85cb-b39596cdf7ff"),
			uuid.FromStringOrNil("0017a4bc-7f51-11ec-8407-2f0fd8f346ef"),
			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
			uuid.FromStringOrNil("5516fd59-00f2-415d-bcd5-7f0d0b798bfc"),
			uuid.Nil,

			commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "test_source@127.0.0.1:5061",
				Name:   "test_source",
			},
			commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "test_destination@127.0.0.1:5061",
				Name:   "test_destination",
			},

			&call.Call{
				ID:         uuid.FromStringOrNil("f93eef0c-ed79-11ea-85cb-b39596cdf7ff"),
				CustomerID: uuid.FromStringOrNil("0017a4bc-7f51-11ec-8407-2f0fd8f346ef"),
				FlowID:     uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
				Source: commonaddress.Address{
					Type:   commonaddress.TypeSIP,
					Target: "test_source@127.0.0.1:5061",
					Name:   "test_source",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeSIP,
					Target: "test_destination@127.0.0.1:5061",
					Name:   "test_destination",
				},
			},

			&rabbitmqhandler.Request{
				URI:      "/v1/calls/f93eef0c-ed79-11ea-85cb-b39596cdf7ff",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "0017a4bc-7f51-11ec-8407-2f0fd8f346ef", "flow_id": "00000000-0000-0000-0000-000000000000", "activeflow_id": "5516fd59-00f2-415d-bcd5-7f0d0b798bfc", "source": {"type": "sip","target": "test_source@127.0.0.1:5061","name": "test_source"},"destination": {"type": "sip","target": "test_destination@127.0.0.1:5061","name": "test_destination"}}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f93eef0c-ed79-11ea-85cb-b39596cdf7ff","customer_id":"0017a4bc-7f51-11ec-8407-2f0fd8f346ef","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"sip","target":"test_source@127.0.0.1:5061","target_name":"","name":"test_source","detail":""},"destination":{"type":"sip","target":"test_destination@127.0.0.1:5061","target_name":"","name":"test_destination","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				rabbitSock:  mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().CreateCallOutgoing(gomock.Any(), tt.callID, tt.customerID, tt.flowID, tt.activeflowID, tt.masterCallID, tt.source, tt.destination).Return(tt.call, nil)
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

func Test_processV1CallsPost(t *testing.T) {

	type test struct {
		name string

		customerID   uuid.UUID
		flowID       uuid.UUID
		masterCallID uuid.UUID
		source       commonaddress.Address
		destinations []commonaddress.Address

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
			commonaddress.Address{},
			[]commonaddress.Address{},

			[]*call.Call{
				{
					ID:          uuid.FromStringOrNil("72d56d08-f3a8-11ea-9c0c-ef8258d54f42"),
					CustomerID:  uuid.FromStringOrNil("34e72f78-7f51-11ec-a83b-cfc69cd4a641"),
					FlowID:      uuid.FromStringOrNil("78fd1276-f3a8-11ea-9734-6735e73fd720"),
					Source:      commonaddress.Address{},
					Destination: commonaddress.Address{},
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
				Data:       []byte(`[{"id":"72d56d08-f3a8-11ea-9c0c-ef8258d54f42","customer_id":"34e72f78-7f51-11ec-a83b-cfc69cd4a641","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"78fd1276-f3a8-11ea-9734-6735e73fd720","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"source address",

			uuid.FromStringOrNil("351014ec-7f51-11ec-9e7c-2b6427f906b7"),
			uuid.FromStringOrNil("d4df6ed6-f3a8-11ea-bf19-6f8063fdcfa1"),
			uuid.Nil,
			commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "test_source@127.0.0.1:5061",
				Name:   "test_source",
			},
			[]commonaddress.Address{},

			[]*call.Call{
				{
					ID:         uuid.FromStringOrNil("cd561ba6-f3a8-11ea-b7ac-57b19fa28e09"),
					CustomerID: uuid.FromStringOrNil("351014ec-7f51-11ec-9e7c-2b6427f906b7"),
					FlowID:     uuid.FromStringOrNil("d4df6ed6-f3a8-11ea-bf19-6f8063fdcfa1"),
					Source: commonaddress.Address{
						Type:   commonaddress.TypeSIP,
						Target: "test_source@127.0.0.1:5061",
						Name:   "test_source",
					},
					Destination: commonaddress.Address{},
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
				Data:       []byte(`[{"id":"cd561ba6-f3a8-11ea-b7ac-57b19fa28e09","customer_id":"351014ec-7f51-11ec-9e7c-2b6427f906b7","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"d4df6ed6-f3a8-11ea-bf19-6f8063fdcfa1","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"sip","target":"test_source@127.0.0.1:5061","target_name":"","name":"test_source","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"flow_id null",

			uuid.FromStringOrNil("3534d6e2-7f51-11ec-8a74-d70202efb516"),
			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
			uuid.Nil,
			commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "test_source@127.0.0.1:5061",
				Name:   "test_source",
			},
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeSIP,
					Target: "test_destination@127.0.0.1:5061",
					Name:   "test_destination",
				},
			},

			[]*call.Call{
				{
					ID:         uuid.FromStringOrNil("09b84a24-f3a9-11ea-80f6-d7e6af125065"),
					CustomerID: uuid.FromStringOrNil("3534d6e2-7f51-11ec-8a74-d70202efb516"),
					FlowID:     uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
					Source: commonaddress.Address{
						Type:   commonaddress.TypeSIP,
						Target: "test_source@127.0.0.1:5061",
						Name:   "test_source",
					},
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeSIP,
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
				Data:       []byte(`[{"id":"09b84a24-f3a9-11ea-80f6-d7e6af125065","customer_id":"3534d6e2-7f51-11ec-8a74-d70202efb516","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"sip","target":"test_source@127.0.0.1:5061","target_name":"","name":"test_source","detail":""},"destination":{"type":"sip","target":"test_destination@127.0.0.1:5061","target_name":"","name":"test_destination","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				rabbitSock:  mockSock,
				callHandler: mockCall,
			}

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

func Test_processV1CallsIDDelete(t *testing.T) {

	type test struct {
		name    string
		id      uuid.UUID
		request *rabbitmqhandler.Request

		responseCall *call.Call
		expectRes    *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("8e49788e-fc9e-41a0-a73c-a24c030848ca"),
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/8e49788e-fc9e-41a0-a73c-a24c030848ca",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
				Data:     nil,
			},

			&call.Call{
				ID:          uuid.FromStringOrNil("8e49788e-fc9e-41a0-a73c-a24c030848ca"),
				CustomerID:  uuid.FromStringOrNil("6ed4431a-7f51-11ec-8855-73041a5777e8"),
				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8e49788e-fc9e-41a0-a73c-a24c030848ca","customer_id":"6ed4431a-7f51-11ec-8855-73041a5777e8","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				rabbitSock:  mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().Delete(context.Background(), tt.id).Return(tt.responseCall, nil)

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

func Test_processV1CallsIDHangupPost(t *testing.T) {

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
				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/91a0b50e-f4ec-11ea-b64c-1bf53742d0d8/hangup",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     nil,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"91a0b50e-f4ec-11ea-b64c-1bf53742d0d8","customer_id":"6ed4431a-7f51-11ec-8855-73041a5777e8","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				rabbitSock:  mockSock,
				callHandler: mockCall,
			}

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
				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},
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
				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},
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

func Test_processV1CallsIDChainedCallIDsPost(t *testing.T) {

	type test struct {
		name          string
		call          *call.Call
		chainedCallID uuid.UUID
		request       *rabbitmqhandler.Request

		responseCall *call.Call
		expectRes    *rabbitmqhandler.Response
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

			&call.Call{
				ID: uuid.FromStringOrNil("bfcdc03a-25bf-11eb-a9b2-bba80a81835b"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bfcdc03a-25bf-11eb-a9b2-bba80a81835b","customer_id":"00000000-0000-0000-0000-000000000000","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				rabbitSock:  mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().ChainedCallIDAdd(context.Background(), tt.call.ID, tt.chainedCallID).Return(tt.responseCall, nil)

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

func Test_processV1CallsIDChainedCallIDsDelete(t *testing.T) {

	type test struct {
		name          string
		call          *call.Call
		chainedCallID uuid.UUID
		request       *rabbitmqhandler.Request

		responseCall *call.Call
		expectRes    *rabbitmqhandler.Response
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

			&call.Call{
				ID: uuid.FromStringOrNil("0eaa2942-25c4-11eb-90a3-63fb2b029bae"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0eaa2942-25c4-11eb-90a3-63fb2b029bae","customer_id":"00000000-0000-0000-0000-000000000000","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				rabbitSock:  mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().ChainedCallIDRemove(context.Background(), tt.call.ID, tt.chainedCallID).Return(tt.responseCall, nil)

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

func Test_processV1CallsIDExternalMediaPost(t *testing.T) {

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		responseCall *call.Call

		expectCallID         uuid.UUID
		expectExternalHost   string
		expectEncapsulation  string
		expectTransport      string
		expectConnectionType string
		expectFormat         string
		expectDirection      string

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",

			&rabbitmqhandler.Request{
				URI:      "/v1/calls/31255b7c-0a6b-11ec-87e2-afe5a545df76/external-media",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"external_host": "127.0.0.1:5060", "encapsulation": "rtp", "transport": "udp", "connection_type": "client", "format": "ulaw", "direction": "both", "data": ""}`),
			},

			&call.Call{
				ID: uuid.FromStringOrNil("31255b7c-0a6b-11ec-87e2-afe5a545df76"),
			},

			uuid.FromStringOrNil("31255b7c-0a6b-11ec-87e2-afe5a545df76"),
			"127.0.0.1:5060",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"31255b7c-0a6b-11ec-87e2-afe5a545df76","customer_id":"00000000-0000-0000-0000-000000000000","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &listenHandler{
				rabbitSock:           mockSock,
				callHandler:          mockCall,
				externalMediaHandler: mockExternal,
			}

			mockCall.EXPECT().ExternalMediaStart(
				context.Background(),
				tt.expectCallID,
				tt.expectExternalHost,
				tt.expectEncapsulation,
				tt.expectTransport,
				tt.expectConnectionType,
				tt.expectFormat,
				tt.expectDirection,
			).Return(tt.responseCall, nil)

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

func Test_processV1CallsIDExternalMediaDelete(t *testing.T) {

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		responseCall *call.Call

		expectCallID uuid.UUID
		expectRes    *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls/7fcac8d6-9730-11ed-bd80-a764f6bc382e/external-media",
				Method: rabbitmqhandler.RequestMethodDelete,
			},

			&call.Call{
				ID: uuid.FromStringOrNil("7fcac8d6-9730-11ed-bd80-a764f6bc382e"),
			},

			uuid.FromStringOrNil("7fcac8d6-9730-11ed-bd80-a764f6bc382e"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7fcac8d6-9730-11ed-bd80-a764f6bc382e","customer_id":"00000000-0000-0000-0000-000000000000","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &listenHandler{
				rabbitSock:           mockSock,
				callHandler:          mockCall,
				externalMediaHandler: mockExternal,
			}

			mockCall.EXPECT().ExternalMediaStop(context.Background(), tt.expectCallID).Return(tt.responseCall, nil)

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

func Test_processV1CallsIDDigitsGet(t *testing.T) {

	type test struct {
		name string

		request *rabbitmqhandler.Request

		id             uuid.UUID
		responseDigits string

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls/669e567e-9016-11ec-9190-07c8a63f44a8/digits",
				Method: rabbitmqhandler.RequestMethodGet,
			},

			uuid.FromStringOrNil("669e567e-9016-11ec-9190-07c8a63f44a8"),
			"1",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"digits":"1"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				rabbitSock:  mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().DigitsGet(gomock.Any(), tt.id).Return(tt.responseDigits, nil)

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

func Test_processV1CallsIDDigitsPost(t *testing.T) {

	type test struct {
		name string

		request *rabbitmqhandler.Request

		id     uuid.UUID
		digits string

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/a5ca555a-9912-11ec-ab1a-2b341f06e3c0/digits",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"digits": "123"}`),
			},

			uuid.FromStringOrNil("a5ca555a-9912-11ec-ab1a-2b341f06e3c0"),
			"123",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
		{
			"set to empty",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/a5ca555a-9912-11ec-ab1a-2b341f06e3c0/digits",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"digits": ""}`),
			},

			uuid.FromStringOrNil("a5ca555a-9912-11ec-ab1a-2b341f06e3c0"),
			"",

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
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				rabbitSock:  mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().DigitsSet(gomock.Any(), tt.id, tt.digits).Return(nil)

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

func Test_processV1CallsIDRecordingIDPut(t *testing.T) {

	type test struct {
		name string

		request *rabbitmqhandler.Request

		id          uuid.UUID
		recordingID uuid.UUID

		responseCall *call.Call

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/0953474c-8fd2-11ed-a24a-7bf36392fef2/recording_id",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"recording_id": "09178464-8fd2-11ed-a1d7-5b5491e83cc7"}`),
			},

			uuid.FromStringOrNil("0953474c-8fd2-11ed-a24a-7bf36392fef2"),
			uuid.FromStringOrNil("09178464-8fd2-11ed-a1d7-5b5491e83cc7"),

			&call.Call{
				ID: uuid.FromStringOrNil("0953474c-8fd2-11ed-a24a-7bf36392fef2"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0953474c-8fd2-11ed-a24a-7bf36392fef2","customer_id":"00000000-0000-0000-0000-000000000000","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"set to empty",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/0977e50c-8fd2-11ed-8684-e34a2113a8cc/recording_id",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"recording_id": "00000000-0000-0000-0000-000000000000"}`),
			},

			uuid.FromStringOrNil("0977e50c-8fd2-11ed-8684-e34a2113a8cc"),
			uuid.Nil,

			&call.Call{
				ID: uuid.FromStringOrNil("0977e50c-8fd2-11ed-8684-e34a2113a8cc"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0977e50c-8fd2-11ed-8684-e34a2113a8cc","customer_id":"00000000-0000-0000-0000-000000000000","asterisk_id":"","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				rabbitSock:  mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().UpdateRecordingID(gomock.Any(), tt.id, tt.recordingID).Return(tt.responseCall, nil)

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

func Test_processV1CallsIDRecordingStartPost(t *testing.T) {

	type test struct {
		name string

		request *rabbitmqhandler.Request

		expectID           uuid.UUID
		expectFormat       recording.Format
		expectEndOfSilence int
		expectEndOfKey     string
		expectDuration     int

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/1c3dc786-9344-11ed-96a2-17c902204823/recording_start",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"format": "wav", "end_of_silence": 1000, "end_of_key": "#", "duration": 86400}`),
			},

			uuid.FromStringOrNil("1c3dc786-9344-11ed-96a2-17c902204823"),
			recording.FormatWAV,
			1000,
			"#",
			86400,

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
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				rabbitSock:  mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().RecordingStart(gomock.Any(), tt.expectID, tt.expectFormat, tt.expectEndOfSilence, tt.expectEndOfKey, tt.expectDuration).Return(nil)
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

func Test_processV1CallsIDRecordingStopPost(t *testing.T) {

	type test struct {
		name string

		request *rabbitmqhandler.Request

		expectID  uuid.UUID
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/1c73262e-9344-11ed-840d-37569c93274f/recording_stop",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("1c73262e-9344-11ed-840d-37569c93274f"),

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
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				rabbitSock:  mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().RecordingStop(gomock.Any(), tt.expectID).Return(nil)
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
