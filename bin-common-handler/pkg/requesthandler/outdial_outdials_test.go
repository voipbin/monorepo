package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	omoutdial "monorepo/bin-outdial-manager/models/outdial"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

func Test_OutdialV1OutdialCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		campaignID  uuid.UUID
		outdialName string
		detail      string
		data        string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		response *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("440529e4-b64f-11ec-9208-ef03201c2688"),
			uuid.FromStringOrNil("476f3e62-b64f-11ec-a49c-bb96b9d8a42d"),
			"test name",
			"test detail",
			"test data",

			"bin-manager.outdial-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/outdials",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"440529e4-b64f-11ec-9208-ef03201c2688","campaign_id":"476f3e62-b64f-11ec-a49c-bb96b9d8a42d","name":"test name","detail":"test detail","data":"test data"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{}`),
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

			_, err := reqHandler.OutdialV1OutdialCreate(ctx, tt.customerID, tt.campaignID, tt.outdialName, tt.detail, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_OutdialV1OutdialGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult []omoutdial.Outdial
	}{
		{
			"normal",

			uuid.FromStringOrNil("74b94c72-b650-11ec-a5cf-ff01639e276f"),
			"2021-03-02 03:23:20.995000",
			10,

			"bin-manager.outdial-manager.request",
			&rabbitmqhandler.Request{
				URI:      fmt.Sprintf("/v1/outdials?page_token=%s&page_size=10&customer_id=74b94c72-b650-11ec-a5cf-ff01639e276f", url.QueryEscape("2021-03-02 03:23:20.995000")),
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"74e6ced6-b650-11ec-ac88-8f6fc68ead7d"}]`),
			},
			[]omoutdial.Outdial{
				{
					ID: uuid.FromStringOrNil("74e6ced6-b650-11ec-ac88-8f6fc68ead7d"),
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

			res, err := reqHandler.OutdialV1OutdialGetsByCustomerID(ctx, tt.customerID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_OutdialV1OutdialGet(t *testing.T) {
	tests := []struct {
		name string

		outdialID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult *omoutdial.Outdial
	}{
		{
			"normal",

			uuid.FromStringOrNil("f3ca7112-b650-11ec-aad7-17a0858dadcf"),

			"bin-manager.outdial-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/outdials/f3ca7112-b650-11ec-aad7-17a0858dadcf",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f3ca7112-b650-11ec-aad7-17a0858dadcf"}`),
			},
			&omoutdial.Outdial{
				ID: uuid.FromStringOrNil("f3ca7112-b650-11ec-aad7-17a0858dadcf"),
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

			res, err := reqHandler.OutdialV1OutdialGet(ctx, tt.outdialID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_OutdialV1OutdialDelete(t *testing.T) {

	tests := []struct {
		name string

		outdialID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult *omoutdial.Outdial
	}{
		{
			"normal",

			uuid.FromStringOrNil("d21d8788-b651-11ec-8ada-53590bef4cc1"),

			"bin-manager.outdial-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/outdials/d21d8788-b651-11ec-8ada-53590bef4cc1",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d21d8788-b651-11ec-8ada-53590bef4cc1"}`),
			},
			&omoutdial.Outdial{
				ID: uuid.FromStringOrNil("d21d8788-b651-11ec-8ada-53590bef4cc1"),
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

			res, err := reqHandler.OutdialV1OutdialDelete(ctx, tt.outdialID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_OutdialV1OutdialUpdateBasicInfo(t *testing.T) {
	tests := []struct {
		name string

		id          uuid.UUID
		outdialName string
		detail      string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult *omoutdial.Outdial
	}{
		{
			"normal",

			uuid.FromStringOrNil("e2fd1a18-b652-11ec-9823-efc193bad6bf"),
			"test name",
			"test detail",

			"bin-manager.outdial-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/outdials/e2fd1a18-b652-11ec-9823-efc193bad6bf",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"test name","detail":"test detail"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e2fd1a18-b652-11ec-9823-efc193bad6bf"}`),
			},
			&omoutdial.Outdial{
				ID: uuid.FromStringOrNil("e2fd1a18-b652-11ec-9823-efc193bad6bf"),
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

			res, err := reqHandler.OutdialV1OutdialUpdateBasicInfo(ctx, tt.id, tt.outdialName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_OutdialV1OutdialUpdateCampaignID(t *testing.T) {
	tests := []struct {
		name string

		id         uuid.UUID
		campaignID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult *omoutdial.Outdial
	}{
		{
			"normal",

			uuid.FromStringOrNil("127cef34-b653-11ec-94f1-bf920056048a"),
			uuid.FromStringOrNil("129e1312-b653-11ec-b265-4b6204d435a5"),

			"bin-manager.outdial-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/outdials/127cef34-b653-11ec-94f1-bf920056048a/campaign_id",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"campaign_id":"129e1312-b653-11ec-b265-4b6204d435a5"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"127cef34-b653-11ec-94f1-bf920056048a"}`),
			},
			&omoutdial.Outdial{
				ID: uuid.FromStringOrNil("127cef34-b653-11ec-94f1-bf920056048a"),
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

			res, err := reqHandler.OutdialV1OutdialUpdateCampaignID(ctx, tt.id, tt.campaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_OutdialV1OutdialUpdateData(t *testing.T) {
	tests := []struct {
		name string

		id   uuid.UUID
		data string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult *omoutdial.Outdial
	}{
		{
			"normal",

			uuid.FromStringOrNil("5d0e4cf0-b653-11ec-8e87-af57dbb5c13a"),
			"test data",

			"bin-manager.outdial-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/outdials/5d0e4cf0-b653-11ec-8e87-af57dbb5c13a/data",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"data":"test data"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5d0e4cf0-b653-11ec-8e87-af57dbb5c13a"}`),
			},
			&omoutdial.Outdial{
				ID: uuid.FromStringOrNil("5d0e4cf0-b653-11ec-8e87-af57dbb5c13a"),
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

			res, err := reqHandler.OutdialV1OutdialUpdateData(ctx, tt.id, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}
