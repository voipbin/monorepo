package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/accounthandler"
)

func Test_processV1AccountsGet(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		responseFilters  map[string]string
		responseAccounts []*account.Account

		expectCustomerID uuid.UUID
		expectSize       uint64
		expectToken      string
		expectRes        *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/accounts?page_size=10&page_token=2023-06-08%2003:22:17.995000&filter_customer_id=bc8f9070-0e5a-11ee-b22e-97ef303987a3",
				Method: sock.RequestMethodGet,
			},

			responseFilters: map[string]string{
				"customer_id": "bc8f9070-0e5a-11ee-b22e-97ef303987a3",
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				rabbitSock:     mockSock,
				utilHandler:    mockUtil,
				accountHandler: mockAccount,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockAccount.EXPECT().Gets(gomock.Any(), tt.expectSize, tt.expectToken, tt.responseFilters).Return(tt.responseAccounts, nil)
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
		request *sock.Request

		responseAccount *account.Account

		expectCustomerID    uuid.UUID
		expectName          string
		expectDetail        string
		expectPaymentType   account.PaymentType
		expectPaymentMethod account.PaymentMethod
		expectRes           *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/accounts",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"c28443b6-0e75-11ee-90ec-1bb28081d375","name":"test name","detail":"test detail","payment_type": "prepaid", "payment_method": ""}`),
			},

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("c28443b6-0e75-11ee-90ec-1bb28081d375"),
			},

			expectCustomerID:    uuid.FromStringOrNil("c28443b6-0e75-11ee-90ec-1bb28081d375"),
			expectName:          "test name",
			expectDetail:        "test detail",
			expectPaymentType:   account.PaymentTypePrepaid,
			expectPaymentMethod: account.PaymentMethodNone,

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

			mockAccount.EXPECT().Create(gomock.Any(), tt.expectCustomerID, tt.expectName, tt.expectDetail, tt.expectPaymentType, tt.expectPaymentMethod).Return(tt.responseAccount, nil)
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
		request *sock.Request

		responseAccount *account.Account

		expectID  uuid.UUID
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/accounts/922907b6-0942-11ee-960e-f31d2cc10daa",
				Method: sock.RequestMethodGet,
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

// func Test_processV1AccountsCustomerIDIDGet(t *testing.T) {

// 	type test struct {
// 		name    string
// 		request *sock.Request

// 		responseAccount *account.Account

// 		expectCustomerID uuid.UUID
// 		expectRes        *rabbitmqhandler.Response
// 	}

// 	tests := []test{
// 		{
// 			name: "normal",
// 			request: &sock.Request{
// 				URI:    "/v1/accounts/customer_id/6b16ec0c-09ff-11ee-bd17-1f6f65cee5c7",
// 				Method: sock.RequestMethodGet,
// 			},

// 			responseAccount: &account.Account{
// 				ID: uuid.FromStringOrNil("6b76f5ac-09ff-11ee-b6ff-8790f56e5a46"),
// 			},

// 			expectCustomerID: uuid.FromStringOrNil("6b16ec0c-09ff-11ee-bd17-1f6f65cee5c7"),
// 			expectRes: &rabbitmqhandler.Response{
// 				StatusCode: 200,
// 				DataType:   "application/json",
// 				Data:       []byte(`{"id":"6b76f5ac-09ff-11ee-b6ff-8790f56e5a46","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","type":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`),
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockSock := rabbitmqhandler.NewMockRabbit(mc)
// 			mockAccount := accounthandler.NewMockAccountHandler(mc)

// 			h := &listenHandler{
// 				rabbitSock:     mockSock,
// 				accountHandler: mockAccount,
// 			}

// 			mockAccount.EXPECT().GetByCustomerID(gomock.Any(), tt.expectCustomerID).Return(tt.responseAccount, nil)
// 			res, err := h.processRequest(tt.request)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if reflect.DeepEqual(res, tt.expectRes) != true {
// 				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
// 			}
// 		})
// 	}
// }

