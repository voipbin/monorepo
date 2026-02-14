package listenhandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-transfer-manager/pkg/transferhandler"
)

func TestNewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTransfer := transferhandler.NewMockTransferHandler(mc)

	h := NewListenHandler(mockSock, "test-queue", "test-exchange", mockTransfer)

	if h == nil {
		t.Error("Expected handler but got nil")
	}
}

func TestSimpleResponse(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedStatus int
	}{
		{
			name:           "returns_200_status",
			statusCode:     200,
			expectedStatus: 200,
		},
		{
			name:           "returns_404_status",
			statusCode:     404,
			expectedStatus: 404,
		},
		{
			name:           "returns_500_status",
			statusCode:     500,
			expectedStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := simpleResponse(tt.statusCode)

			if result.StatusCode != tt.expectedStatus {
				t.Errorf("Wrong status code. expect: %d, got: %d", tt.expectedStatus, result.StatusCode)
			}
		})
	}
}

func TestProcessRequest_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTransfer := transferhandler.NewMockTransferHandler(mc)

	h := &listenHandler{
		sockHandler:     mockSock,
		queueListen:     "test-queue",
		exchangeDelay:   "test-exchange",
		transferHandler: mockTransfer,
	}

	tests := []struct {
		name             string
		request          *sock.Request
		expectedStatus   int
		expectedNotFound bool
	}{
		{
			name: "returns_404_for_unknown_endpoint",
			request: &sock.Request{
				URI:    "/unknown/endpoint",
				Method: sock.RequestMethodGet,
			},
			expectedStatus:   404,
			expectedNotFound: true,
		},
		{
			name: "returns_404_for_wrong_method",
			request: &sock.Request{
				URI:    "/v1/transfers",
				Method: sock.RequestMethodGet,
			},
			expectedStatus:   404,
			expectedNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := h.processRequest(tt.request)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if response.StatusCode != tt.expectedStatus {
				t.Errorf("Wrong status code. expect: %d, got: %d", tt.expectedStatus, response.StatusCode)
			}
		})
	}
}

func TestProcessRequest_ErrorHandling(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTransfer := transferhandler.NewMockTransferHandler(mc)

	h := &listenHandler{
		sockHandler:     mockSock,
		queueListen:     "test-queue",
		exchangeDelay:   "test-exchange",
		transferHandler: mockTransfer,
	}

	tests := []struct {
		name           string
		request        *sock.Request
		expectedStatus int
	}{
		{
			name: "returns_400_for_invalid_json",
			request: &sock.Request{
				URI:      "/v1/transfers",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{invalid json`),
			},
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := h.processRequest(tt.request)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if response.StatusCode != tt.expectedStatus {
				t.Errorf("Wrong status code. expect: %d, got: %d", tt.expectedStatus, response.StatusCode)
			}
		})
	}
}
