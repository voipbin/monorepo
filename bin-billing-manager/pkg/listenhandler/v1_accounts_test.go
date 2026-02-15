package listenhandler

import (
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/accounthandler"
)

func Test_processV1AccountsIDGet(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		responseAccount *account.Account

		expectID  uuid.UUID
		expectRes *sock.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/accounts/922907b6-0942-11ee-960e-f31d2cc10daa",
				Method: sock.RequestMethodGet,
			},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("922907b6-0942-11ee-960e-f31d2cc10daa"),
				},
			},

			expectID: uuid.FromStringOrNil("922907b6-0942-11ee-960e-f31d2cc10daa"),
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"922907b6-0942-11ee-960e-f31d2cc10daa","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","plan_type":"","balance_credit":0,"balance_token":0,"payment_type":"","payment_method":"","tm_last_topup":null,"tm_next_topup":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
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

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1AccountsIDPut(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		responseAccount *account.Account

		expectAccountID uuid.UUID
		expectName      string
		expectDetail    string
		expectRes       *sock.Response
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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3a952284-4ccf-11ee-bd5e-03a7d7220fad"),
				},
			},

			expectAccountID: uuid.FromStringOrNil("3a952284-4ccf-11ee-bd5e-03a7d7220fad"),
			expectName:      "update name",
			expectDetail:    "update detail",
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3a952284-4ccf-11ee-bd5e-03a7d7220fad","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","plan_type":"","balance_credit":0,"balance_token":0,"payment_type":"","payment_method":"","tm_last_topup":null,"tm_next_topup":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
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

func Test_processV1AccountsIDBalanceAddForcePost(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		responseAccount *account.Account

		expectAccountID uuid.UUID
		expectBalance   int64
		expectRes       *sock.Response
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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("42d34adc-0dbb-11ee-a41b-eb337ba453c8"),
				},
			},

			expectAccountID: uuid.FromStringOrNil("42d34adc-0dbb-11ee-a41b-eb337ba453c8"),
			expectBalance:   20,
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"42d34adc-0dbb-11ee-a41b-eb337ba453c8","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","plan_type":"","balance_credit":0,"balance_token":0,"payment_type":"","payment_method":"","tm_last_topup":null,"tm_next_topup":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
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
		expectBalance   int64
		expectRes       *sock.Response
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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("43180e06-0dbb-11ee-8124-17d122da2950"),
				},
			},

			expectAccountID: uuid.FromStringOrNil("43180e06-0dbb-11ee-8124-17d122da2950"),
			expectBalance:   20,
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"43180e06-0dbb-11ee-8124-17d122da2950","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","plan_type":"","balance_credit":0,"balance_token":0,"payment_type":"","payment_method":"","tm_last_topup":null,"tm_next_topup":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
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
		expectRes         *sock.Response
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
			expectRes: &sock.Response{
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

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
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
		expectRes           *sock.Response
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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("512ab538-4cd2-11ee-91be-7779c29dd4f8"),
				},
			},

			expectAccountID:     uuid.FromStringOrNil("512ab538-4cd2-11ee-91be-7779c29dd4f8"),
			expectPaymentType:   account.PaymentTypePrepaid,
			expectPaymentMethod: account.PaymentMethodNone,
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"512ab538-4cd2-11ee-91be-7779c29dd4f8","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","plan_type":"","balance_credit":0,"balance_token":0,"payment_type":"","payment_method":"","tm_last_topup":null,"tm_next_topup":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
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

func Test_processV1AccountsIDIsValidResourceLimitPost(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		responseValid bool

		expectAccountID    uuid.UUID
		expectResourceType account.ResourceType
		expectRes          *sock.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/accounts/6b8a1c20-133e-11ee-a1b2-3f0139f4ec84/is_valid_resource_limit",
				Method:   sock.RequestMethodPost,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"resource_type":"agent"}`),
			},

			responseValid: true,

			expectAccountID:    uuid.FromStringOrNil("6b8a1c20-133e-11ee-a1b2-3f0139f4ec84"),
			expectResourceType: account.ResourceTypeAgent,
			expectRes: &sock.Response{
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

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				accountHandler: mockAccount,
			}

			mockAccount.EXPECT().IsValidResourceLimit(gomock.Any(), tt.expectAccountID, tt.expectResourceType).Return(tt.responseValid, nil)
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
