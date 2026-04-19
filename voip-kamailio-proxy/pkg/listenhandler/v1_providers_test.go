package listenhandler

import (
	"encoding/json"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/voip-kamailio-proxy/pkg/listenhandler/request"
)

func Test_processV1ProvidersHealthPost(t *testing.T) {
	tests := []struct {
		name             string
		request          *sock.Request
		expectStatusCode int
	}{
		{
			name: "invalid json body",
			request: &sock.Request{
				URI:    "/v1/providers/health",
				Method: sock.RequestMethodPost,
				Data:   []byte(`not valid json`),
			},
			expectStatusCode: 400,
		},
		{
			name: "empty hostname",
			request: &sock.Request{
				URI:    "/v1/providers/health",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"hostname": ""}`),
			},
			expectStatusCode: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := &listenHandler{
				sockHandler:       mockSock,
				rabbitQueueListen: "kamailio.request",
				sipTimeout:        5 * time.Second,
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Expected nil error, got: %v", err)
			}
			if res.StatusCode != tt.expectStatusCode {
				t.Errorf("Expected status %d, got %d", tt.expectStatusCode, res.StatusCode)
			}
		})
	}
}

func Test_processRequest_routing(t *testing.T) {
	tests := []struct {
		name             string
		request          *sock.Request
		expectStatusCode int
	}{
		{
			name: "unknown route returns 404",
			request: &sock.Request{
				URI:    "/v1/unknown",
				Method: sock.RequestMethodGet,
			},
			expectStatusCode: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := &listenHandler{
				sockHandler:       mockSock,
				rabbitQueueListen: "kamailio.request",
				sipTimeout:        5 * time.Second,
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Expected nil error, got: %v", err)
			}
			if res.StatusCode != tt.expectStatusCode {
				t.Errorf("Expected %d, got %d", tt.expectStatusCode, res.StatusCode)
			}
		})
	}
}

// Verify JSON response format for a healthy result
func Test_healthResponseFormat(t *testing.T) {
	res := &request.V1ResponseProvidersHealthPost{
		Status:     "healthy",
		ResultCode: "200",
	}
	data, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]string
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["status"] != "healthy" {
		t.Errorf("expected status=healthy, got %s", decoded["status"])
	}
	if decoded["result_code"] != "200" {
		t.Errorf("expected result_code=200, got %s", decoded["result_code"])
	}
}
