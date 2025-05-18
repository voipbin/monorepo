package listenhandler

import (
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
)

func Test_processV1AccountsGet(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		expectPageSize  uint64
		expectPageToken string
		expectFields    map[account.Field]any

		responseAccounts []*account.Account

		response *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/accounts?page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000",
				Method:   sock.RequestMethodGet,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":"6af495b0-fecb-11ed-b59e-e70b3afff8a1","deleted":false}`),
			},

			expectPageSize:  10,
			expectPageToken: "2021-03-01 03:30:17.000000",
			expectFields: map[account.Field]any{
				account.FieldCustomerID: uuid.FromStringOrNil("6af495b0-fecb-11ed-b59e-e70b3afff8a1"),
				account.FieldDeleted:    false,
			},

			responseAccounts: []*account.Account{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("645891fe-e863-11ec-b291-9f454e92f1bb"),
					},
				},
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"645891fe-e863-11ec-b291-9f454e92f1bb","customer_id":"00000000-0000-0000-0000-000000000000"}]`),
			},
		},
		{
			name: "2 results",

			request: &sock.Request{
				URI:      "/v1/accounts?page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000",
				Method:   sock.RequestMethodGet,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":"6b2efe9e-fecb-11ed-aa65-ff71705cd816","deleted":false}`),
			},

			expectPageSize:  10,
			expectPageToken: "2021-03-01 03:30:17.000000",
			expectFields: map[account.Field]any{
				account.FieldCustomerID: uuid.FromStringOrNil("6b2efe9e-fecb-11ed-aa65-ff71705cd816"),
				account.FieldDeleted:    false,
			},

			responseAccounts: []*account.Account{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6b5f9da6-fecb-11ed-a0ea-4fdabd236387"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6b906c9c-fecb-11ed-a341-f38426e7e737"),
					},
				},
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"6b5f9da6-fecb-11ed-a0ea-4fdabd236387","customer_id":"00000000-0000-0000-0000-000000000000"},{"id":"6b906c9c-fecb-11ed-a341-f38426e7e737","customer_id":"00000000-0000-0000-0000-000000000000"}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				utilHandler:    mockUtil,
				accountHandler: mockAccount,
			}

			mockAccount.EXPECT().Gets(gomock.Any(), tt.expectPageToken, tt.expectPageSize, tt.expectFields).Return(tt.responseAccounts, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}
		})
	}
}

func Test_processV1AccountsPost(t *testing.T) {

	tests := []struct {
		name string

		expectCustomerID uuid.UUID
		expectType       account.Type
		expectName       string
		expectDetail     string
		expectSecret     string
		expectToken      string

		responseAccount *account.Account

		request  *sock.Request
		response *sock.Response
	}{
		{
			name: "normal",

			expectCustomerID: uuid.FromStringOrNil("456609ea-fecc-11ed-a717-5f6984c51794"),
			expectType:       account.TypeLine,
			expectName:       "test name",
			expectDetail:     "test detail",
			expectSecret:     "test secret",
			expectToken:      "test token",

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("459c19ae-fecc-11ed-9558-fbf54c7aa51e"),
				},
			},

			request: &sock.Request{
				URI:      "/v1/accounts",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"456609ea-fecc-11ed-a717-5f6984c51794","type":"line","name":"test name","detail":"test detail","secret":"test secret","token":"test token"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"459c19ae-fecc-11ed-9558-fbf54c7aa51e","customer_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				accountHandler: mockAccount,
			}

			mockAccount.EXPECT().Create(gomock.Any(), tt.expectCustomerID, tt.expectType, tt.expectName, tt.expectDetail, tt.expectSecret, tt.expectToken).Return(tt.responseAccount, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}
		})
	}
}

func Test_processV1AccountsIDGet(t *testing.T) {

	tests := []struct {
		name string

		expectID uuid.UUID

		responseAccount *account.Account

		request  *sock.Request
		response *sock.Response
	}{
		{
			name: "normal",

			expectID: uuid.FromStringOrNil("1793ed06-fecd-11ed-ab65-07ce8687961d"),

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1793ed06-fecd-11ed-ab65-07ce8687961d"),
				},
			},

			request: &sock.Request{
				URI:    "/v1/accounts/1793ed06-fecd-11ed-ab65-07ce8687961d",
				Method: sock.RequestMethodGet,
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1793ed06-fecd-11ed-ab65-07ce8687961d","customer_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				accountHandler: mockAccount,
			}

			mockAccount.EXPECT().Get(gomock.Any(), tt.expectID).Return(tt.responseAccount, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}
		})
	}
}

func Test_processV1AccountsIDPut(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		response        *sock.Response
		responseAccount *account.Account

		expectID       uuid.UUID
		expectedFields map[account.Field]any
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/accounts/17c1c726-fecd-11ed-8139-dff04db7fa05",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"test name","detail":"test detail","secret":"test secret","token":"test token"}`),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"17c1c726-fecd-11ed-8139-dff04db7fa05","customer_id":"00000000-0000-0000-0000-000000000000"}`),
			},
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("17c1c726-fecd-11ed-8139-dff04db7fa05"),
				},
			},

			expectID: uuid.FromStringOrNil("17c1c726-fecd-11ed-8139-dff04db7fa05"),
			expectedFields: map[account.Field]any{
				account.FieldName:   "test name",
				account.FieldDetail: "test detail",
				account.FieldSecret: "test secret",
				account.FieldToken:  "test token",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				accountHandler: mockAccount,
			}

			mockAccount.EXPECT().Update(gomock.Any(), tt.expectID, tt.expectedFields).Return(tt.responseAccount, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}
		})
	}
}

func Test_processV1AccountsIDDelete(t *testing.T) {

	tests := []struct {
		name string

		expectID uuid.UUID

		responseAccount *account.Account

		request  *sock.Request
		response *sock.Response
	}{
		{
			name: "normal",

			expectID: uuid.FromStringOrNil("17eeb786-fecd-11ed-8113-5f6f4693c29f"),

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("17eeb786-fecd-11ed-8113-5f6f4693c29f"),
				},
			},

			request: &sock.Request{
				URI:    "/v1/accounts/17eeb786-fecd-11ed-8113-5f6f4693c29f",
				Method: sock.RequestMethodDelete,
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"17eeb786-fecd-11ed-8113-5f6f4693c29f","customer_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				accountHandler: mockAccount,
			}

			mockAccount.EXPECT().Delete(gomock.Any(), tt.expectID).Return(tt.responseAccount, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}
		})
	}
}
