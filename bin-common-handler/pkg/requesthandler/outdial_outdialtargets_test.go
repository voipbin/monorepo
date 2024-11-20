package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	omoutdialtarget "monorepo/bin-outdial-manager/models/outdialtarget"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_OutdialV1OutdialtargetCreate(t *testing.T) {

	tests := []struct {
		name string

		outdialID         uuid.UUID
		outdialtargetName string
		detail            string
		data              string

		destination0 *address.Address
		destination1 *address.Address
		destination2 *address.Address
		destination3 *address.Address
		destination4 *address.Address

		expectTarget  string
		expectRequest *sock.Request

		response *sock.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("05378520-b656-11ec-b5e4-bb71e495d2b6"),
			"test name",
			"test detail",
			"test data",

			&address.Address{
				Type:   address.TypeTel,
				Target: "+821100000001",
			},
			&address.Address{
				Type:   address.TypeTel,
				Target: "+821100000002",
			},
			&address.Address{
				Type:   address.TypeTel,
				Target: "+821100000003",
			},
			&address.Address{
				Type:   address.TypeTel,
				Target: "+821100000004",
			},
			&address.Address{
				Type:   address.TypeTel,
				Target: "+821100000005",
			},

			"bin-manager.outdial-manager.request",
			&sock.Request{
				URI:      "/v1/outdials/05378520-b656-11ec-b5e4-bb71e495d2b6/targets",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"test name","detail":"test detail","data":"test data","destination_0":{"type":"tel","target":"+821100000001","target_name":"","name":"","detail":""},"destination_1":{"type":"tel","target":"+821100000002","target_name":"","name":"","detail":""},"destination_2":{"type":"tel","target":"+821100000003","target_name":"","name":"","detail":""},"destination_3":{"type":"tel","target":"+821100000004","target_name":"","name":"","detail":""},"destination_4":{"type":"tel","target":"+821100000005","target_name":"","name":"","detail":""}}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"86e32246-b656-11ec-b2f8-f7db504bdc2e"}`),
			},
		},
		{
			"3 addresses",

			uuid.FromStringOrNil("05378520-b656-11ec-b5e4-bb71e495d2b6"),
			"test name",
			"test detail",
			"test data",

			&address.Address{
				Type:   address.TypeTel,
				Target: "+821100000001",
			},
			&address.Address{
				Type:   address.TypeTel,
				Target: "+821100000002",
			},
			nil,
			nil,
			&address.Address{
				Type:   address.TypeTel,
				Target: "+821100000005",
			},

			"bin-manager.outdial-manager.request",
			&sock.Request{
				URI:      "/v1/outdials/05378520-b656-11ec-b5e4-bb71e495d2b6/targets",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"test name","detail":"test detail","data":"test data","destination_0":{"type":"tel","target":"+821100000001","target_name":"","name":"","detail":""},"destination_1":{"type":"tel","target":"+821100000002","target_name":"","name":"","detail":""},"destination_4":{"type":"tel","target":"+821100000005","target_name":"","name":"","detail":""}}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"86e32246-b656-11ec-b2f8-f7db504bdc2e"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			_, err := reqHandler.OutdialV1OutdialtargetCreate(ctx, tt.outdialID, tt.outdialtargetName, tt.detail, tt.data, tt.destination0, tt.destination1, tt.destination2, tt.destination3, tt.destination4)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_OutdialV1OutdialtargetGetsAvailable(t *testing.T) {

	tests := []struct {
		name string

		outdialID uuid.UUID
		tryCount0 int
		tryCount1 int
		tryCount2 int
		tryCount3 int
		tryCount4 int
		limit     int

		expectTarget  string
		expectRequest *sock.Request

		response *sock.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("730b3636-b657-11ec-ae8e-3fc6ae86d1ec"),
			2,
			2,
			2,
			2,
			2,
			1,

			"bin-manager.outdial-manager.request",
			&sock.Request{
				URI:      "/v1/outdials/730b3636-b657-11ec-ae8e-3fc6ae86d1ec/available?try_count_0=2&try_count_1=2&try_count_2=2&try_count_3=2&try_count_4=2&limit=1",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"11f4a94e-b658-11ec-a89f-53f8b09275ce"}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			_, err := reqHandler.OutdialV1OutdialtargetGetsAvailable(ctx, tt.outdialID, tt.tryCount0, tt.tryCount1, tt.tryCount2, tt.tryCount3, tt.tryCount4, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_OutdialV1OutdialtargetDelete(t *testing.T) {

	tests := []struct {
		name string

		outdialtargetID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectResult *omoutdialtarget.OutdialTarget
	}{
		{
			"normal",

			uuid.FromStringOrNil("53ca0620-b658-11ec-99ca-7fe26b40d142"),

			"bin-manager.outdial-manager.request",
			&sock.Request{
				URI:      "/v1/outdialtargets/53ca0620-b658-11ec-99ca-7fe26b40d142",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"53ca0620-b658-11ec-99ca-7fe26b40d142"}`),
			},
			&omoutdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("53ca0620-b658-11ec-99ca-7fe26b40d142"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.OutdialV1OutdialtargetDelete(ctx, tt.outdialtargetID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_OutdialV1OutdialtargetGetsByOutdialID(t *testing.T) {

	tests := []struct {
		name string

		outdialID uuid.UUID
		pageToken string
		pageSize  uint64

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectResult []omoutdialtarget.OutdialTarget
	}{
		{
			"normal",

			uuid.FromStringOrNil("835e7280-c78e-11ec-9d4c-871c179d2bd9"),
			"2021-03-02 03:23:20.995000",
			10,

			"bin-manager.outdial-manager.request",
			&sock.Request{
				URI:      fmt.Sprintf("/v1/outdials/835e7280-c78e-11ec-9d4c-871c179d2bd9/targets?page_token=%s&page_size=10", url.QueryEscape("2021-03-02 03:23:20.995000")),
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"957b59ec-c78e-11ec-9d18-0b17e7b3a2ed"}]`),
			},
			[]omoutdialtarget.OutdialTarget{
				{
					ID: uuid.FromStringOrNil("957b59ec-c78e-11ec-9d18-0b17e7b3a2ed"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.OutdialV1OutdialtargetGetsByOutdialID(ctx, tt.outdialID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_OutdialV1OutdialtargetGet(t *testing.T) {

	tests := []struct {
		name string

		outdialtargetID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectResult *omoutdialtarget.OutdialTarget
	}{
		{
			"normal",

			uuid.FromStringOrNil("7ca49c9a-b658-11ec-839e-4732258c6c84"),

			"bin-manager.outdial-manager.request",
			&sock.Request{
				URI:      "/v1/outdialtargets/7ca49c9a-b658-11ec-839e-4732258c6c84",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"53ca0620-b658-11ec-99ca-7fe26b40d142"}`),
			},
			&omoutdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("53ca0620-b658-11ec-99ca-7fe26b40d142"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.OutdialV1OutdialtargetGet(ctx, tt.outdialtargetID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_OutdialV1OutdialtargetUpdateStatusProgressing(t *testing.T) {

	tests := []struct {
		name string

		outdialtargetID  uuid.UUID
		destinationIndex int

		expectTarget  string
		expectRequest *sock.Request

		response *sock.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("a843a03a-b658-11ec-b4a2-6f6326f02f30"),
			1,

			"bin-manager.outdial-manager.request",
			&sock.Request{
				URI:      "/v1/outdialtargets/a843a03a-b658-11ec-b4a2-6f6326f02f30/progressing",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"destination_index":1}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a843a03a-b658-11ec-b4a2-6f6326f02f30"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			_, err := reqHandler.OutdialV1OutdialtargetUpdateStatusProgressing(ctx, tt.outdialtargetID, tt.destinationIndex)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_OutdialV1OutdialtargetUpdateStatus(t *testing.T) {

	tests := []struct {
		name string

		outdialtargetID uuid.UUID
		status          omoutdialtarget.Status

		expectTarget  string
		expectRequest *sock.Request

		response *sock.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("16b4b4c2-b65f-11ec-92be-dba3f52ebb01"),
			omoutdialtarget.StatusIdle,

			"bin-manager.outdial-manager.request",
			&sock.Request{
				URI:      "/v1/outdialtargets/16b4b4c2-b65f-11ec-92be-dba3f52ebb01/status",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"status":"idle"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"16b4b4c2-b65f-11ec-92be-dba3f52ebb01"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			_, err := reqHandler.OutdialV1OutdialtargetUpdateStatus(ctx, tt.outdialtargetID, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