func Test_processV1AccountsIDPut(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		responseAccount *account.Account

		expectAccountID uuid.UUID
		expectName      string
		expectDetail    string
		expectRes       *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/accounts/3a952284-4ccf-11ee-bd5e-03a7d7220fad",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"update name","detail":"update detail"}`),
			},

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("3a952284-4ccf-11ee-bd5e-03a7d7220fad"),
			},

			expectAccountID: uuid.FromStringOrNil("3a952284-4ccf-11ee-bd5e-03a7d7220fad"),
			expectName:      "update name",
			expectDetail:    "update detail",
			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3a952284-4ccf-11ee-bd5e-03a7d7220fad","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","type":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockAccount.EXPECT().UpdateBasicInfo(gomock.Any(), tt.expectAccountID, tt.expectName, tt.expectDetail).Return(tt.responseAccount, nil)
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

func Test_processV1AccountsIDDelete(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		responseAccount *account.Account

		expectAccountID uuid.UUID
		expectRes       *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/accounts/a9e3587c-4ccf-11ee-9872-8b9300051977",
				Method: sock.RequestMethodDelete,
			},

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("a9e3587c-4ccf-11ee-9872-8b9300051977"),
			},

			expectAccountID: uuid.FromStringOrNil("a9e3587c-4ccf-11ee-9872-8b9300051977"),
			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a9e3587c-4ccf-11ee-9872-8b9300051977","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","type":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockAccount.EXPECT().Delete(gomock.Any(), tt.expectAccountID).Return(tt.responseAccount, nil)
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
		request *sock.Request

		responseAccount *account.Account

		expectAccountID uuid.UUID
		expectBalance   float32
		expectRes       *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/accounts/42d34adc-0dbb-11ee-a41b-eb337ba453c8/balance_add_force",
				Method:   sock.RequestMethodPost,
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
		request *sock.Request

		responseAccount *account.Account

		expectAccountID uuid.UUID
		expectBalance   float32
		expectRes       *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/accounts/43180e06-0dbb-11ee-8124-17d122da2950/balance_subtract_force",
				Method:   sock.RequestMethodPost,
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

func Test_processV1AccountsIDIsValidBalancePost(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		responseValid bool

		expectAccountID   uuid.UUID
		expectBillingType billing.ReferenceType
		expectCountry     string
		expectCount       int
		expectRes         *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/accounts/5a687db0-133e-11ee-b2ff-2f0139f4ec84/is_valid_balance",
				Method:   sock.RequestMethodPost,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"billing_type":"call","country":"us","count":3}`),
			},

			responseValid: true,

			expectAccountID:   uuid.FromStringOrNil("5a687db0-133e-11ee-b2ff-2f0139f4ec84"),
			expectBillingType: billing.ReferenceTypeCall,
			expectCountry:     "us",
			expectCount:       3,
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

			mockAccount.EXPECT().IsValidBalance(gomock.Any(), tt.expectAccountID, tt.expectBillingType, tt.expectCountry, tt.expectCount).Return(tt.responseValid, nil)
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

func Test_processV1AccountsIDPaymentInfoPut(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		responseAccount *account.Account

		expectAccountID     uuid.UUID
		expectPaymentType   account.PaymentType
		expectPaymentMethod account.PaymentMethod
		expectRes           *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/accounts/512ab538-4cd2-11ee-91be-7779c29dd4f8/payment_info",
				Method:   sock.RequestMethodPut,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"payment_type":"prepaid","payment_method":""}`),
			},

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("512ab538-4cd2-11ee-91be-7779c29dd4f8"),
			},

			expectAccountID:     uuid.FromStringOrNil("512ab538-4cd2-11ee-91be-7779c29dd4f8"),
			expectPaymentType:   account.PaymentTypePrepaid,
			expectPaymentMethod: account.PaymentMethodNone,
			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"512ab538-4cd2-11ee-91be-7779c29dd4f8","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","type":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			mockAccount.EXPECT().UpdatePaymentInfo(gomock.Any(), tt.expectAccountID, tt.expectPaymentType, tt.expectPaymentMethod).Return(tt.responseAccount, nil)
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
