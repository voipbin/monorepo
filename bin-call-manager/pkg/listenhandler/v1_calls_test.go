package listenhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
)

func Test_processV1CallsIDGet(t *testing.T) {

	tests := []struct {
		name      string
		request   *sock.Request
		call      *call.Call
		expectRes *sock.Response
	}{
		{
			"basic",
			&sock.Request{
				URI:    "/v1/calls/638769c2-620d-11eb-bd1f-6b576e26b4e6",
				Method: sock.RequestMethodGet,
			},
			&call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("638769c2-620d-11eb-bd1f-6b576e26b4e6"),
					CustomerID: uuid.FromStringOrNil("ab0fb69e-7f50-11ec-b0d3-2b4311e649e0"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"638769c2-620d-11eb-bd1f-6b576e26b4e6","customer_id":"ab0fb69e-7f50-11ec-b0d3-2b4311e649e0","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
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
		name string

		request   *sock.Request
		pageSize  uint64
		pageToken string

		responseCalls   []*call.Call
		responseFilters map[string]string
		expectRes       *sock.Response
	}{
		{
			"normal",

			&sock.Request{
				URI:    "/v1/calls?page_size=10&page_token=2020-05-03%2021:35:02.809&customer_id=ac03d4ea-7f50-11ec-908d-d39407ab524d&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},
			10,
			"2020-05-03 21:35:02.809",

			[]*call.Call{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("866ad964-620e-11eb-9f09-9fab48a7edd3"),
						CustomerID: uuid.FromStringOrNil("ac03d4ea-7f50-11ec-908d-d39407ab524d"),
					},
				},
			},
			map[string]string{
				"customer_id": "ac03d4ea-7f50-11ec-908d-d39407ab524d",
				"deleted":     "false",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"866ad964-620e-11eb-9f09-9fab48a7edd3","customer_id":"ac03d4ea-7f50-11ec-908d-d39407ab524d","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"2 items",

			&sock.Request{
				URI:    "/v1/calls?page_size=10&page_token=2020-05-03%2021:35:02.809&customer_id=ac35aeb6-7f50-11ec-b7c5-abac92baf1fb&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},
			10,
			"2020-05-03 21:35:02.809",

			[]*call.Call{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("866ad964-620e-11eb-9f09-9fab48a7edd3"),
						CustomerID: uuid.FromStringOrNil("ac35aeb6-7f50-11ec-b7c5-abac92baf1fb"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("095e2ec4-5f6c-11ec-b64c-efe2fb8efcbc"),
						CustomerID: uuid.FromStringOrNil("ac35aeb6-7f50-11ec-b7c5-abac92baf1fb"),
					},
				},
			},
			map[string]string{
				"customer_id": "ac35aeb6-7f50-11ec-b7c5-abac92baf1fb",
				"deleted":     "false",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"866ad964-620e-11eb-9f09-9fab48a7edd3","customer_id":"ac35aeb6-7f50-11ec-b7c5-abac92baf1fb","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"095e2ec4-5f6c-11ec-b64c-efe2fb8efcbc","customer_id":"ac35aeb6-7f50-11ec-b7c5-abac92baf1fb","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				utilHandler: mockUtil,
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockCall.EXPECT().Gets(gomock.Any(), tt.pageSize, tt.pageToken, tt.responseFilters).Return(tt.responseCalls, nil)
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

func Test_processV1CallsIDHealthPost(t *testing.T) {

	type test struct {
		name string

		id         uuid.UUID
		retryCount int

		request *sock.Request
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("1a94c1e6-982e-11ea-9298-43412daaf0da"),
			0,

			&sock.Request{
				URI:    "/v1/calls/1a94c1e6-982e-11ea-9298-43412daaf0da/health-check",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"retry_count": 0}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().HealthCheck(gomock.Any(), tt.id, tt.retryCount)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			} else if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}

func TestProcessV1CallsIDActionTimeoutPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)

	h := &listenHandler{
		sockHandler: mockSock,
		callHandler: mockCall,
	}

	type test struct {
		name      string
		id        uuid.UUID
		request   *sock.Request
		action    *fmaction.Action
		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal test",
			uuid.FromStringOrNil("1a94c1e6-982e-11ea-9298-43412daaf0da"),
			&sock.Request{
				URI:      "/v1/calls/1a94c1e6-982e-11ea-9298-43412daaf0da/action-timeout",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"action_id": "ec4c8192-994b-11ea-ab64-9b63b984b7c4", "action_type": "echo", "tm_execute": "2020-05-03T21:35:02.809"}`),
			},
			&fmaction.Action{
				ID:        uuid.FromStringOrNil("ec4c8192-994b-11ea-ab64-9b63b984b7c4"),
				Type:      fmaction.TypeEcho,
				TMExecute: "2020-05-03T21:35:02.809",
			},
			&sock.Response{
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

		request *sock.Request

		callID       uuid.UUID
		customerID   uuid.UUID
		flowID       uuid.UUID
		activeflowID uuid.UUID
		masterCallID uuid.UUID

		source         commonaddress.Address
		destination    commonaddress.Address
		groupcallID    uuid.UUID
		earlyExecution bool
		connect        bool

		call      *call.Call
		expectRes *sock.Response
	}{
		{
			name: "empty",

			request: &sock.Request{
				URI:      "/v1/calls/47a468d4-ed66-11ea-be25-97f0d867d634",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "ff0a0722-7f50-11ec-a839-4be463701c2f", "flow_id": "59518eae-ed66-11ea-85ef-b77bdbc74ccc", "activeflow_id": "bf88a888-ddab-435b-8ae1-1eb8a3072230", "master_call_id": "11b1b1fa-8c93-11ec-9597-2320d5458176"}`),
			},

			callID:       uuid.FromStringOrNil("47a468d4-ed66-11ea-be25-97f0d867d634"),
			customerID:   uuid.FromStringOrNil("ff0a0722-7f50-11ec-a839-4be463701c2f"),
			flowID:       uuid.FromStringOrNil("59518eae-ed66-11ea-85ef-b77bdbc74ccc"),
			activeflowID: uuid.FromStringOrNil("bf88a888-ddab-435b-8ae1-1eb8a3072230"),
			masterCallID: uuid.FromStringOrNil("11b1b1fa-8c93-11ec-9597-2320d5458176"),

			call: &call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("47a468d4-ed66-11ea-be25-97f0d867d634"),
					CustomerID: uuid.FromStringOrNil("ff0a0722-7f50-11ec-a839-4be463701c2f"),
				},
				FlowID:      uuid.FromStringOrNil("59518eae-ed66-11ea-85ef-b77bdbc74ccc"),
				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},
			},

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"47a468d4-ed66-11ea-be25-97f0d867d634","customer_id":"ff0a0722-7f50-11ec-a839-4be463701c2f","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"59518eae-ed66-11ea-85ef-b77bdbc74ccc","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			name: "have all",

			request: &sock.Request{
				URI:      "/v1/calls/47a468d4-ed66-11ea-be25-97f0d867d634",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "ffeda266-7f50-11ec-8089-df3388aef0cc", "flow_id": "59518eae-ed66-11ea-85ef-b77bdbc74ccc", "activeflow_id": "2e9f9862-9803-47f0-8f40-66f1522ef7f3", "source": {"type": "sip", "target": "test_source@127.0.0.1:5061", "name": "test_source"}, "destination": {"type":"tel","target":"+821100000001"}, "groupcall_id":"266c6cce-bae2-11ed-afd7-ebef79165c1f","early_execution": true, "connect": true}`),
			},

			callID:       uuid.FromStringOrNil("47a468d4-ed66-11ea-be25-97f0d867d634"),
			customerID:   uuid.FromStringOrNil("ffeda266-7f50-11ec-8089-df3388aef0cc"),
			flowID:       uuid.FromStringOrNil("59518eae-ed66-11ea-85ef-b77bdbc74ccc"),
			activeflowID: uuid.FromStringOrNil("2e9f9862-9803-47f0-8f40-66f1522ef7f3"),
			masterCallID: uuid.Nil,

			source: commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "test_source@127.0.0.1:5061",
				Name:   "test_source",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			groupcallID:    uuid.FromStringOrNil("266c6cce-bae2-11ed-afd7-ebef79165c1f"),
			earlyExecution: true,
			connect:        true,

			call: &call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("47a468d4-ed66-11ea-be25-97f0d867d634"),
					CustomerID: uuid.FromStringOrNil("ffeda266-7f50-11ec-8089-df3388aef0cc"),
				},
				FlowID: uuid.FromStringOrNil("59518eae-ed66-11ea-85ef-b77bdbc74ccc"),
				Source: commonaddress.Address{
					Type:   commonaddress.TypeSIP,
					Target: "test_source@127.0.0.1:5061",
					Name:   "test_source",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
			},

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"47a468d4-ed66-11ea-be25-97f0d867d634","customer_id":"ffeda266-7f50-11ec-8089-df3388aef0cc","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"59518eae-ed66-11ea-85ef-b77bdbc74ccc","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"sip","target":"test_source@127.0.0.1:5061","target_name":"","name":"test_source","detail":""},"destination":{"type":"tel","target":"+821100000001","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().CreateCallOutgoing(gomock.Any(), tt.callID, tt.customerID, tt.flowID, tt.activeflowID, tt.masterCallID, tt.groupcallID, tt.source, tt.destination, tt.earlyExecution, tt.connect).Return(tt.call, nil)
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

		customerID     uuid.UUID
		flowID         uuid.UUID
		masterCallID   uuid.UUID
		source         commonaddress.Address
		destinations   []commonaddress.Address
		earlyExeuction bool
		connect        bool

		responseCalls      []*call.Call
		responseGroupcalls []*groupcall.Groupcall
		request            *sock.Request
		expectRes          *sock.Response
	}

	tests := []test{
		{
			name: "have all",

			customerID:   uuid.FromStringOrNil("351014ec-7f51-11ec-9e7c-2b6427f906b7"),
			flowID:       uuid.FromStringOrNil("d4df6ed6-f3a8-11ea-bf19-6f8063fdcfa1"),
			masterCallID: uuid.Nil,
			source: commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "test_source@127.0.0.1:5061",
				Name:   "test_source",
			},
			destinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
			},
			earlyExeuction: true,
			connect:        true,

			responseCalls: []*call.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("cd561ba6-f3a8-11ea-b7ac-57b19fa28e09"),
					},
				},
			},
			responseGroupcalls: []*groupcall.Groupcall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("4f1e2d54-7d52-4c1f-ad49-5b51feea0055"),
					},
				},
			},

			request: &sock.Request{
				URI:      "/v1/calls",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "351014ec-7f51-11ec-9e7c-2b6427f906b7", "owner_type": "agent", "owner_id": "705556a4-2bff-11ef-ad2b-3358935b1074", "flow_id": "d4df6ed6-f3a8-11ea-bf19-6f8063fdcfa1", "source": {"type": "sip", "target": "test_source@127.0.0.1:5061", "name": "test_source"}, "destinations": [{"type":"tel", "target": "+821100000001"}], "early_execution": true, "connect": true}`),
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"calls":[{"id":"cd561ba6-f3a8-11ea-b7ac-57b19fa28e09","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}],"groupcalls":[{"id":"4f1e2d54-7d52-4c1f-ad49-5b51feea0055","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","status":"","flow_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","ring_method":"","answer_method":"","answer_call_id":"00000000-0000-0000-0000-000000000000","call_ids":null,"answer_groupcall_id":"00000000-0000-0000-0000-000000000000","groupcall_ids":null,"call_count":0,"groupcall_count":0,"tm_create":"","tm_update":"","tm_delete":""}]}`),
			},
		},
		{
			name: "empty",

			customerID:     uuid.FromStringOrNil("34e72f78-7f51-11ec-a83b-cfc69cd4a641"),
			flowID:         uuid.FromStringOrNil("78fd1276-f3a8-11ea-9734-6735e73fd720"),
			masterCallID:   uuid.FromStringOrNil("a1c63272-8c91-11ec-8ee7-8b50458d3214"),
			source:         commonaddress.Address{},
			destinations:   []commonaddress.Address{},
			earlyExeuction: false,
			connect:        false,

			responseCalls: []*call.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("72d56d08-f3a8-11ea-9c0c-ef8258d54f42"),
					},
				},
			},
			responseGroupcalls: []*groupcall.Groupcall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("50e70d60-d722-43ce-a6b6-d69a28e36cbe"),
					},
				},
			},

			request: &sock.Request{
				URI:      "/v1/calls",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "34e72f78-7f51-11ec-a83b-cfc69cd4a641", "owner_type": "agent", "owner_id": "71145ca2-2bff-11ef-868b-17bb55414f44", "flow_id": "78fd1276-f3a8-11ea-9734-6735e73fd720", "master_call_id": "a1c63272-8c91-11ec-8ee7-8b50458d3214", "source": {}, "destinations": []}`),
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"calls":[{"id":"72d56d08-f3a8-11ea-9c0c-ef8258d54f42","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}],"groupcalls":[{"id":"50e70d60-d722-43ce-a6b6-d69a28e36cbe","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","status":"","flow_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","ring_method":"","answer_method":"","answer_call_id":"00000000-0000-0000-0000-000000000000","call_ids":null,"answer_groupcall_id":"00000000-0000-0000-0000-000000000000","groupcall_ids":null,"call_count":0,"groupcall_count":0,"tm_create":"","tm_update":"","tm_delete":""}]}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().CreateCallsOutgoing(gomock.Any(), tt.customerID, tt.flowID, tt.masterCallID, tt.source, tt.destinations, tt.earlyExeuction, tt.connect).Return(tt.responseCalls, tt.responseGroupcalls, nil)
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
		request *sock.Request

		responseCall *call.Call
		expectRes    *sock.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("8e49788e-fc9e-41a0-a73c-a24c030848ca"),
			&sock.Request{
				URI:      "/v1/calls/8e49788e-fc9e-41a0-a73c-a24c030848ca",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
				Data:     nil,
			},

			&call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8e49788e-fc9e-41a0-a73c-a24c030848ca"),
					CustomerID: uuid.FromStringOrNil("6ed4431a-7f51-11ec-8855-73041a5777e8"),
				},
				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8e49788e-fc9e-41a0-a73c-a24c030848ca","customer_id":"6ed4431a-7f51-11ec-8855-73041a5777e8","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
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
		name    string
		request *sock.Request

		responseCall *call.Call

		expectID  uuid.UUID
		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/calls/91a0b50e-f4ec-11ea-b64c-1bf53742d0d8/hangup",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     nil,
			},

			&call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("91a0b50e-f4ec-11ea-b64c-1bf53742d0d8"),
					CustomerID: uuid.FromStringOrNil("6ed4431a-7f51-11ec-8855-73041a5777e8"),
				},
			},

			uuid.FromStringOrNil("91a0b50e-f4ec-11ea-b64c-1bf53742d0d8"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"91a0b50e-f4ec-11ea-b64c-1bf53742d0d8","customer_id":"6ed4431a-7f51-11ec-8855-73041a5777e8","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().HangingUp(context.Background(), tt.expectID, call.HangupReasonNormal).Return(tt.responseCall, nil)

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

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)

	h := &listenHandler{
		sockHandler: mockSock,
		callHandler: mockCall,
	}

	tests := []struct {
		name      string
		call      *call.Call
		force     bool
		request   *sock.Request
		expectRes *sock.Response
	}{
		{
			"normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("37b3a214-0afd-11eb-88ea-7bdd69288e90"),
					CustomerID: uuid.FromStringOrNil("6efe9d54-7f51-11ec-95dc-4f25e8777704"),
				},
				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},
			},
			false,
			&sock.Request{
				URI:      "/v1/calls/37b3a214-0afd-11eb-88ea-7bdd69288e90/action-next",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"force":false}`),
			},
			&sock.Response{
				StatusCode: 200,
			},
		},
		{
			"next force",
			&call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("37b3a214-0afd-11eb-88ea-7bdd69288e90"),
					CustomerID: uuid.FromStringOrNil("6f29a850-7f51-11ec-a44d-0714d04b7ff2"),
				},
				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},
			},
			true,
			&sock.Request{
				URI:       "/v1/calls/37b3a214-0afd-11eb-88ea-7bdd69288e90/action-next",
				Method:    sock.RequestMethodPost,
				Publisher: "queue-manager",
				DataType:  "application/json",
				Data:      []byte(`{"force":true}`),
			},
			&sock.Response{
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
		request       *sock.Request

		responseCall *call.Call
		expectRes    *sock.Response
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bfcdc03a-25bf-11eb-a9b2-bba80a81835b"),
					CustomerID: uuid.FromStringOrNil("86711e1c-7f51-11ec-a807-0385a20de8ac"),
				},
			},
			uuid.FromStringOrNil("76490d6a-25c0-11eb-970b-3bf9ae938f41"),
			&sock.Request{
				URI:      "/v1/calls/bfcdc03a-25bf-11eb-a9b2-bba80a81835b/chained-call-ids",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"chained_call_id": "76490d6a-25c0-11eb-970b-3bf9ae938f41"}`),
			},

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bfcdc03a-25bf-11eb-a9b2-bba80a81835b"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bfcdc03a-25bf-11eb-a9b2-bba80a81835b","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
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
		request       *sock.Request

		responseCall *call.Call
		expectRes    *sock.Response
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0eaa2942-25c4-11eb-90a3-63fb2b029bae"),
				},
			},
			uuid.FromStringOrNil("0ee268f2-25c4-11eb-917c-07eef32616dc"),
			&sock.Request{
				URI:    "/v1/calls/0eaa2942-25c4-11eb-90a3-63fb2b029bae/chained-call-ids/0ee268f2-25c4-11eb-917c-07eef32616dc",
				Method: sock.RequestMethodDelete,
			},

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0eaa2942-25c4-11eb-90a3-63fb2b029bae"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0eaa2942-25c4-11eb-90a3-63fb2b029bae","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
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
		request *sock.Request

		responseCall *call.Call

		expectCallID          uuid.UUID
		expectExternalMediaID uuid.UUID
		expectExternalHost    string
		expectEncapsulation   externalmedia.Encapsulation
		expectTransport       externalmedia.Transport
		expectConnectionType  string
		expectFormat          string
		expectDirection       string

		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",

			&sock.Request{
				URI:      "/v1/calls/31255b7c-0a6b-11ec-87e2-afe5a545df76/external-media",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"external_media_id":"6e4d943c-b333-11ef-8f62-fb15e69cf29c","external_host": "127.0.0.1:5060", "encapsulation": "rtp", "transport": "udp", "connection_type": "client", "format": "ulaw", "direction": "both", "data": ""}`),
			},

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31255b7c-0a6b-11ec-87e2-afe5a545df76"),
				},
			},

			uuid.FromStringOrNil("31255b7c-0a6b-11ec-87e2-afe5a545df76"),
			uuid.FromStringOrNil("6e4d943c-b333-11ef-8f62-fb15e69cf29c"),
			"127.0.0.1:5060",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"31255b7c-0a6b-11ec-87e2-afe5a545df76","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				callHandler:          mockCall,
				externalMediaHandler: mockExternal,
			}

			mockCall.EXPECT().ExternalMediaStart(
				context.Background(),
				tt.expectCallID,
				tt.expectExternalMediaID,
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
		request *sock.Request

		responseCall *call.Call

		expectCallID uuid.UUID
		expectRes    *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/calls/7fcac8d6-9730-11ed-bd80-a764f6bc382e/external-media",
				Method: sock.RequestMethodDelete,
			},

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7fcac8d6-9730-11ed-bd80-a764f6bc382e"),
				},
			},

			uuid.FromStringOrNil("7fcac8d6-9730-11ed-bd80-a764f6bc382e"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7fcac8d6-9730-11ed-bd80-a764f6bc382e","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
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

		request *sock.Request

		id             uuid.UUID
		responseDigits string

		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/calls/669e567e-9016-11ec-9190-07c8a63f44a8/digits",
				Method: sock.RequestMethodGet,
			},

			uuid.FromStringOrNil("669e567e-9016-11ec-9190-07c8a63f44a8"),
			"1",

			&sock.Response{
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

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
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

		request *sock.Request

		id     uuid.UUID
		digits string

		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/calls/a5ca555a-9912-11ec-ab1a-2b341f06e3c0/digits",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"digits": "123"}`),
			},

			uuid.FromStringOrNil("a5ca555a-9912-11ec-ab1a-2b341f06e3c0"),
			"123",

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
		{
			"set to empty",
			&sock.Request{
				URI:      "/v1/calls/a5ca555a-9912-11ec-ab1a-2b341f06e3c0/digits",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"digits": ""}`),
			},

			uuid.FromStringOrNil("a5ca555a-9912-11ec-ab1a-2b341f06e3c0"),
			"",

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
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

		request *sock.Request

		id          uuid.UUID
		recordingID uuid.UUID

		responseCall *call.Call

		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/calls/0953474c-8fd2-11ed-a24a-7bf36392fef2/recording_id",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"recording_id": "09178464-8fd2-11ed-a1d7-5b5491e83cc7"}`),
			},

			uuid.FromStringOrNil("0953474c-8fd2-11ed-a24a-7bf36392fef2"),
			uuid.FromStringOrNil("09178464-8fd2-11ed-a1d7-5b5491e83cc7"),

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0953474c-8fd2-11ed-a24a-7bf36392fef2"),
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0953474c-8fd2-11ed-a24a-7bf36392fef2","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"set to empty",
			&sock.Request{
				URI:      "/v1/calls/0977e50c-8fd2-11ed-8684-e34a2113a8cc/recording_id",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"recording_id": "00000000-0000-0000-0000-000000000000"}`),
			},

			uuid.FromStringOrNil("0977e50c-8fd2-11ed-8684-e34a2113a8cc"),
			uuid.Nil,

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0977e50c-8fd2-11ed-8684-e34a2113a8cc"),
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0977e50c-8fd2-11ed-8684-e34a2113a8cc","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
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

		request *sock.Request

		responseCall *call.Call

		expectID           uuid.UUID
		expectFormat       recording.Format
		expectEndOfSilence int
		expectEndOfKey     string
		expectDuration     int
		expectOnEndFlowID  uuid.UUID

		expectRes *sock.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/calls/1c3dc786-9344-11ed-96a2-17c902204823/recording_start",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"format": "wav", "end_of_silence": 1000, "end_of_key": "#", "duration": 86400, "on_end_flow_id":"41a3daca-0547-11f0-8be6-0390b1e5bfb9"}`),
			},

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1c3dc786-9344-11ed-96a2-17c902204823"),
				},
			},

			expectID:           uuid.FromStringOrNil("1c3dc786-9344-11ed-96a2-17c902204823"),
			expectFormat:       recording.FormatWAV,
			expectEndOfSilence: 1000,
			expectEndOfKey:     "#",
			expectDuration:     86400,
			expectOnEndFlowID:  uuid.FromStringOrNil("41a3daca-0547-11f0-8be6-0390b1e5bfb9"),

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1c3dc786-9344-11ed-96a2-17c902204823","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().RecordingStart(gomock.Any(), tt.expectID, tt.expectFormat, tt.expectEndOfSilence, tt.expectEndOfKey, tt.expectDuration, tt.expectOnEndFlowID).Return(tt.responseCall, nil)
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

		request *sock.Request

		responseCall *call.Call

		expectID  uuid.UUID
		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/calls/1c73262e-9344-11ed-840d-37569c93274f/recording_stop",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1c73262e-9344-11ed-840d-37569c93274f"),
				},
			},

			uuid.FromStringOrNil("1c73262e-9344-11ed-840d-37569c93274f"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1c73262e-9344-11ed-840d-37569c93274f","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","channel_id":"","bridge_id":"","flow_id":"00000000-0000-0000-0000-000000000000","active_flow_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"external_media_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"action_next_hold":false,"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","dialroute_id":"00000000-0000-0000-0000-000000000000","dialroutes":null,"tm_ringing":"","tm_progressing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().RecordingStop(gomock.Any(), tt.expectID).Return(tt.responseCall, nil)
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

func Test_processV1CallsIDTalkPost(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		expectID       uuid.UUID
		expectText     string
		expectGender   string
		expectLanguage string
		expectRes      *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/calls/deb5c376-a4a2-11ed-b6d3-4f72b2fef2c1/talk",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"text":"hello world","gender":"female","language":"en-US"}`),
			},

			uuid.FromStringOrNil("deb5c376-a4a2-11ed-b6d3-4f72b2fef2c1"),
			"hello world",
			"female",
			"en-US",
			&sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().Talk(gomock.Any(), tt.expectID, false, tt.expectText, tt.expectGender, tt.expectLanguage).Return(nil)
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

func Test_processV1CallsIDPlayPost(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		expectID     uuid.UUID
		expectMedias []string
		expectRes    *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/calls/3dce82e3-ffca-47d3-96e6-0679195c7949/play",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"media_urls":["https://test.com/3d55ea46-5a91-442a-b1bc-d6100be0e11d.wav","https://test.com/6a094e77-a837-4511-8c4f-e2fec3aac44b.wav"]}`),
			},

			uuid.FromStringOrNil("3dce82e3-ffca-47d3-96e6-0679195c7949"),
			[]string{
				"https://test.com/3d55ea46-5a91-442a-b1bc-d6100be0e11d.wav",
				"https://test.com/6a094e77-a837-4511-8c4f-e2fec3aac44b.wav",
			},
			&sock.Response{
				StatusCode: 200,
			},
		},
		{
			"empty media urls",
			&sock.Request{
				URI:      "/v1/calls/c7d981fc-3119-4733-966c-88bfc61587fb/play",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{}`),
			},

			uuid.FromStringOrNil("c7d981fc-3119-4733-966c-88bfc61587fb"),
			nil,
			&sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().Play(gomock.Any(), tt.expectID, false, tt.expectMedias).Return(nil)
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

func Test_processV1CallsIDMediaStopPost(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		expectID  uuid.UUID
		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/calls/f2caffa0-ab13-4d7d-857e-a6ea64986f40/media_stop",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("f2caffa0-ab13-4d7d-857e-a6ea64986f40"),
			&sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().MediaStop(gomock.Any(), tt.expectID).Return(nil)
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

func Test_processV1CallsIDHoldPost(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		expectID  uuid.UUID
		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/calls/8d724fda-cef4-11ed-97c9-3be39b7e862c/hold",
				Method: sock.RequestMethodPost,
			},

			uuid.FromStringOrNil("8d724fda-cef4-11ed-97c9-3be39b7e862c"),
			&sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().HoldOn(gomock.Any(), tt.expectID).Return(nil)
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

func Test_processV1CallsIDHoldDelete(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		expectID  uuid.UUID
		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/calls/8da4a16a-cef4-11ed-875f-d7797c9e2710/hold",
				Method: sock.RequestMethodDelete,
			},

			uuid.FromStringOrNil("8da4a16a-cef4-11ed-875f-d7797c9e2710"),
			&sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().HoldOff(gomock.Any(), tt.expectID).Return(nil)
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

func Test_processV1CallsIDMutePost(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		expectID        uuid.UUID
		expectDirection call.MuteDirection

		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/calls/4066ee68-d13c-11ed-b9b3-8fe28137c5ad/mute",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"direction":"both"}`),
			},

			uuid.FromStringOrNil("4066ee68-d13c-11ed-b9b3-8fe28137c5ad"),
			call.MuteDirectionBoth,

			&sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().MuteOn(gomock.Any(), tt.expectID, tt.expectDirection).Return(nil)
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

