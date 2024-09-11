package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_CallV1GroupcallGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     []cmgroupcall.Groupcall
	}{
		{
			"normal",

			"2020-09-20T03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			"/v1/groupcalls?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			"bin-manager.call-manager.request",
			&sock.Request{
				URI:    "/v1/groupcalls?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"ac47b7fa-be33-11ed-b247-f381628bad10"}]`),
			},
			[]cmgroupcall.Groupcall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("ac47b7fa-be33-11ed-b247-f381628bad10"),
					},
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

			"/v1/groupcalls?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			"bin-manager.call-manager.request",
			&sock.Request{
				URI:    "/v1/groupcalls?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d789d812-be33-11ed-9f54-1b18b82aa8f8"},{"id":"d7af6dfc-be33-11ed-939f-b3e2f9be4bd5"}]`),
			},
			[]cmgroupcall.Groupcall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d789d812-be33-11ed-9f54-1b18b82aa8f8"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d7af6dfc-be33-11ed-939f-b3e2f9be4bd5"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := h.CallV1GroupcallGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
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

		ctx               context.Context
		id                uuid.UUID
		customerID        uuid.UUID
		source            commonaddress.Address
		destinations      []commonaddress.Address
		flowID            uuid.UUID
		masterCallID      uuid.UUID
		masterGroupcallID uuid.UUID
		ringMethod        cmgroupcall.RingMethod
		answerMethod      cmgroupcall.AnswerMethod

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cmgroupcall.Groupcall
	}{
		{
			name: "normal",

			id:         uuid.FromStringOrNil("812e519b-bbf6-415f-9c49-b4db5c76f68d"),
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
			flowID:            uuid.FromStringOrNil("2b1f1682-bbae-11ed-b06b-3be413b33b07"),
			masterCallID:      uuid.FromStringOrNil("2b4f5b44-bbae-11ed-9629-dfffd3ac6a43"),
			masterGroupcallID: uuid.FromStringOrNil("e6310999-eab1-48fc-b0ce-f5ee55743864"),
			ringMethod:        cmgroupcall.RingMethodRingAll,
			answerMethod:      cmgroupcall.AnswerMethodHangupOthers,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2b7dc4ac-bbae-11ed-b868-f762b6f7fd23"}`),
			},

			expectTarget: "bin-manager.call-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/groupcalls",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"id":"812e519b-bbf6-415f-9c49-b4db5c76f68d","customer_id":"2ac49ec8-bbae-11ed-b9cd-8f47fd0602b9","flow_id":"2b1f1682-bbae-11ed-b06b-3be413b33b07","source":{"type":"tel","target":"+821100000001","target_name":"","name":"","detail":""},"destinations":[{"type":"tel","target":"+821100000002","target_name":"","name":"","detail":""},{"type":"tel","target":"+821100000003","target_name":"","name":"","detail":""}],"master_call_id":"2b4f5b44-bbae-11ed-9629-dfffd3ac6a43","master_groupcall_id":"e6310999-eab1-48fc-b0ce-f5ee55743864","ring_method":"ring_all","answer_method":"hangup_others"}`),
			},
			expectRes: &cmgroupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2b7dc4ac-bbae-11ed-b868-f762b6f7fd23"),
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

			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1GroupcallCreate(ctx, tt.id, tt.customerID, tt.flowID, tt.source, tt.destinations, tt.masterCallID, tt.masterGroupcallID, tt.ringMethod, tt.answerMethod)
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

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cmgroupcall.Groupcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("1717ba30-be34-11ed-87e7-5739c7ea8622"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1717ba30-be34-11ed-87e7-5739c7ea8622"}`),
			},

			"bin-manager.call-manager.request",
			&sock.Request{
				URI:    "/v1/groupcalls/1717ba30-be34-11ed-87e7-5739c7ea8622",
				Method: sock.RequestMethodGet,
			},
			&cmgroupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1717ba30-be34-11ed-87e7-5739c7ea8622"),
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
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cmgroupcall.Groupcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("06d9ec2a-be33-11ed-acc5-876b594da79c"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"06d9ec2a-be33-11ed-acc5-876b594da79c"}`),
			},

			"bin-manager.call-manager.request",
			&sock.Request{
				URI:    "/v1/groupcalls/06d9ec2a-be33-11ed-acc5-876b594da79c",
				Method: sock.RequestMethodDelete,
			},
			&cmgroupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("06d9ec2a-be33-11ed-acc5-876b594da79c"),
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
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cmgroupcall.Groupcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("3d82215c-be33-11ed-aed4-7b9daa884e9f"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3d82215c-be33-11ed-aed4-7b9daa884e9f"}`),
			},

			"bin-manager.call-manager.request",
			&sock.Request{
				URI:    "/v1/groupcalls/3d82215c-be33-11ed-aed4-7b9daa884e9f/hangup",
				Method: sock.RequestMethodPost,
			},
			&cmgroupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3d82215c-be33-11ed-aed4-7b9daa884e9f"),
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
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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

