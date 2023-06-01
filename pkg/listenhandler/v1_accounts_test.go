package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/accounthandler"
)

func Test_processV1AccountsGet(t *testing.T) {

	tests := []struct {
		name string

		expectCustomerID uuid.UUID
		expectPageSize   uint64
		expectPageToken  string

		responseAccounts []*account.Account

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
	}{
		{
			name: "normal",

			expectCustomerID: uuid.FromStringOrNil("6af495b0-fecb-11ed-b59e-e70b3afff8a1"),
			expectPageSize:   10,
			expectPageToken:  "2021-03-01 03:30:17.000000",

			responseAccounts: []*account.Account{
				{
					ID: uuid.FromStringOrNil("645891fe-e863-11ec-b291-9f454e92f1bb"),
				},
			},

			request: &rabbitmqhandler.Request{
				URI:    "/v1/accounts?customer_id=6af495b0-fecb-11ed-b59e-e70b3afff8a1&page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"645891fe-e863-11ec-b291-9f454e92f1bb","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","secret":"","token":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			name: "2 results",

			expectCustomerID: uuid.FromStringOrNil("6b2efe9e-fecb-11ed-aa65-ff71705cd816"),
			expectPageSize:   10,
			expectPageToken:  "2021-03-01 03:30:17.000000",

			responseAccounts: []*account.Account{
				{
					ID: uuid.FromStringOrNil("6b5f9da6-fecb-11ed-a0ea-4fdabd236387"),
				},
				{
					ID: uuid.FromStringOrNil("6b906c9c-fecb-11ed-a341-f38426e7e737"),
				},
			},
			request: &rabbitmqhandler.Request{
				URI:    "/v1/accounts?customer_id=6b2efe9e-fecb-11ed-aa65-ff71705cd816&page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"6b5f9da6-fecb-11ed-a0ea-4fdabd236387","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","secret":"","token":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"6b906c9c-fecb-11ed-a341-f38426e7e737","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","secret":"","token":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				rabbitSock:     mockSock,
				accountHandler: mockAccount,
			}

			mockAccount.EXPECT().GetsByCustomerID(gomock.Any(), tt.expectCustomerID, tt.expectPageToken, tt.expectPageSize).Return(tt.responseAccounts, nil)
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

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
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
				ID: uuid.FromStringOrNil("459c19ae-fecc-11ed-9558-fbf54c7aa51e"),
			},

			request: &rabbitmqhandler.Request{
				URI:      "/v1/accounts",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"456609ea-fecc-11ed-a717-5f6984c51794","type":"line","name":"test name","detail":"test detail","secret":"test secret","token":"test token"}`),
			},
			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"459c19ae-fecc-11ed-9558-fbf54c7aa51e","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","secret":"","token":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				rabbitSock:     mockSock,
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

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
	}{
		{
			name: "normal",

			expectID: uuid.FromStringOrNil("1793ed06-fecd-11ed-ab65-07ce8687961d"),

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("1793ed06-fecd-11ed-ab65-07ce8687961d"),
			},

			request: &rabbitmqhandler.Request{
				URI:    "/v1/accounts/1793ed06-fecd-11ed-ab65-07ce8687961d",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1793ed06-fecd-11ed-ab65-07ce8687961d","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","secret":"","token":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				rabbitSock:     mockSock,
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

		expectID     uuid.UUID
		expectName   string
		expectDetail string
		expectSecret string
		expectToken  string

		responseAccount *account.Account

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
	}{
		{
			name: "normal",

			expectID:     uuid.FromStringOrNil("17c1c726-fecd-11ed-8139-dff04db7fa05"),
			expectName:   "test name",
			expectDetail: "test detail",
			expectSecret: "test secret",
			expectToken:  "test token",

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("17c1c726-fecd-11ed-8139-dff04db7fa05"),
			},

			request: &rabbitmqhandler.Request{
				URI:      "/v1/accounts/17c1c726-fecd-11ed-8139-dff04db7fa05",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"test name","detail":"test detail","secret":"test secret","token":"test token"}`),
			},
			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"17c1c726-fecd-11ed-8139-dff04db7fa05","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","secret":"","token":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				rabbitSock:     mockSock,
				accountHandler: mockAccount,
			}

			mockAccount.EXPECT().Update(gomock.Any(), tt.expectID, tt.expectName, tt.expectDetail, tt.expectSecret, tt.expectToken).Return(tt.responseAccount, nil)
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

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
	}{
		{
			name: "normal",

			expectID: uuid.FromStringOrNil("17eeb786-fecd-11ed-8113-5f6f4693c29f"),

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("17eeb786-fecd-11ed-8113-5f6f4693c29f"),
			},

			request: &rabbitmqhandler.Request{
				URI:    "/v1/accounts/17eeb786-fecd-11ed-8113-5f6f4693c29f",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"17eeb786-fecd-11ed-8113-5f6f4693c29f","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","secret":"","token":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				rabbitSock:     mockSock,
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
