package listenhandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-customer-manager/pkg/accesskeyhandler"
	"monorepo/bin-customer-manager/pkg/customerhandler"

	"go.uber.org/mock/gomock"
)

func TestNewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCustomer := customerhandler.NewMockCustomerHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := NewListenHandler(mockSock, mockReq, mockCustomer, mockAccesskey)
	if h == nil {
		t.Error("NewListenHandler returned nil")
	}
}

func TestSimpleResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"200", 200},
		{"404", 404},
		{"500", 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := simpleResponse(tt.statusCode)
			if res.StatusCode != tt.statusCode {
				t.Errorf("simpleResponse(%d).StatusCode = %d, expected %d", tt.statusCode, res.StatusCode, tt.statusCode)
			}
		})
	}
}

func Test_processRequest_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCustomer := customerhandler.NewMockCustomerHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &listenHandler{
		sockHandler:      mockSock,
		reqHandler:       mockReq,
		customerHandler:  mockCustomer,
		accesskeyHandler: mockAccesskey,
	}

	request := &sock.Request{
		URI:      "/v1/unknown",
		Method:   sock.RequestMethodGet,
		DataType: "application/json",
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest returned error: %v", err)
	}

	if res.StatusCode != 404 {
		t.Errorf("processRequest returned status %d, expected 404", res.StatusCode)
	}
}
