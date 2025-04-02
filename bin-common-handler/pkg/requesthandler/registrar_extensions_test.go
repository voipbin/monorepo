package requesthandler

import (
	"context"
	reflect "reflect"
	"testing"

	rmextension "monorepo/bin-registrar-manager/models/extension"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_RegistrarExtensionCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		ext           string
		password      string
		extensionName string
		detail        string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *rmextension.Extension
	}{
		{
			"normal",

			uuid.FromStringOrNil("324cf776-7ff0-11ec-a0ea-e30825a4224f"),
			"4c98b74a-6f9e-11eb-a82f-37575ab16881",
			"53710356-6f9e-11eb-8a91-43345d98682a",
			"test name",
			"test detail",

			"bin-manager.registrar-manager.request",
			&sock.Request{
				URI:      "/v1/extensions",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"324cf776-7ff0-11ec-a0ea-e30825a4224f","extension":"4c98b74a-6f9e-11eb-a82f-37575ab16881","password":"53710356-6f9e-11eb-8a91-43345d98682a","domain_id":"00000000-0000-0000-0000-000000000000","name":"test name","detail":"test detail"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"121dd178-5712-11ee-b6b3-4b0ab7784e17","customer_id":"324cf776-7ff0-11ec-a0ea-e30825a4224f","domain_id":"","extension":"4c98b74a-6f9e-11eb-a82f-37575ab16881","password":"53710356-6f9e-11eb-8a91-43345d98682a","name":"test name","detail":"test detail"}`),
			},
			&rmextension.Extension{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("121dd178-5712-11ee-b6b3-4b0ab7784e17"),
					CustomerID: uuid.FromStringOrNil("324cf776-7ff0-11ec-a0ea-e30825a4224f"),
				},
				Name:      "test name",
				Detail:    "test detail",
				Extension: "4c98b74a-6f9e-11eb-a82f-37575ab16881",
				Password:  "53710356-6f9e-11eb-8a91-43345d98682a",
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

			res, err := reqHandler.RegistrarV1ExtensionCreate(ctx, tt.customerID, tt.ext, tt.password, tt.extensionName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_RegistrarExtensionUpdate(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		extensionName string
		detail        string
		password      string

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *rmextension.Extension
	}{
		{
			"normal",

			uuid.FromStringOrNil("0be5298a-6f9f-11eb-bb77-f71f5b5f95f7"),
			"update name",
			"update detail",
			"update password",

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0be5298a-6f9f-11eb-bb77-f71f5b5f95f7","customer_id":"324cf776-7ff0-11ec-a0ea-e30825a4224f","name":"update name","detail":"update detail","password":"update password"}`),
			},
			"bin-manager.registrar-manager.request",
			&sock.Request{
				URI:      "/v1/extensions/0be5298a-6f9f-11eb-bb77-f71f5b5f95f7",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"update name","detail":"update detail","password":"update password"}`),
			},
			&rmextension.Extension{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("0be5298a-6f9f-11eb-bb77-f71f5b5f95f7"),
					CustomerID: uuid.FromStringOrNil("324cf776-7ff0-11ec-a0ea-e30825a4224f"),
				},
				Name:     "update name",
				Detail:   "update detail",
				Password: "update password",
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

func Test_RegistrarV1ExtensionGet(t *testing.T) {

	tests := []struct {
		name string

		extensionID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *rmextension.Extension
	}{
		{
			"normal",

			uuid.FromStringOrNil("342f9734-6fa1-11eb-a937-17d537105d6a"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"342f9734-6fa1-11eb-a937-17d537105d6a","customer_id":"324cf776-7ff0-11ec-a0ea-e30825a4224f","domain_id":"4351e596-6fa1-11eb-b086-db7f03792b30","name":"test domain","detail":"test domain detail","extension":"test","password":"password","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},

			"bin-manager.registrar-manager.request",
			&sock.Request{
				URI:      "/v1/extensions/342f9734-6fa1-11eb-a937-17d537105d6a",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&rmextension.Extension{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("342f9734-6fa1-11eb-a937-17d537105d6a"),
					CustomerID: uuid.FromStringOrNil("324cf776-7ff0-11ec-a0ea-e30825a4224f"),
				},
				Extension: "test",
				Password:  "password",
				Name:      "test domain",
				Detail:    "test domain detail",
				TMCreate:  "2020-09-20 03:23:20.995000",
				TMUpdate:  "",
				TMDelete:  "",
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

func Test_RegistrarExtensionDelete(t *testing.T) {

	tests := []struct {
		name string

		extesnionID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *rmextension.Extension
	}{
		{
			"normal",

			uuid.FromStringOrNil("b2ca6024-6fa1-11eb-aa5a-738c234d2ee1"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b2ca6024-6fa1-11eb-aa5a-738c234d2ee1"}`),
			},

			"bin-manager.registrar-manager.request",
			&sock.Request{
				URI:      "/v1/extensions/b2ca6024-6fa1-11eb-aa5a-738c234d2ee1",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&rmextension.Extension{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("b2ca6024-6fa1-11eb-aa5a-738c234d2ee1"),
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

func Test_RegistrarV1ExtensionGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		response *sock.Response

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		expectRes     []rmextension.Extension
	}{
		{
			"normal",

			"2020-09-20 03:23:20.995000",
			10,
			map[string]string{
				"customer_id": "f18dcabe-4ff3-11ee-80be-875a8c6041d4",
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"f1d6686e-4ff3-11ee-a1cc-cbb904dc2d7e"}]`),
			},

			"/v1/extensions?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10",
			"bin-manager.registrar-manager.request",
			&sock.Request{
				URI:      "/v1/extensions?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10&filter_customer_id=f18dcabe-4ff3-11ee-80be-875a8c6041d4",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			[]rmextension.Extension{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("f1d6686e-4ff3-11ee-a1cc-cbb904dc2d7e"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			reqHandler := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}

			ctx := context.Background()
			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RegistrarV1ExtensionGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_RegistrarV1ExtensionGetsByExtension(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		extension  string

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *rmextension.Extension
	}{
		{
			"normal",

			uuid.FromStringOrNil("5703f08a-5710-11ee-9295-77eb098ad269"),
			"test-exten",

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d19c3956-6ed8-11eb-b971-fb12bc338aeb","customer_id":"5703f08a-5710-11ee-9295-77eb098ad269","domain_id":"e45dafce-6fa1-11eb-9e87-7ba8b7ae10f0","name":"test","detail":"test detail","extension":"test","password":"password","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},

			"bin-manager.registrar-manager.request",
			&sock.Request{
				URI:    "/v1/extensions/extension/test-exten?customer_id=5703f08a-5710-11ee-9295-77eb098ad269",
				Method: sock.RequestMethodGet,
			},
			&rmextension.Extension{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("d19c3956-6ed8-11eb-b971-fb12bc338aeb"),
					CustomerID: uuid.FromStringOrNil("5703f08a-5710-11ee-9295-77eb098ad269"),
				},
				Name:      "test",
				Detail:    "test detail",
				Extension: "test",
				Password:  "password",
				TMCreate:  "2020-09-20 03:23:20.995000",
				TMUpdate:  "",
				TMDelete:  "",
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

			res, err := reqHandler.RegistrarV1ExtensionGetByExtension(ctx, tt.customerID, tt.extension)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
