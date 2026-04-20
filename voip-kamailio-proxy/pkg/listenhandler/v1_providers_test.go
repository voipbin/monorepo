package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/voip-kamailio-proxy/pkg/listenhandler/request"
	"monorepo/voip-kamailio-proxy/pkg/siphandler"
)

// stubHealthyChecker returns a SIPChecker stub that always reports healthy.
func stubHealthyChecker(_ context.Context, _ string, _ time.Duration) (*siphandler.HealthCheckResult, error) {
	return &siphandler.HealthCheckResult{Healthy: true, ResponseCode: "200"}, nil
}

// stubErrorChecker returns a SIPChecker stub that always returns an error.
func stubErrorChecker(_ context.Context, _ string, _ time.Duration) (*siphandler.HealthCheckResult, error) {
	return nil, fmt.Errorf("sip dial error")
}

func Test_processV1ProvidersHealthPost(t *testing.T) {
	tests := []struct {
		name             string
		sipChecker       siphandler.SIPChecker
		request          *sock.Request
		expectStatusCode int
		expectStatus     string
	}{
		{
			name:       "invalid json body",
			sipChecker: nil, // not called
			request: &sock.Request{
				URI:    "/v1/providers/health",
				Method: sock.RequestMethodPost,
				Data:   []byte(`not valid json`),
			},
			expectStatusCode: 400,
		},
		{
			name:       "empty hostname",
			sipChecker: nil, // not called
			request: &sock.Request{
				URI:    "/v1/providers/health",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"hostname": ""}`),
			},
			expectStatusCode: 400,
		},
		{
			name:       "healthy provider returns 200 with healthy status",
			sipChecker: stubHealthyChecker,
			request: &sock.Request{
				URI:    "/v1/providers/health",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"hostname": "sip.telnyx.com"}`),
			},
			expectStatusCode: 200,
			expectStatus:     "healthy",
		},
		{
			name:       "sip checker error returns 500",
			sipChecker: stubErrorChecker,
			request: &sock.Request{
				URI:    "/v1/providers/health",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"hostname": "sip.telnyx.com"}`),
			},
			expectStatusCode: 500,
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
				sipChecker:        tt.sipChecker,
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Expected nil error, got: %v", err)
			}
			if res.StatusCode != tt.expectStatusCode {
				t.Errorf("Expected status %d, got %d", tt.expectStatusCode, res.StatusCode)
			}
			if tt.expectStatus != "" {
				var body request.V1ResponseProvidersHealthPost
				if err := json.Unmarshal(res.Data, &body); err != nil {
					t.Fatalf("Could not unmarshal response body: %v", err)
				}
				if body.Status != tt.expectStatus {
					t.Errorf("Expected body.status=%q, got %q", tt.expectStatus, body.Status)
				}
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
				sipChecker:        nil, // not called for unknown routes
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
