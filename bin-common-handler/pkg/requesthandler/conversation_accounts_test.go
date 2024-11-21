package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cvaccount "monorepo/bin-conversation-manager/models/account"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_ConversationV1AccountGet(t *testing.T) {

	tests := []struct {
		name string

		accountID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cvaccount.Account
	}{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("a6512c7e-003b-11ee-90ce-77b8ed60c6b0"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a6512c7e-003b-11ee-90ce-77b8ed60c6b0"}`),
			},

			expectTarget: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:    "/v1/accounts/a6512c7e-003b-11ee-90ce-77b8ed60c6b0",
				Method: sock.RequestMethodGet,
			},
			expectRes: &cvaccount.Account{
				ID: uuid.FromStringOrNil("a6512c7e-003b-11ee-90ce-77b8ed60c6b0"),
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

			cf, err := reqHandler.ConversationV1AccountGet(ctx, tt.accountID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, cf)
			}
		})
	}
}

func Test_ConversationV1AccountGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes []cvaccount.Account
	}{
		{
			name: "normal",

			pageToken: "2021-03-02 03:23:20.995000",
			pageSize:  10,
			filters: map[string]string{
				"deleted": "false",
			},

			expectURL:    "/v1/accounts?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10",
			expectTarget: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/accounts?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10&filter_deleted=false",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"78a9b0c8-003d-11ee-a05f-2bc10442c9f9"},{"id":"78d2ae88-003d-11ee-a2d7-574f3cd765cd"}]`),
			},

			expectRes: []cvaccount.Account{
				{
					ID: uuid.FromStringOrNil("78a9b0c8-003d-11ee-a05f-2bc10442c9f9"),
				},
				{
					ID: uuid.FromStringOrNil("78d2ae88-003d-11ee-a2d7-574f3cd765cd"),
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

			res, err := reqHandler.ConversationV1AccountGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationV1AccountCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		accountType cvaccount.Type
		accountName string
		detail      string
		secret      string
		token       string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *cvaccount.Account
	}{
		{
			name: "normal",

			customerID:  uuid.FromStringOrNil("2292b6c0-003e-11ee-9fb5-fff568769b60"),
			accountType: cvaccount.TypeLine,
			accountName: "test name",
			detail:      "test detail",
			secret:      "test secret",
			token:       "test token",

			expectTarget: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/accounts",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"2292b6c0-003e-11ee-9fb5-fff568769b60","type":"line","name":"test name","detail":"test detail","secret":"test secret","token":"test token"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"22c10b42-003e-11ee-9d2b-5fc3b9f2d82a"}`),
			},

			expectRes: &cvaccount.Account{
				ID: uuid.FromStringOrNil("22c10b42-003e-11ee-9d2b-5fc3b9f2d82a"),
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

			res, err := reqHandler.ConversationV1AccountCreate(ctx, tt.customerID, tt.accountType, tt.accountName, tt.detail, tt.secret, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationV1AccountUpdate(t *testing.T) {

	tests := []struct {
		name string

		id          uuid.UUID
		accountName string
		detail      string
		secret      string
		token       string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *cvaccount.Account
	}{
		{
			name: "normal",

			id:          uuid.FromStringOrNil("a3c2b754-003e-11ee-aa7e-e760c874d75f"),
			accountName: "test name",
			detail:      "test detail",
			secret:      "test secret",
			token:       "test token",

			expectTarget: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/accounts/a3c2b754-003e-11ee-aa7e-e760c874d75f",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"test name","detail":"test detail","secret":"test secret","token":"test token"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a3c2b754-003e-11ee-aa7e-e760c874d75f"}`),
			},

			expectRes: &cvaccount.Account{
				ID: uuid.FromStringOrNil("a3c2b754-003e-11ee-aa7e-e760c874d75f"),
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

			res, err := reqHandler.ConversationV1AccountUpdate(ctx, tt.id, tt.accountName, tt.detail, tt.secret, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationV1AccountDelete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *cvaccount.Account
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("cb700d10-003e-11ee-be73-4b361dcf2748"),

			expectTarget: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/accounts/cb700d10-003e-11ee-be73-4b361dcf2748",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeNone,
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"cb700d10-003e-11ee-be73-4b361dcf2748"}`),
			},

			expectRes: &cvaccount.Account{
				ID: uuid.FromStringOrNil("cb700d10-003e-11ee-be73-4b361dcf2748"),
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

			res, err := reqHandler.ConversationV1AccountDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
