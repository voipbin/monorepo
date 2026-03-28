package paddlehandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"monorepo/bin-billing-manager/models/account"
)

func Test_CreatePortalSession(t *testing.T) {
	tests := []struct {
		name             string
		paddleCustomerID string
		serverStatus     int
		serverBody       string
		expectURL        string
		expectErr        bool
	}{
		{
			name:             "successful portal session",
			paddleCustomerID: "ctm_abc123",
			serverStatus:     201,
			serverBody:       `{"data":{"id":"cpls_123","urls":{"general":{"overview":"https://portal.paddle.com/session123"}}}}`,
			expectURL:        "https://portal.paddle.com/session123",
		},
		{
			name:             "empty customer ID",
			paddleCustomerID: "",
			expectErr:        true,
		},
		{
			name:             "paddle returns 401",
			paddleCustomerID: "ctm_abc123",
			serverStatus:     401,
			serverBody:       `{"error":{"type":"authentication_error","code":"unauthorized","detail":"Invalid API key"}}`,
			expectErr:        true,
		},
		{
			name:             "paddle returns 404",
			paddleCustomerID: "ctm_nonexistent",
			serverStatus:     404,
			serverBody:       `{"error":{"type":"not_found","code":"not_found","detail":"Customer not found"}}`,
			expectErr:        true,
		},
		{
			name:             "paddle returns 500",
			paddleCustomerID: "ctm_abc123",
			serverStatus:     500,
			serverBody:       `{"error":{"type":"internal_error","code":"internal_error","detail":"Internal server error"}}`,
			expectErr:        true,
		},
		{
			name:             "paddle returns empty portal URL",
			paddleCustomerID: "ctm_abc123",
			serverStatus:     201,
			serverBody:       `{"data":{"id":"cpls_123","urls":{"general":{"overview":""}}}}`,
			expectErr:        true,
		},
		{
			name:             "paddle returns malformed JSON",
			paddleCustomerID: "ctm_abc123",
			serverStatus:     201,
			serverBody:       `{invalid json`,
			expectErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h *paddleHandler

			if tt.serverStatus != 0 {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("Authorization") != "Bearer test-api-key" {
						t.Errorf("unexpected Authorization header: %s", r.Header.Get("Authorization"))
					}
					if r.Header.Get("Content-Type") != "application/json" {
						t.Errorf("unexpected Content-Type header: %s", r.Header.Get("Content-Type"))
					}
					if r.URL.Path != "/customer-portal-sessions" {
						t.Errorf("unexpected path: %s", r.URL.Path)
					}
					if r.Method != http.MethodPost {
						t.Errorf("unexpected method: %s", r.Method)
					}
					w.WriteHeader(tt.serverStatus)
					_, _ = w.Write([]byte(tt.serverBody))
				}))
				defer server.Close()

				h = &paddleHandler{
					apiKey:     "test-api-key",
					baseURL:    server.URL,
					httpClient: http.DefaultClient,
					priceMap:   map[string]account.PlanType{},
				}
			} else {
				h = &paddleHandler{
					apiKey:     "test-api-key",
					baseURL:    "http://localhost:0",
					httpClient: http.DefaultClient,
					priceMap:   map[string]account.PlanType{},
				}
			}

			url, err := h.CreatePortalSession(context.Background(), tt.paddleCustomerID)
			if (err != nil) != tt.expectErr {
				t.Errorf("error = %v, expectErr = %v", err, tt.expectErr)
			}
			if url != tt.expectURL {
				t.Errorf("url = %s, expectURL = %s", url, tt.expectURL)
			}
		})
	}
}

func Test_GetPlanTypeByPriceID(t *testing.T) {
	tests := []struct {
		name       string
		priceID    string
		expectPlan account.PlanType
		expectErr  bool
	}{
		{
			name:       "basic plan",
			priceID:    "pri_basic_123",
			expectPlan: account.PlanTypeBasic,
		},
		{
			name:       "professional plan",
			priceID:    "pri_pro_456",
			expectPlan: account.PlanTypeProfessional,
		},
		{
			name:      "unknown price ID",
			priceID:   "pri_unknown_789",
			expectErr: true,
		},
		{
			name:      "empty price ID",
			priceID:   "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewPaddleHandler("test-key", "pri_basic_123", "pri_pro_456")

			plan, err := h.GetPlanTypeByPriceID(tt.priceID)
			if (err != nil) != tt.expectErr {
				t.Errorf("error = %v, expectErr = %v", err, tt.expectErr)
			}
			if plan != tt.expectPlan {
				t.Errorf("plan = %s, expectPlan = %s", plan, tt.expectPlan)
			}
		})
	}
}

func Test_NewPaddleHandler(t *testing.T) {
	tests := []struct {
		name                string
		priceIDBasic        string
		priceIDProfessional string
		expectMapLen        int
	}{
		{
			name:                "both price IDs",
			priceIDBasic:        "pri_basic",
			priceIDProfessional: "pri_pro",
			expectMapLen:        2,
		},
		{
			name:                "only basic",
			priceIDBasic:        "pri_basic",
			priceIDProfessional: "",
			expectMapLen:        1,
		},
		{
			name:                "only professional",
			priceIDBasic:        "",
			priceIDProfessional: "pri_pro",
			expectMapLen:        1,
		},
		{
			name:                "no price IDs",
			priceIDBasic:        "",
			priceIDProfessional: "",
			expectMapLen:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewPaddleHandler("test-key", tt.priceIDBasic, tt.priceIDProfessional)
			ph := h.(*paddleHandler)
			if len(ph.priceMap) != tt.expectMapLen {
				t.Errorf("priceMap length = %d, expected = %d", len(ph.priceMap), tt.expectMapLen)
			}
			if ph.baseURL != defaultPaddleBaseURL {
				t.Errorf("baseURL = %s, expected = %s", ph.baseURL, defaultPaddleBaseURL)
			}
		})
	}
}
