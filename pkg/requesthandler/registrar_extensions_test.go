package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	rmextension "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_RMExtensionCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		ext           string
		password      string
		domainID      uuid.UUID
		extensionName string
		detail        string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *rmextension.Extension
	}{
		{
			"normal",

			uuid.FromStringOrNil("324cf776-7ff0-11ec-a0ea-e30825a4224f"),
			"4c98b74a-6f9e-11eb-a82f-37575ab16881",
			"53710356-6f9e-11eb-8a91-43345d98682a",
			uuid.FromStringOrNil("22de2e58-6f9e-11eb-8fee-ef16005005d7"),
			"test name",
			"test detail",

			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/extensions",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"324cf776-7ff0-11ec-a0ea-e30825a4224f","domain_id":"22de2e58-6f9e-11eb-8fee-ef16005005d7","extension":"4c98b74a-6f9e-11eb-a82f-37575ab16881","password":"53710356-6f9e-11eb-8a91-43345d98682a","name":"test name","detail":"test detail"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"68040bf2-6ed5-11eb-9924-9febe8425cbe","customer_id":"324cf776-7ff0-11ec-a0ea-e30825a4224f","domain_id":"22de2e58-6f9e-11eb-8fee-ef16005005d7","name":"test name","detail":"test detail","extension":"4c98b74a-6f9e-11eb-a82f-37575ab16881","password":"53710356-6f9e-11eb-8a91-43345d98682a","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},
			&rmextension.Extension{
				ID:         uuid.FromStringOrNil("68040bf2-6ed5-11eb-9924-9febe8425cbe"),
				CustomerID: uuid.FromStringOrNil("324cf776-7ff0-11ec-a0ea-e30825a4224f"),
				Name:       "test name",
				Detail:     "test detail",
				DomainID:   uuid.FromStringOrNil("22de2e58-6f9e-11eb-8fee-ef16005005d7"),
				Extension:  "4c98b74a-6f9e-11eb-a82f-37575ab16881",
				Password:   "53710356-6f9e-11eb-8a91-43345d98682a",
				TMCreate:   "2020-09-20 03:23:20.995000",
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

			res, err := reqHandler.RegistrarV1ExtensionCreate(ctx, tt.customerID, tt.ext, tt.password, tt.domainID, tt.extensionName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_RMExtensionUpdate(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		extensionName string
		detail        string
		password      string

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *rmextension.Extension
	}{
		{
			"normal",

			uuid.FromStringOrNil("0be5298a-6f9f-11eb-bb77-f71f5b5f95f7"),
			"update name",
			"update detail",
			"update password",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0be5298a-6f9f-11eb-bb77-f71f5b5f95f7","customer_id":"324cf776-7ff0-11ec-a0ea-e30825a4224f","name":"update name","detail":"update detail","password":"update password"}`),
			},
			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/extensions/0be5298a-6f9f-11eb-bb77-f71f5b5f95f7",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"update name","detail":"update detail","password":"update password"}`),
			},
			&rmextension.Extension{
				ID:         uuid.FromStringOrNil("0be5298a-6f9f-11eb-bb77-f71f5b5f95f7"),
				CustomerID: uuid.FromStringOrNil("324cf776-7ff0-11ec-a0ea-e30825a4224f"),
				Name:       "update name",
				Detail:     "update detail",
				Password:   "update password",
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

			res, err := reqHandler.RegistrarV1ExtensionUpdate(ctx, tt.id, tt.extensionName, tt.detail, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_RMExtensionGet(t *testing.T) {

	tests := []struct {
		name string

		extensionID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *rmextension.Extension
	}{
		{
			"normal",

			uuid.FromStringOrNil("342f9734-6fa1-11eb-a937-17d537105d6a"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"342f9734-6fa1-11eb-a937-17d537105d6a","customer_id":"324cf776-7ff0-11ec-a0ea-e30825a4224f","domain_id":"4351e596-6fa1-11eb-b086-db7f03792b30","name":"test domain","detail":"test domain detail","extension":"test","password":"password","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},

			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/extensions/342f9734-6fa1-11eb-a937-17d537105d6a",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&rmextension.Extension{
				ID:         uuid.FromStringOrNil("342f9734-6fa1-11eb-a937-17d537105d6a"),
				CustomerID: uuid.FromStringOrNil("324cf776-7ff0-11ec-a0ea-e30825a4224f"),
				DomainID:   uuid.FromStringOrNil("4351e596-6fa1-11eb-b086-db7f03792b30"),
				Extension:  "test",
				Password:   "password",
				Name:       "test domain",
				Detail:     "test domain detail",
				TMCreate:   "2020-09-20 03:23:20.995000",
				TMUpdate:   "",
				TMDelete:   "",
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

			res, err := reqHandler.RegistrarV1ExtensionGet(ctx, tt.extensionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_RMExtensionDelete(t *testing.T) {

	tests := []struct {
		name string

		extesnionID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *rmextension.Extension
	}{
		{
			"normal",

			uuid.FromStringOrNil("b2ca6024-6fa1-11eb-aa5a-738c234d2ee1"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b2ca6024-6fa1-11eb-aa5a-738c234d2ee1"}`),
			},

			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/extensions/b2ca6024-6fa1-11eb-aa5a-738c234d2ee1",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&rmextension.Extension{
				ID: uuid.FromStringOrNil("b2ca6024-6fa1-11eb-aa5a-738c234d2ee1"),
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

			res, err := reqHandler.RegistrarV1ExtensionDelete(ctx, tt.extesnionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_RMExtensionsGets(t *testing.T) {

	tests := []struct {
		name string

		domainID  uuid.UUID
		pageToken string
		pageSize  uint64

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     []rmextension.Extension
	}{
		{
			"normal",

			uuid.FromStringOrNil("e45dafce-6fa1-11eb-9e87-7ba8b7ae10f0"),
			"2020-09-20 03:23:20.995000",
			10,

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d19c3956-6ed8-11eb-b971-fb12bc338aeb","customer_id":"324cf776-7ff0-11ec-a0ea-e30825a4224f","domain_id":"e45dafce-6fa1-11eb-9e87-7ba8b7ae10f0","name":"test","detail":"test detail","extension":"test","password":"password","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}]`),
			},

			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      fmt.Sprintf("/v1/extensions?page_token=%s&page_size=10&domain_id=e45dafce-6fa1-11eb-9e87-7ba8b7ae10f0", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			[]rmextension.Extension{
				{
					ID:         uuid.FromStringOrNil("d19c3956-6ed8-11eb-b971-fb12bc338aeb"),
					CustomerID: uuid.FromStringOrNil("324cf776-7ff0-11ec-a0ea-e30825a4224f"),
					DomainID:   uuid.FromStringOrNil("e45dafce-6fa1-11eb-9e87-7ba8b7ae10f0"),
					Name:       "test",
					Detail:     "test detail",
					Extension:  "test",
					Password:   "password",
					TMCreate:   "2020-09-20 03:23:20.995000",
					TMUpdate:   "",
					TMDelete:   "",
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

			res, err := reqHandler.RegistrarV1ExtensionGets(ctx, tt.domainID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
