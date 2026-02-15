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

	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/customerhandler"
)

func Test_processV1CustomersCompleteSignupPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseResult *customer.CompleteSignupResult
		expectRes      *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/customers/complete_signup",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"temp_token":"tmp_abcdef123","code":"123456"}`),
			},

			responseResult: &customer.CompleteSignupResult{
				CustomerID: "d1d2d3d4-0000-0000-0000-000000000001",
				Accesskey: &accesskey.Accesskey{
					ID: uuid.FromStringOrNil("aaaa1111-bbbb-cccc-dddd-eeeeeeeeeeee"),
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"customer_id":"d1d2d3d4-0000-0000-0000-000000000001","accesskey":{"id":"aaaa1111-bbbb-cccc-dddd-eeeeeeeeeeee","customer_id":"00000000-0000-0000-0000-000000000000","token":"","tm_expire":null,"tm_create":null,"tm_update":null,"tm_delete":null}}`),
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

			mockCustomer.EXPECT().CompleteSignup(
				gomock.Any(),
				"tmp_abcdef123",
				"123456",
			).Return(tt.responseResult, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", string(tt.expectRes.Data), string(res.Data))
			}
		})
	}
}

func Test_processV1CustomersCompleteSignupPost_badRequest(t *testing.T) {
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
		URI:      "/v1/customers/complete_signup",
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

func Test_processV1CustomersCompleteSignupPost_completeSignupError(t *testing.T) {
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
		URI:      "/v1/customers/complete_signup",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"temp_token":"tmp_abc","code":"123456"}`),
	}

	mockCustomer.EXPECT().CompleteSignup(gomock.Any(), "tmp_abc", "123456").Return(nil, fmt.Errorf("too many attempts"))

	res, err := h.processRequest(req)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != 429 {
		t.Errorf("Wrong match. expect: 429, got: %d", res.StatusCode)
	}
}
