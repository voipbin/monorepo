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