func Test_processV1CallsIDMuteDelete(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		expectID        uuid.UUID
		expectDirection call.MuteDirection
		expectRes       *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/calls/40979fd6-d13c-11ed-8eb5-37d5f6dccaad/mute",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
				Data:     []byte(`{"direction":"both"}`),
			},

			uuid.FromStringOrNil("40979fd6-d13c-11ed-8eb5-37d5f6dccaad"),
			call.MuteDirectionBoth,

			&sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().MuteOff(gomock.Any(), tt.expectID, tt.expectDirection).Return(nil)
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

func Test_processV1CallsIDMOHPost(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		expectID  uuid.UUID
		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/calls/641e256a-d13c-11ed-a8f9-8f52289423ea/moh",
				Method: sock.RequestMethodPost,
			},

			uuid.FromStringOrNil("641e256a-d13c-11ed-a8f9-8f52289423ea"),
			&sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().MOHOn(gomock.Any(), tt.expectID).Return(nil)
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

func Test_processV1CallsIDMOHDelete(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		expectID  uuid.UUID
		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/calls/6445ce6c-d13c-11ed-aed2-0b27c33f8967/moh",
				Method: sock.RequestMethodDelete,
			},

			uuid.FromStringOrNil("6445ce6c-d13c-11ed-aed2-0b27c33f8967"),
			&sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().MOHOff(gomock.Any(), tt.expectID).Return(nil)
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

func Test_processV1CallsIDSilencePost(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		expectID  uuid.UUID
		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/calls/81e16bd4-d13c-11ed-8792-eb55e711e84c/silence",
				Method: sock.RequestMethodPost,
			},

			uuid.FromStringOrNil("81e16bd4-d13c-11ed-8792-eb55e711e84c"),
			&sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().SilenceOn(gomock.Any(), tt.expectID).Return(nil)
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

func Test_processV1CallsIDSilenceDelete(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		expectID  uuid.UUID
		expectRes *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/calls/8205fd8c-d13c-11ed-a6fd-f3011f730d2f/silence",
				Method: sock.RequestMethodDelete,
			},

			uuid.FromStringOrNil("8205fd8c-d13c-11ed-a6fd-f3011f730d2f"),
			&sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().SilenceOff(gomock.Any(), tt.expectID).Return(nil)
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
