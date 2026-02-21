package listenhandler

import (
	"fmt"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/customerhandler"
)

func Test_processV1CustomersSignupPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseSignupResult *customer.SignupResult
		expectRes            *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/customers/signup",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"name":"test signup","detail":"signup detail","email":"signup@voipbin.net","phone_number":"+821100000001","address":"somewhere","webhook_method":"POST","webhook_uri":"test.com","client_ip":"10.0.0.1"}`),
			},

			responseSignupResult: &customer.SignupResult{
				Customer: &customer.Customer{
					ID:            uuid.FromStringOrNil("e1e2e3e4-0000-0000-0000-000000000001"),
					Name:          "test signup",
					Detail:        "signup detail",
					Email:         "signup@voipbin.net",
					PhoneNumber:   "+821100000001",
					Address:       "somewhere",
					WebhookMethod: "POST",
					WebhookURI:    "test.com",
				},
				TempToken: "abc123",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"customer":{"id":"e1e2e3e4-0000-0000-0000-000000000001","name":"test signup","detail":"signup detail","email":"signup@voipbin.net","phone_number":"+821100000001","address":"somewhere","webhook_method":"POST","webhook_uri":"test.com","billing_account_id":"00000000-0000-0000-0000-000000000000","email_verified":false,"status":"","tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null},"temp_token":"abc123"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				reqHandler:      mockReq,
				customerHandler: mockCustomer,
			}

			mockCustomer.EXPECT().Signup(
				gomock.Any(),
				"test signup",
				"signup detail",
				"signup@voipbin.net",
				"+821100000001",
				"somewhere",
				customer.WebhookMethod("POST"),
				"test.com",
				"10.0.0.1",
			).Return(tt.responseSignupResult, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1CustomersSignupPost_badRequest(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCustomer := customerhandler.NewMockCustomerHandler(mc)

	h := &listenHandler{
		sockHandler:     mockSock,
		reqHandler:      mockReq,
		customerHandler: mockCustomer,
	}

	req := &sock.Request{
		URI:      "/v1/customers/signup",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`invalid json`),
	}

	res, err := h.processRequest(req)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != 400 {
		t.Errorf("Wrong match. expect: 400, got: %d", res.StatusCode)
	}
}

func Test_processV1CustomersSignupPost_signupError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCustomer := customerhandler.NewMockCustomerHandler(mc)

	h := &listenHandler{
		sockHandler:     mockSock,
		reqHandler:      mockReq,
		customerHandler: mockCustomer,
	}

	req := &sock.Request{
		URI:      "/v1/customers/signup",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"name":"test","email":"test@voipbin.net"}`),
	}

	mockCustomer.EXPECT().Signup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("signup failed"))

	res, err := h.processRequest(req)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != 400 {
		t.Errorf("Wrong match. expect: 400, got: %d", res.StatusCode)
	}
}

func Test_processV1CustomersEmailVerifyPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseEmailVerifyResult *customer.EmailVerifyResult
		expectRes                 *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/customers/email_verify",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"token":"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"}`),
			},

			responseEmailVerifyResult: &customer.EmailVerifyResult{
				Customer: &customer.Customer{
					ID:            uuid.FromStringOrNil("f1f2f3f4-0000-0000-0000-000000000001"),
					Email:         "verify@voipbin.net",
					EmailVerified: true,
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"customer":{"id":"f1f2f3f4-0000-0000-0000-000000000001","email":"verify@voipbin.net","billing_account_id":"00000000-0000-0000-0000-000000000000","email_verified":true,"status":"","tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				reqHandler:      mockReq,
				customerHandler: mockCustomer,
			}

			mockCustomer.EXPECT().EmailVerify(
				gomock.Any(),
				"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			).Return(tt.responseEmailVerifyResult, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1CustomersEmailVerifyPost_badRequest(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCustomer := customerhandler.NewMockCustomerHandler(mc)

	h := &listenHandler{
		sockHandler:     mockSock,
		reqHandler:      mockReq,
		customerHandler: mockCustomer,
	}

	req := &sock.Request{
		URI:      "/v1/customers/email_verify",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`invalid json`),
	}

	res, err := h.processRequest(req)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != 400 {
		t.Errorf("Wrong match. expect: 400, got: %d", res.StatusCode)
	}
}

func Test_processV1CustomersEmailVerifyPost_verifyError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCustomer := customerhandler.NewMockCustomerHandler(mc)

	h := &listenHandler{
		sockHandler:     mockSock,
		reqHandler:      mockReq,
		customerHandler: mockCustomer,
	}

	req := &sock.Request{
		URI:      "/v1/customers/email_verify",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"token":"sometoken"}`),
	}

	mockCustomer.EXPECT().EmailVerify(gomock.Any(), "sometoken").Return(nil, fmt.Errorf("verify failed"))

	res, err := h.processRequest(req)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != 400 {
		t.Errorf("Wrong match. expect: 400, got: %d", res.StatusCode)
	}
}
