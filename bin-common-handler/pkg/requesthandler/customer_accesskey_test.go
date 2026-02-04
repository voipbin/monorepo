package requesthandler

import (
	"context"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_CustomerV1AccesskeyList(t *testing.T) {

	tests := []struct {
		name string

		pageToken  string
		pageSize   uint64
		customerID uuid.UUID
		filters    map[csaccesskey.Field]any

		response *sock.Response

		expectedTarget  string
		expectedRequest *sock.Request

		expectedRes []csaccesskey.Accesskey
	}{
		{
			name: "normal",

			pageToken:  "2021-03-02T03:23:20.995000Z",
			pageSize:   10,
			customerID: uuid.FromStringOrNil("eee40e32-ab3b-11ef-8190-ab49eeb155a1"),
			filters: map[csaccesskey.Field]any{
				csaccesskey.FieldDeleted: false,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"db965d74-ab3d-11ef-a273-17bee7b8b789"},{"id":"dc5d389a-ab3d-11ef-bc7f-ab1929a9dc03"}]`),
			},

			expectedTarget: "bin-manager.customer-manager.request",
			expectedRequest: &sock.Request{
				URI:      "/v1/accesskeys?page_token=2021-03-02T03%3A23%3A20.995000Z&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"deleted":false}`),
			},
			expectedRes: []csaccesskey.Accesskey{
				{
					ID: uuid.FromStringOrNil("db965d74-ab3d-11ef-a273-17bee7b8b789"),
				},
				{
					ID: uuid.FromStringOrNil("dc5d389a-ab3d-11ef-bc7f-ab1929a9dc03"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectedTarget, tt.expectedRequest).Return(tt.response, nil)

			res, err := h.CustomerV1AccesskeyList(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_CustomerV1AccesskeyGet(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *csaccesskey.Accesskey
	}{
		{
			"normal",

			uuid.FromStringOrNil("2fc401a8-ab3e-11ef-af2e-3f81db3d8e6b"),

			"bin-manager.customer-manager.request",
			&sock.Request{
				URI:      "/v1/accesskeys/2fc401a8-ab3e-11ef-af2e-3f81db3d8e6b",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2fc401a8-ab3e-11ef-af2e-3f81db3d8e6b"}`),
			},
			&csaccesskey.Accesskey{
				ID: uuid.FromStringOrNil("2fc401a8-ab3e-11ef-af2e-3f81db3d8e6b"),
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

			res, err := reqHandler.CustomerV1AccesskeyGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerV1AccesskeyDelete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		response *sock.Response

		expectedTarget  string
		expectedRequest *sock.Request
		expectedRes     *csaccesskey.Accesskey
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("90e1dbd6-ab3e-11ef-b303-e30eded5a44f"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"90e1dbd6-ab3e-11ef-b303-e30eded5a44f"}`),
			},

			expectedTarget: "bin-manager.customer-manager.request",
			expectedRequest: &sock.Request{
				URI:      "/v1/accesskeys/90e1dbd6-ab3e-11ef-b303-e30eded5a44f",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},

			expectedRes: &csaccesskey.Accesskey{
				ID: uuid.FromStringOrNil("90e1dbd6-ab3e-11ef-b303-e30eded5a44f"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectedTarget, tt.expectedRequest).Return(tt.response, nil)

			res, err := reqHandler.CustomerV1AccesskeyDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_CustomerV1AccesskeyCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		accesskeyName string
		detail        string
		expire        int32

		response *sock.Response

		expectedTarget  string
		expectedRequest *sock.Request
		expectedRes     *csaccesskey.Accesskey
	}{
		{
			name: "normal",

			customerID:    uuid.FromStringOrNil("d6abc410-ab3e-11ef-8c3a-cf881e15d9a1"),
			accesskeyName: "test name",
			detail:        "test detail",
			expire:        86400000,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"47943ef0-cb2f-11ee-adbd-136bc293c7e1"}`),
			},

			expectedTarget: "bin-manager.customer-manager.request",
			expectedRequest: &sock.Request{
				URI:      "/v1/accesskeys",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"d6abc410-ab3e-11ef-8c3a-cf881e15d9a1","name":"test name","detail":"test detail","expire":86400000}`),
			},
			expectedRes: &csaccesskey.Accesskey{
				ID: uuid.FromStringOrNil("47943ef0-cb2f-11ee-adbd-136bc293c7e1"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectedTarget, tt.expectedRequest).Return(tt.response, nil)

			res, err := reqHandler.CustomerV1AccesskeyCreate(
				ctx,
				tt.customerID,
				tt.accesskeyName,
				tt.detail,
				tt.expire,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectedRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_CustomerV1AccesskeyUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		id       uuid.UUID
		userName string
		detail   string

		response        *sock.Response
		expectedTarget  string
		expectedRequest *sock.Request
		expectedRes     *csaccesskey.Accesskey
	}{
		{
			name: "normal",

			id:       uuid.FromStringOrNil("c43857f2-ab3f-11ef-a6ba-c7e6743d141e"),
			userName: "test name",
			detail:   "test detail",

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c43857f2-ab3f-11ef-a6ba-c7e6743d141e"}`),
			},

			expectedTarget: "bin-manager.customer-manager.request",
			expectedRequest: &sock.Request{
				URI:      "/v1/accesskeys/c43857f2-ab3f-11ef-a6ba-c7e6743d141e",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"test name","detail":"test detail"}`),
			},
			expectedRes: &csaccesskey.Accesskey{
				ID: uuid.FromStringOrNil("c43857f2-ab3f-11ef-a6ba-c7e6743d141e"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectedTarget, tt.expectedRequest).Return(tt.response, nil)

			res, err := reqHandler.CustomerV1AccesskeyUpdate(ctx, tt.id, tt.userName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectedRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedRes, res)
			}
		})
	}
}
