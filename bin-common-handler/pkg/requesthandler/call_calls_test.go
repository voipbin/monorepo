package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_CallV1CallHealth(t *testing.T) {

	tests := []struct {
		name string

		callID     uuid.UUID
		delay      int
		retryCount int

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("5bfcdcd6-4c6e-11ec-bed9-8fe4c0fdf5ba"),
			0,
			3,

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/5bfcdcd6-4c6e-11ec-bed9-8fe4c0fdf5ba/health-check",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"retry_count":3}`),
			},
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
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.CallV1CallHealth(ctx, tt.callID, tt.delay, tt.retryCount)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallV1CallActionTimeout(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID
		delay  int
		action *fmaction.Action

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("c8ab1794-4c6e-11ec-86bc-773d32f65e3b"),
			0,
			&fmaction.Action{
				ID:        uuid.FromStringOrNil("eccec152-4c6e-11ec-bb47-d343ee142464"),
				Type:      fmaction.TypeAnswer,
				TMExecute: "2020-09-20T03:23:20.995000",
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/c8ab1794-4c6e-11ec-86bc-773d32f65e3b/action-timeout",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"action_id":"eccec152-4c6e-11ec-bb47-d343ee142464","action_type":"answer","tm_execute":"2020-09-20T03:23:20.995000"}`),
			},
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
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.CallV1CallActionTimeout(ctx, tt.callID, tt.delay, tt.action)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallV1CallActionNext(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID
		force  bool

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("bee79b78-4c6f-11ec-a254-cb0b4d8d4c9c"),
			false,

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/bee79b78-4c6f-11ec-a254-cb0b4d8d4c9c/action-next",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
		{
			"force",

			uuid.FromStringOrNil("bee79b78-4c6f-11ec-a254-cb0b4d8d4c9c"),
			true,

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/bee79b78-4c6f-11ec-a254-cb0b4d8d4c9c/action-next",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"force":true}`),
			},
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
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.CallV1CallActionNext(ctx, tt.callID, tt.force)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallV1CallsCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID     uuid.UUID
		flowID         uuid.UUID
		masterCallID   uuid.UUID
		source         *address.Address
		destinations   []address.Address
		ealryExecution bool
		connect        bool

		response *rabbitmqhandler.Response

		expectTarget        string
		expectRequest       *rabbitmqhandler.Request
		expectResCalls      []*cmcall.Call
		expectResGroupcalls []*cmgroupcall.Groupcall
	}{
		{
			name: "normal",

			customerID:   uuid.FromStringOrNil("3a09efda-7f52-11ec-a775-cfd868cdc292"),
			flowID:       uuid.FromStringOrNil("0783c168-4c70-11ec-a613-bfcd98aaa6da"),
			masterCallID: uuid.FromStringOrNil("ecd7b104-8c97-11ec-895d-67294ed5a4d0"),
			source: &address.Address{
				Type:   address.TypeTel,
				Target: "+821021656521",
			},
			destinations: []address.Address{
				{
					Type:   address.TypeTel,
					Target: "+821021656522",
				},
			},
			ealryExecution: true,
			connect:        true,

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"calls":[{"id":"fa0ddb32-25cd-11eb-a604-8b239b305055"}],"groupcalls":[{"id":"69b105a6-939b-4eb0-99a5-0efa5b3cd80e"}]}`),
			},

			expectTarget: "bin-manager.call-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/calls",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"flow_id":"0783c168-4c70-11ec-a613-bfcd98aaa6da","customer_id":"3a09efda-7f52-11ec-a775-cfd868cdc292","master_call_id":"ecd7b104-8c97-11ec-895d-67294ed5a4d0","source":{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""},"destinations":[{"type":"tel","target":"+821021656522","target_name":"","name":"","detail":""}],"early_execution":true,"connect":true}`),
			},
			expectResCalls: []*cmcall.Call{
				{
					ID: uuid.FromStringOrNil("fa0ddb32-25cd-11eb-a604-8b239b305055"),
				},
			},
			expectResGroupcalls: []*cmgroupcall.Groupcall{
				{
					ID: uuid.FromStringOrNil("69b105a6-939b-4eb0-99a5-0efa5b3cd80e"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			resCalls, resGroupcalls, err := reqHandler.CallV1CallsCreate(ctx, tt.customerID, tt.flowID, tt.masterCallID, tt.source, tt.destinations, tt.ealryExecution, tt.connect)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(resCalls, tt.expectResCalls) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResCalls, resCalls)
			}
			if !reflect.DeepEqual(resGroupcalls, tt.expectResGroupcalls) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResGroupcalls, resGroupcalls)
			}

		})
	}
}