func Test_CallV1GroupcallUpdateAnswerGroupcallID(t *testing.T) {
	tests := []struct {
		name string

		groupcallID       uuid.UUID
		answerGroupcallID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cmgroupcall.Groupcall
	}{
		{
			name: "normal",

			groupcallID:       uuid.FromStringOrNil("e50ab9b0-e328-11ed-abe1-abcb6aaa1b47"),
			answerGroupcallID: uuid.FromStringOrNil("e5399dc0-e328-11ed-af77-6f6bf2e02462"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"e50ab9b0-e328-11ed-abe1-abcb6aaa1b47"}`),
			},

			expectTarget: "bin-manager.call-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/groupcalls/e50ab9b0-e328-11ed-abe1-abcb6aaa1b47/answer_groupcall_id",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"answer_groupcall_id":"e5399dc0-e328-11ed-af77-6f6bf2e02462"}`),
			},
			expectRes: &cmgroupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e50ab9b0-e328-11ed-abe1-abcb6aaa1b47"),
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

			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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

func Test_CallV1GroupcallHangupOthers(t *testing.T) {
	tests := []struct {
		name string

		groupcallID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
	}{
		{
			name: "normal",

			groupcallID: uuid.FromStringOrNil("5bde1332-e32b-11ed-8ed7-ef18f8cb924d"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"5bde1332-e32b-11ed-8ed7-ef18f8cb924d"}`),
			},

			expectTarget: "bin-manager.call-manager.request",
			expectRequest: &sock.Request{
				URI:    "/v1/groupcalls/5bde1332-e32b-11ed-8ed7-ef18f8cb924d/hangup_others",
				Method: sock.RequestMethodPost,
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

			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CallV1GroupcallHangupOthers(ctx, tt.groupcallID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallV1GroupcallHangupCall(t *testing.T) {
	tests := []struct {
		name string

		groupcallID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
	}{
		{
			name: "normal",

			groupcallID: uuid.FromStringOrNil("d810fc56-e337-11ed-bdbb-57b799f47676"),

			response: &sock.Response{
				StatusCode: 200,
			},

			expectTarget: "bin-manager.call-manager.request",
			expectRequest: &sock.Request{
				URI:    "/v1/groupcalls/d810fc56-e337-11ed-bdbb-57b799f47676/hangup_call",
				Method: sock.RequestMethodPost,
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

			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CallV1GroupcallHangupCall(ctx, tt.groupcallID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallV1GroupcallHangupGroupcall(t *testing.T) {
	tests := []struct {
		name string

		groupcallID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
	}{
		{
			name: "normal",

			groupcallID: uuid.FromStringOrNil("02fdbce2-e338-11ed-92cd-7f5633b75bad"),

			response: &sock.Response{
				StatusCode: 200,
			},

			expectTarget: "bin-manager.call-manager.request",
			expectRequest: &sock.Request{
				URI:    "/v1/groupcalls/02fdbce2-e338-11ed-92cd-7f5633b75bad/hangup_groupcall",
				Method: sock.RequestMethodPost,
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

			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CallV1GroupcallHangupGroupcall(ctx, tt.groupcallID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
