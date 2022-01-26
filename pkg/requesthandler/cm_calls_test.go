package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestCMV1CallHealth(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	type test struct {
		name string

		callID     uuid.UUID
		delay      int
		retryCount int

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("5bfcdcd6-4c6e-11ec-bed9-8fe4c0fdf5ba"),
			0,
			0,

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/5bfcdcd6-4c6e-11ec-bed9-8fe4c0fdf5ba/health-check",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"retry_count":0,"delay":0}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.CMV1CallHealth(ctx, tt.callID, tt.delay, tt.retryCount)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestCMV1CallActionTimeout(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	type test struct {
		name string

		callID uuid.UUID
		delay  int
		action *fmaction.Action

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}

	tests := []test{
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
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.CMV1CallActionTimeout(ctx, tt.callID, tt.delay, tt.action)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestCMV1CallActionNext(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	type test struct {
		name string

		callID uuid.UUID
		force  bool

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("bee79b78-4c6f-11ec-a254-cb0b4d8d4c9c"),
			false,

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/bee79b78-4c6f-11ec-a254-cb0b4d8d4c9c/action-next",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"force":false}`),
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
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.CMV1CallActionNext(ctx, tt.callID, tt.force)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestCMV1CallCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	type test struct {
		name string

		userID      uint64
		flowID      uuid.UUID
		source      *cmaddress.Address
		destination *cmaddress.Address

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *cmcall.Call
	}

	tests := []test{
		{
			"normal",

			1,
			uuid.FromStringOrNil("0783c168-4c70-11ec-a613-bfcd98aaa6da"),
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821021656521",
			},
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821021656522",
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"flow_id":"0783c168-4c70-11ec-a613-bfcd98aaa6da","user_id":1,"source":{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""},"destination":{"type":"tel","target":"+821021656522","target_name":"","name":"","detail":""}}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"fa0ddb32-25cd-11eb-a604-8b239b305055"}`),
			},
			&cmcall.Call{
				ID: uuid.FromStringOrNil("fa0ddb32-25cd-11eb-a604-8b239b305055"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CMV1CallCreate(ctx, tt.userID, tt.flowID, tt.source, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestCMV1CallCreateWithID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	type test struct {
		name string

		callID      uuid.UUID
		userID      uint64
		flowID      uuid.UUID
		source      *cmaddress.Address
		destination *cmaddress.Address

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *cmcall.Call
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("9dcdc9a0-4d1c-11ec-81cc-bf06212a283e"),
			1,
			uuid.FromStringOrNil("9f4b89b6-4d1c-11ec-a565-af220567858d"),
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821021656521",
			},
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821021656522",
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/9dcdc9a0-4d1c-11ec-81cc-bf06212a283e",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"flow_id":"9f4b89b6-4d1c-11ec-a565-af220567858d","user_id":1,"source":{"type":"tel","target":"+821021656521","target_name":"","name":"","detail":""},"destination":{"type":"tel","target":"+821021656522","target_name":"","name":"","detail":""}}`),
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
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CMV1CallCreateWithID(ctx, tt.callID, tt.userID, tt.flowID, tt.source, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestCMV1CallGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	type test struct {
		name string

		callID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *cmcall.Call
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("7ab80df4-4c72-11ec-b095-17146a0e7e4c"),

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/7ab80df4-4c72-11ec-b095-17146a0e7e4c",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
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
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CMV1CallGet(ctx, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestCMV1CallGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	type test struct {
		name string

		userID    uint64
		pageToken string
		pageSize  uint64

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     []cmcall.Call
	}

	tests := []test{
		{
			"normal",

			1,
			"2020-09-20T03:23:20.995000",
			10,

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&user_id=1",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
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

			1,
			"2020-09-20T03:23:20.995000",
			10,

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&user_id=1",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
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
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CMV1CallGets(ctx, tt.userID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestCMCallAddChainedCall(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	type test struct {
		name string

		callID        uuid.UUID
		chainedCallID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		// expectResult  *flow.Flow
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("887a7600-25c9-11eb-ab60-338d7ef0ba0f"),
			uuid.FromStringOrNil("8d48ded8-25c9-11eb-a8da-a7bcaada697c"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/887a7600-25c9-11eb-ab60-338d7ef0ba0f/chained-call-ids",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"chained_call_id":"8d48ded8-25c9-11eb-a8da-a7bcaada697c"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CMV1CallAddChainedCall(ctx, tt.callID, tt.chainedCallID); err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}
		})
	}
}

func TestCMCallHangup(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	type test struct {
		name string

		callID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *cmcall.Call
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("fa0ddb32-25cd-11eb-a604-8b239b305055"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"fa0ddb32-25cd-11eb-a604-8b239b305055","user_id":1,"asterisk_id":"","channel_id":"","flow_id":"59518eae-ed66-11ea-85ef-b77bdbc74ccc","conf_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":"","tm_execute":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/fa0ddb32-25cd-11eb-a604-8b239b305055",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},
			&cmcall.Call{
				ID:          uuid.FromStringOrNil("fa0ddb32-25cd-11eb-a604-8b239b305055"),
				UserID:      1,
				FlowID:      uuid.FromStringOrNil("59518eae-ed66-11ea-85ef-b77bdbc74ccc"),
				Source:      cmaddress.Address{},
				Destination: cmaddress.Address{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CMV1CallHangup(ctx, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}