func Test_CallV1CallCreateWithID(t *testing.T) {

	tests := []struct {
		name string

		callID         uuid.UUID
		customerID     uuid.UUID
		flowID         uuid.UUID
		activeflowID   uuid.UUID
		masterCallID   uuid.UUID
		groupcallID    uuid.UUID
		earlyExecution bool
		connect        bool

		source      *address.Address
		destination *address.Address

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *cmcall.Call
	}{
		{
			"normal",

			uuid.FromStringOrNil("9dcdc9a0-4d1c-11ec-81cc-bf06212a283e"),
			uuid.FromStringOrNil("45a4dbac-7f52-11ec-98a8-7f1e6d2fae52"),
			uuid.FromStringOrNil("9f4b89b6-4d1c-11ec-a565-af220567858d"),
			uuid.FromStringOrNil("0a5273c9-73ac-4590-87de-4c7f33da7614"),
			uuid.FromStringOrNil("f993c284-8c97-11ec-aaa3-a76b1106d031"),
			uuid.FromStringOrNil("8214ceaa-bbe0-11ed-9ae2-b72d8846362b"),
			true,
			true,

			&address.Address{
				Type:   address.TypeTel,
				Target: "+821021656521",
			},
			&address.Address{
				Type:   address.TypeTel,
				Target: "+821021656522",
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/9dcdc9a0-4d1c-11ec-81cc-bf06212a283e",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"flow_id":"9f4b89b6-4d1c-11ec-a565-af220567858d","activeflow_id":"0a5273c9-73ac-4590-87de-4c7f33da7614","customer_id":"45a4dbac-7f52-11ec-98a8-7f1e6d2fae52","master_call_id":"f993c284-8c97-11ec-aaa3-a76b1106d031","source":{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""},"destination":{"type":"tel","target":"+821021656522","target_name":"","name":"","detail":""},"groupcall_id":"8214ceaa-bbe0-11ed-9ae2-b72d8846362b","early_execution":true,"connect":true}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9dcdc9a0-4d1c-11ec-81cc-bf06212a283e"}`),
			},
			&cmcall.Call{
				ID: uuid.FromStringOrNil("9dcdc9a0-4d1c-11ec-81cc-bf06212a283e"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1CallCreateWithID(ctx, tt.callID, tt.customerID, tt.flowID, tt.activeflowID, tt.masterCallID, tt.source, tt.destination, tt.groupcallID, tt.earlyExecution, tt.connect)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1CallGet(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *cmcall.Call
	}{
		{
			"normal",

			uuid.FromStringOrNil("7ab80df4-4c72-11ec-b095-17146a0e7e4c"),

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls/7ab80df4-4c72-11ec-b095-17146a0e7e4c",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7ab80df4-4c72-11ec-b095-17146a0e7e4c"}`),
			},
			&cmcall.Call{
				ID: uuid.FromStringOrNil("7ab80df4-4c72-11ec-b095-17146a0e7e4c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1CallGet(ctx, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1CallGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     []cmcall.Call
	}{
		{
			"normal",

			"2020-09-20T03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_deleted=false",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d3ce27ac-4c72-11ec-b790-6b79445cbb01"}]`),
			},
			[]cmcall.Call{
				{
					ID: uuid.FromStringOrNil("d3ce27ac-4c72-11ec-b790-6b79445cbb01"),
				},
			},
		},
		{
			"2 calls",

			"2020-09-20T03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_deleted=false",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"11cfd8e8-4c73-11ec-8f06-b73cd86fc9ae"},{"id":"12237ce6-4c73-11ec-8a2a-57b7a8d6a6f4"}]`),
			},
			[]cmcall.Call{
				{
					ID: uuid.FromStringOrNil("11cfd8e8-4c73-11ec-8f06-b73cd86fc9ae"),
				},
				{
					ID: uuid.FromStringOrNil("12237ce6-4c73-11ec-8a2a-57b7a8d6a6f4"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1CallGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CMCallAddChainedCall(t *testing.T) {

	tests := []struct {
		name string

		callID        uuid.UUID
		chainedCallID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cmcall.Call
	}{
		{
			"normal",

			uuid.FromStringOrNil("887a7600-25c9-11eb-ab60-338d7ef0ba0f"),
			uuid.FromStringOrNil("8d48ded8-25c9-11eb-a8da-a7bcaada697c"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "appliation/json",
				Data:       []byte(`{"id":"887a7600-25c9-11eb-ab60-338d7ef0ba0f"}`),
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/887a7600-25c9-11eb-ab60-338d7ef0ba0f/chained-call-ids",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"chained_call_id":"8d48ded8-25c9-11eb-a8da-a7bcaada697c"}`),
			},
			&cmcall.Call{
				ID: uuid.FromStringOrNil("887a7600-25c9-11eb-ab60-338d7ef0ba0f"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1CallAddChainedCall(ctx, tt.callID, tt.chainedCallID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_CMCallRemoveChainedCall(t *testing.T) {

	tests := []struct {
		name string

		callID        uuid.UUID
		chainedCallID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cmcall.Call
	}{
		{
			"normal",

			uuid.FromStringOrNil("1ced9274-8ee0-11ec-8c36-13795e573d73"),
			uuid.FromStringOrNil("1d38dcd4-8ee0-11ec-ace4-178f58435f40"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "appliation/json",
				Data:       []byte(`{"id":"1ced9274-8ee0-11ec-8c36-13795e573d73"}`),
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls/1ced9274-8ee0-11ec-8c36-13795e573d73/chained-call-ids/1d38dcd4-8ee0-11ec-ace4-178f58435f40",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&cmcall.Call{
				ID: uuid.FromStringOrNil("1ced9274-8ee0-11ec-8c36-13795e573d73"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1CallRemoveChainedCall(ctx, tt.callID, tt.chainedCallID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_CallV1CallDelete(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *cmcall.Call
	}{
		{
			"normal",

			uuid.FromStringOrNil("045c4e0d-7838-46bf-b28d-3aeaa943a53e"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"045c4e0d-7838-46bf-b28d-3aeaa943a53e"}`),
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls/045c4e0d-7838-46bf-b28d-3aeaa943a53e",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&cmcall.Call{
				ID: uuid.FromStringOrNil("045c4e0d-7838-46bf-b28d-3aeaa943a53e"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1CallDelete(ctx, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_CallV1CallHangup(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *cmcall.Call
	}{
		{
			"normal",

			uuid.FromStringOrNil("fa0ddb32-25cd-11eb-a604-8b239b305055"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"fa0ddb32-25cd-11eb-a604-8b239b305055","customer_id":"a789f1d6-7f52-11ec-b563-e3d43178d814","asterisk_id":"","channel_id":"","flow_id":"59518eae-ed66-11ea-85ef-b77bdbc74ccc","conf_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":"","tm_execute":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/fa0ddb32-25cd-11eb-a604-8b239b305055/hangup",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
			},
			&cmcall.Call{
				ID:          uuid.FromStringOrNil("fa0ddb32-25cd-11eb-a604-8b239b305055"),
				CustomerID:  uuid.FromStringOrNil("a789f1d6-7f52-11ec-b563-e3d43178d814"),
				FlowID:      uuid.FromStringOrNil("59518eae-ed66-11ea-85ef-b77bdbc74ccc"),
				Source:      address.Address{},
				Destination: address.Address{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1CallHangup(ctx, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_CallV1CallExternalMediaStart(t *testing.T) {

	tests := []struct {
		name string

		callID         uuid.UUID
		externalHost   string
		encapsulation  string
		transport      string
		connectionType string
		format         string
		direction      string

		response *rabbitmqhandler.Response

		expectRequest *rabbitmqhandler.Request
		expectRes     *cmcall.Call
	}{
		{
			"normal",

			uuid.FromStringOrNil("a099a2a4-0ac7-11ec-b8ae-438c5d2fe6fb"),
			"localhost:5060",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a099a2a4-0ac7-11ec-b8ae-438c5d2fe6fb"}`),
			},

			&rabbitmqhandler.Request{
				URI:      "/v1/calls/a099a2a4-0ac7-11ec-b8ae-438c5d2fe6fb/external-media",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"external_host":"localhost:5060","encapsulation":"rtp","transport":"udp","connection_type":"client","format":"ulaw","direction":"both"}`),
			},
			&cmcall.Call{
				ID: uuid.FromStringOrNil("a099a2a4-0ac7-11ec-b8ae-438c5d2fe6fb"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1CallExternalMediaStart(ctx, tt.callID, tt.externalHost, tt.encapsulation, tt.transport, tt.connectionType, tt.format, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1CallExternalMediaStop(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		response *rabbitmqhandler.Response

		expectRequest *rabbitmqhandler.Request
		expectRes     *cmcall.Call
	}{
		{
			"normal",

			uuid.FromStringOrNil("487233ec-97c1-11ed-968d-47ee0ef18dbf"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"487233ec-97c1-11ed-968d-47ee0ef18dbf"}`),
			},

			&rabbitmqhandler.Request{
				URI:    "/v1/calls/487233ec-97c1-11ed-968d-47ee0ef18dbf/external-media",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&cmcall.Call{
				ID: uuid.FromStringOrNil("487233ec-97c1-11ed-968d-47ee0ef18dbf"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1CallExternalMediaStop(ctx, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1CallGetDigits(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		response *rabbitmqhandler.Response

		expectRequest *rabbitmqhandler.Request
		expectRes     string
	}{
		{
			"normal",

			uuid.FromStringOrNil("3f73caf8-901a-11ec-8ec8-b7367d212083"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"digits":"1"}`),
			},

			&rabbitmqhandler.Request{
				URI:      "/v1/calls/3f73caf8-901a-11ec-8ec8-b7367d212083/digits",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			"1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1CallGetDigits(ctx, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1CallSendDigits(t *testing.T) {

	tests := []struct {
		name string

		callID        uuid.UUID
		digits        string
		expectRequest *rabbitmqhandler.Request

		response *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("be3f07ee-9916-11ec-a7c8-ef03823980a7"),
			"123",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/be3f07ee-9916-11ec-a7c8-ef03823980a7/digits",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"digits":"123"}`),
			},

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
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CallV1CallSendDigits(ctx, tt.callID, tt.digits); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallV1CallRecordingStart(t *testing.T) {

	tests := []struct {
		name string

		callID       uuid.UUID
		format       cmrecording.Format
		endOfSilence int
		endOfKey     string
		duration     int

		response *rabbitmqhandler.Response

		expectRequest *rabbitmqhandler.Request
		expectRes     *cmcall.Call
	}{
		{
			"normal",

			uuid.FromStringOrNil("6533f61e-9348-11ed-83bc-ab5a0adfe5e5"),
			cmrecording.FormatWAV,
			1000,
			"#",
			86400,

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"6533f61e-9348-11ed-83bc-ab5a0adfe5e5"}`),
			},

			&rabbitmqhandler.Request{
				URI:      "/v1/calls/6533f61e-9348-11ed-83bc-ab5a0adfe5e5/recording_start",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"format":"wav","end_of_silence":1000,"end_of_key":"#","duration":86400}`),
			},
			&cmcall.Call{
				ID: uuid.FromStringOrNil("6533f61e-9348-11ed-83bc-ab5a0adfe5e5"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1CallRecordingStart(ctx, tt.callID, tt.format, tt.endOfSilence, tt.endOfKey, tt.duration)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1CallRecordingStop(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		response *rabbitmqhandler.Response

		expectRequest *rabbitmqhandler.Request
		expectRes     *cmcall.Call
	}{
		{
			"normal",

			uuid.FromStringOrNil("6593f41a-9348-11ed-bdd2-3b5bf8891acb"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"6593f41a-9348-11ed-bdd2-3b5bf8891acb"}`),
			},

			&rabbitmqhandler.Request{
				URI:    "/v1/calls/6593f41a-9348-11ed-bdd2-3b5bf8891acb/recording_stop",
				Method: rabbitmqhandler.RequestMethodPost,
			},
			&cmcall.Call{
				ID: uuid.FromStringOrNil("6593f41a-9348-11ed-bdd2-3b5bf8891acb"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1CallRecordingStop(ctx, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1CallUpdateConfbridgeID(t *testing.T) {

	tests := []struct {
		name string

		callID       uuid.UUID
		confbridgeID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *cmcall.Call
	}{
		{
			"normal",

			uuid.FromStringOrNil("6c2d5016-467d-4d53-86ce-f5b5fc451b1c"),
			uuid.FromStringOrNil("9955fda3-fc5e-40eb-9c2d-7d0152e3c6ba"),

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/6c2d5016-467d-4d53-86ce-f5b5fc451b1c/confbridge_id",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"confbridge_id":"9955fda3-fc5e-40eb-9c2d-7d0152e3c6ba"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6c2d5016-467d-4d53-86ce-f5b5fc451b1c"}`),
			},
			&cmcall.Call{
				ID: uuid.FromStringOrNil("6c2d5016-467d-4d53-86ce-f5b5fc451b1c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1CallUpdateConfbridgeID(ctx, tt.callID, tt.confbridgeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1CallTalk(t *testing.T) {

	tests := []struct {
		name string

		callID         uuid.UUID
		text           string
		gender         string
		language       string
		requestTimeout int

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("966842b8-a4b3-11ed-afc1-cfd28f99c181"),
			"hello world",
			"female",
			"en-US",
			10000,

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/966842b8-a4b3-11ed-afc1-cfd28f99c181/talk",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"text":"hello world","gender":"female","language":"en-US"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CallV1CallTalk(ctx, tt.callID, tt.text, tt.gender, tt.language, tt.requestTimeout); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallV1CallPlay(t *testing.T) {

	tests := []struct {
		name string

		callID    uuid.UUID
		meidaURLs []string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("ae44f0c7-887b-4cd2-9f30-a7ff80dd7300"),
			[]string{
				"https://test.com/735efc89-5255-4ca0-8181-8ad802d2e24b.wav",
				"https://test.com/a3366726-8fcc-4730-a03b-256bc343c1ea.wav",
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/ae44f0c7-887b-4cd2-9f30-a7ff80dd7300/play",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"media_urls":["https://test.com/735efc89-5255-4ca0-8181-8ad802d2e24b.wav","https://test.com/a3366726-8fcc-4730-a03b-256bc343c1ea.wav"]}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CallV1CallPlay(ctx, tt.callID, tt.meidaURLs); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallV1CallMediaStop(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("582ad5a4-e787-4b0b-8480-09253372a518"),

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls/582ad5a4-e787-4b0b-8480-09253372a518/media_stop",
				Method: rabbitmqhandler.RequestMethodPost,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CallV1CallMediaStop(ctx, tt.callID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallV1CallHoldOn(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("b36a092a-cef5-11ed-8c7c-f765b9f87cd6"),

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls/b36a092a-cef5-11ed-8c7c-f765b9f87cd6/hold",
				Method: rabbitmqhandler.RequestMethodPost,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CallV1CallHoldOn(ctx, tt.callID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallV1CallHoldOff(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("b39a2b6e-cef5-11ed-8aee-9b00ffa23b49"),

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls/b39a2b6e-cef5-11ed-8aee-9b00ffa23b49/hold",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CallV1CallHoldOff(ctx, tt.callID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallV1CallMuteOn(t *testing.T) {

	tests := []struct {
		name string

		callID    uuid.UUID
		direction call.MuteDirection

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("b3c32e88-cef5-11ed-9f30-1b12722669f5"),
			call.MuteDirectionBoth,

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/b3c32e88-cef5-11ed-9f30-1b12722669f5/mute",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"direction":"both"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CallV1CallMuteOn(ctx, tt.callID, tt.direction); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallV1CallMuteOff(t *testing.T) {

	tests := []struct {
		name string

		callID    uuid.UUID
		direction call.MuteDirection

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("b3ebe8dc-cef5-11ed-a05a-8730dc1ef961"),
			call.MuteDirectionBoth,

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/b3ebe8dc-cef5-11ed-a05a-8730dc1ef961/mute",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"direction":"both"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CallV1CallMuteOff(ctx, tt.callID, tt.direction); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallV1CallMusicOnHoldOn(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("52ce3c32-d0ba-11ed-a847-9b2877b2f2f3"),

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls/52ce3c32-d0ba-11ed-a847-9b2877b2f2f3/moh",
				Method: rabbitmqhandler.RequestMethodPost,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CallV1CallMusicOnHoldOn(ctx, tt.callID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallV1CallMusicOnHoldOff(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("5307dca8-d0ba-11ed-a3ae-eb92cdba1a1e"),

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls/5307dca8-d0ba-11ed-a3ae-eb92cdba1a1e/moh",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CallV1CallMusicOnHoldOff(ctx, tt.callID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallV1CallSilenceOn(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("75f7278c-d0ba-11ed-9015-b7646ccfa33e"),

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls/75f7278c-d0ba-11ed-9015-b7646ccfa33e/silence",
				Method: rabbitmqhandler.RequestMethodPost,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CallV1CallSilenceOn(ctx, tt.callID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallV1CallSilenceOff(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("76367266-d0ba-11ed-a4fd-43b05781859b"),

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/calls/76367266-d0ba-11ed-a4fd-43b05781859b/silence",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CallV1CallSilenceOff(ctx, tt.callID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
