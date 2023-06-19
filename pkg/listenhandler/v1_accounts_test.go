package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/accounthandler"
)

func Test_processV1AccountsGet(t *testing.T) {

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		responseAccounts []*account.Account

		expectCustomerID uuid.UUID
		expectSize       uint64
		expectToken      string
		expectRes        *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &rabbitmqhandler.Request{
				URI:    "/v1/accounts?page_size=10&page_token=2023-06-08%2003:22:17.995000&customer_id=bc8f9070-0e5a-11ee-b22e-97ef303987a3",
				Method: rabbitmqhandler.RequestMethodGet,
			},

			responseAccounts: []*account.Account{
				{
					ID: uuid.FromStringOrNil("dafc10d0-0b97-11ee-af30-2fb7811295dd"),
				},
				{
					ID: uuid.FromStringOrNil("db4e7e24-0b97-11ee-91f9-c7d5620abcd7"),
				},
			},

			expectCustomerID: uuid.FromStringOrNil("bc8f9070-0e5a-11ee-b22e-97ef303987a3"),
			expectSize:       10,
			expectToken:      "2023-06-08 03:22:17.995000",
			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"dafc10d0-0b97-11ee-af30-2fb7811295dd","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","type":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"db4e7e24-0b97-11ee-91f9-c7d5620abcd7","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","type":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}]`),
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

			mockAccount.EXPECT().Gets(gomock.Any(), tt.expectCustomerID, tt.expectSize, tt.expectToken).Return(tt.responseAccounts, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1AccountsPost(t *testing.T) {

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		responseAccount *account.Account

		expectCustomerID uuid.UUID
		expectName       string
		expectDetail     string
		expectRes        *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &rabbitmqhandler.Request{
				URI:      "/v1/accounts",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"c28443b6-0e75-11ee-90ec-1bb28081d375","name":"test name","detail":"test detail"}`),
			},

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("c28443b6-0e75-11ee-90ec-1bb28081d375"),
			},

			expectCustomerID: uuid.FromStringOrNil("c28443b6-0e75-11ee-90ec-1bb28081d375"),
			expectName:       "test name",
			expectDetail:     "test detail",
			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c28443b6-0e75-11ee-90ec-1bb28081d375","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","type":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockAccount.EXPECT().Create(gomock.Any(), tt.expectCustomerID, tt.expectName, tt.expectDetail).Return(tt.responseAccount, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1AccountsIDGet(t *testing.T) {

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		responseAccount *account.Account

		expectID  uuid.UUID
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &rabbitmqhandler.Request{
				URI:    "/v1/accounts/922907b6-0942-11ee-960e-f31d2cc10daa",
				Method: rabbitmqhandler.RequestMethodGet,
			},

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("922907b6-0942-11ee-960e-f31d2cc10daa"),
			},

			expectID: uuid.FromStringOrNil("922907b6-0942-11ee-960e-f31d2cc10daa"),
			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"922907b6-0942-11ee-960e-f31d2cc10daa","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","type":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1AccountsCustomerIDIDGet(t *testing.T) {

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		responseAccount *account.Account

		expectCustomerID uuid.UUID
		expectRes        *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &rabbitmqhandler.Request{
				URI:    "/v1/accounts/customer_id/6b16ec0c-09ff-11ee-bd17-1f6f65cee5c7",
				Method: rabbitmqhandler.RequestMethodGet,
			},

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("6b76f5ac-09ff-11ee-b6ff-8790f56e5a46"),
			},

			expectCustomerID: uuid.FromStringOrNil("6b16ec0c-09ff-11ee-bd17-1f6f65cee5c7"),
			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6b76f5ac-09ff-11ee-b6ff-8790f56e5a46","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","type":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockAccount.EXPECT().GetByCustomerID(gomock.Any(), tt.expectCustomerID).Return(tt.responseAccount, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1AccountsCustomerIDIDIsValidBalancePost(t *testing.T) {

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		responseValid bool

		expectCustomerID uuid.UUID
		expectRes        *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &rabbitmqhandler.Request{
				URI:    "/v1/accounts/customer_id/6bb2670e-09ff-11ee-b8cb-e30d5d7c597c/is_valid_balance",
				Method: rabbitmqhandler.RequestMethodPost,
			},

			responseValid: true,

			expectCustomerID: uuid.FromStringOrNil("6bb2670e-09ff-11ee-b8cb-e30d5d7c597c"),
			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"valid":true}`),
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

			mockAccount.EXPECT().IsValidBalanceByCustomerID(gomock.Any(), tt.expectCustomerID).Return(tt.responseValid, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1AccountsIDBalanceAddForcePost(t *testing.T) {

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		responseAccount *account.Account

		expectAccountID uuid.UUID
		expectBalance   float32
		expectRes       *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &rabbitmqhandler.Request{
				URI:      "/v1/accounts/42d34adc-0dbb-11ee-a41b-eb337ba453c8/balance_add_force",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"balance":20}`),
			},

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("42d34adc-0dbb-11ee-a41b-eb337ba453c8"),
			},

			expectAccountID: uuid.FromStringOrNil("42d34adc-0dbb-11ee-a41b-eb337ba453c8"),
			expectBalance:   20,
			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"42d34adc-0dbb-11ee-a41b-eb337ba453c8","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","type":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockAccount.EXPECT().AddBalance(gomock.Any(), tt.expectAccountID, tt.expectBalance).Return(tt.responseAccount, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1AccountsIDBalanceSubtractForcePost(t *testing.T) {

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		responseAccount *account.Account

		expectAccountID uuid.UUID
		expectBalance   float32
		expectRes       *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &rabbitmqhandler.Request{
				URI:      "/v1/accounts/43180e06-0dbb-11ee-8124-17d122da2950/balance_subtract_force",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"balance":20}`),
			},

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("43180e06-0dbb-11ee-8124-17d122da2950"),
			},

			expectAccountID: uuid.FromStringOrNil("43180e06-0dbb-11ee-8124-17d122da2950"),
			expectBalance:   20,
			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"43180e06-0dbb-11ee-8124-17d122da2950","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","type":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockAccount.EXPECT().SubtractBalance(gomock.Any(), tt.expectAccountID, tt.expectBalance).Return(tt.responseAccount, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
