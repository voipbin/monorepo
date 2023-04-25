package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"

	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_CallV1GroupcallGets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     []cmgroupcall.Groupcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("aa584af4-be33-11ed-aa3e-af30b76641da"),
			"2020-09-20T03:23:20.995000",
			10,

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/groupcalls?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&customer_id=aa584af4-be33-11ed-aa3e-af30b76641da",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"ac47b7fa-be33-11ed-b247-f381628bad10"}]`),
			},
			[]cmgroupcall.Groupcall{
				{
					ID: uuid.FromStringOrNil("ac47b7fa-be33-11ed-b247-f381628bad10"),
				},
			},
		},
		{
			"2 calls",

			uuid.FromStringOrNil("aca75b42-be33-11ed-95f1-dfdca58a24cf"),
			"2020-09-20T03:23:20.995000",
			10,

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/groupcalls?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&customer_id=aca75b42-be33-11ed-95f1-dfdca58a24cf",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d789d812-be33-11ed-9f54-1b18b82aa8f8"},{"id":"d7af6dfc-be33-11ed-939f-b3e2f9be4bd5"}]`),
			},
			[]cmgroupcall.Groupcall{
				{
					ID: uuid.FromStringOrNil("d789d812-be33-11ed-9f54-1b18b82aa8f8"),
				},
				{
					ID: uuid.FromStringOrNil("d7af6dfc-be33-11ed-939f-b3e2f9be4bd5"),
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

			res, err := reqHandler.CallV1GroupcallGets(ctx, tt.customerID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1GroupcallCreate(t *testing.T) {

	tests := []struct {
		name string

		ctx          context.Context
		customerID   uuid.UUID
		source       commonaddress.Address
		destinations []commonaddress.Address
		flowID       uuid.UUID
		masterCallID uuid.UUID
		ringMethod   cmgroupcall.RingMethod
		answerMethod cmgroupcall.AnswerMethod

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cmgroupcall.Groupcall
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("2ac49ec8-bbae-11ed-b9cd-8f47fd0602b9"),
			source: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000003",
				},
			},
			flowID:       uuid.FromStringOrNil("2b1f1682-bbae-11ed-b06b-3be413b33b07"),
			masterCallID: uuid.FromStringOrNil("2b4f5b44-bbae-11ed-9629-dfffd3ac6a43"),
			ringMethod:   cmgroupcall.RingMethodRingAll,
			answerMethod: cmgroupcall.AnswerMethodHangupOthers,

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2b7dc4ac-bbae-11ed-b868-f762b6f7fd23"}`),
			},

			expectTarget: "bin-manager.call-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/groupcalls",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"2ac49ec8-bbae-11ed-b9cd-8f47fd0602b9","source":{"type":"tel","target":"+821100000001","target_name":"","name":"","detail":""},"destinations":[{"type":"tel","target":"+821100000002","target_name":"","name":"","detail":""},{"type":"tel","target":"+821100000003","target_name":"","name":"","detail":""}],"flow_id":"2b1f1682-bbae-11ed-b06b-3be413b33b07","master_call_id":"2b4f5b44-bbae-11ed-9629-dfffd3ac6a43","ring_method":"ring_all","answer_method":"hangup_others"}`),
			},
			expectRes: &cmgroupcall.Groupcall{
				ID: uuid.FromStringOrNil("2b7dc4ac-bbae-11ed-b868-f762b6f7fd23"),
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

			res, err := reqHandler.CallV1GroupcallCreate(ctx, tt.customerID, tt.source, tt.destinations, tt.flowID, tt.masterCallID, tt.ringMethod, tt.answerMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1GroupcallGet(t *testing.T) {

	tests := []struct {
		name string

		groupcallID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cmgroupcall.Groupcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("1717ba30-be34-11ed-87e7-5739c7ea8622"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1717ba30-be34-11ed-87e7-5739c7ea8622"}`),
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/groupcalls/1717ba30-be34-11ed-87e7-5739c7ea8622",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&cmgroupcall.Groupcall{
				ID: uuid.FromStringOrNil("1717ba30-be34-11ed-87e7-5739c7ea8622"),
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

			res, err := reqHandler.CallV1GroupcallGet(ctx, tt.groupcallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_CallV1GroupcallDelete(t *testing.T) {

	tests := []struct {
		name string

		groupcallID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cmgroupcall.Groupcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("06d9ec2a-be33-11ed-acc5-876b594da79c"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"06d9ec2a-be33-11ed-acc5-876b594da79c"}`),
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/groupcalls/06d9ec2a-be33-11ed-acc5-876b594da79c",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&cmgroupcall.Groupcall{
				ID: uuid.FromStringOrNil("06d9ec2a-be33-11ed-acc5-876b594da79c"),
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

			res, err := reqHandler.CallV1GroupcallDelete(ctx, tt.groupcallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_CallV1GroupcallHangup(t *testing.T) {

	tests := []struct {
		name string

		groupcallID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cmgroupcall.Groupcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("3d82215c-be33-11ed-aed4-7b9daa884e9f"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3d82215c-be33-11ed-aed4-7b9daa884e9f"}`),
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/groupcalls/3d82215c-be33-11ed-aed4-7b9daa884e9f/hangup",
				Method: rabbitmqhandler.RequestMethodPost,
			},
			&cmgroupcall.Groupcall{
				ID: uuid.FromStringOrNil("3d82215c-be33-11ed-aed4-7b9daa884e9f"),
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

			res, err := reqHandler.CallV1GroupcallHangup(ctx, tt.groupcallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_CallV1GroupcallDecreaseGroupcallCount(t *testing.T) {
	tests := []struct {
		name string

		groupcallID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cmgroupcall.Groupcall
	}{
		{
			name: "normal",

			groupcallID: uuid.FromStringOrNil("1441cca6-e328-11ed-9fec-d34d08a9a0b0"),

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"1441cca6-e328-11ed-9fec-d34d08a9a0b0"}`),
			},

			expectTarget: "bin-manager.call-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:    "/v1/groupcalls/1441cca6-e328-11ed-9fec-d34d08a9a0b0/decrease_groupcall_count",
				Method: rabbitmqhandler.RequestMethodPost,
			},
			expectRes: &cmgroupcall.Groupcall{
				ID: uuid.FromStringOrNil("1441cca6-e328-11ed-9fec-d34d08a9a0b0"),
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

			res, err := reqHandler.CallV1GroupcallDecreaseGroupcallCount(ctx, tt.groupcallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1GroupcallUpdateAnswerGroupcallID(t *testing.T) {
	tests := []struct {
		name string

		groupcallID       uuid.UUID
		answerGroupcallID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cmgroupcall.Groupcall
	}{
		{
			name: "normal",

			groupcallID:       uuid.FromStringOrNil("e50ab9b0-e328-11ed-abe1-abcb6aaa1b47"),
			answerGroupcallID: uuid.FromStringOrNil("e5399dc0-e328-11ed-af77-6f6bf2e02462"),

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"e50ab9b0-e328-11ed-abe1-abcb6aaa1b47"}`),
			},

			expectTarget: "bin-manager.call-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/groupcalls/e50ab9b0-e328-11ed-abe1-abcb6aaa1b47/answer_groupcall_id",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"answer_groupcall_id":"e5399dc0-e328-11ed-af77-6f6bf2e02462"}`),
			},
			expectRes: &cmgroupcall.Groupcall{
				ID: uuid.FromStringOrNil("e50ab9b0-e328-11ed-abe1-abcb6aaa1b47"),
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

			res, err := reqHandler.CallV1GroupcallUpdateAnswerGroupcallID(ctx, tt.groupcallID, tt.answerGroupcallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
