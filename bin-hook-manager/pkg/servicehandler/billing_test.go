package servicehandler

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	hmhook "monorepo/bin-hook-manager/models/hook"
)

func Test_Billing(t *testing.T) {
	tests := []struct {
		name string

		host string
		path string
		body []byte

		expectReq *hmhook.Hook
	}{
		{
			name: "paddle webhook",

			host: "hook.voipbin.net",
			path: "/v1.0/billing/paddle",
			body: []byte(`{"event_id":"evt_001","event_type":"transaction.completed"}`),

			expectReq: &hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/billing/paddle",
				ReceivedData: []byte(`{"event_id":"evt_001","event_type":"transaction.completed"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
			}

			ctx := context.Background()

			r, _ := http.NewRequest("POST", "http://"+tt.host+tt.path, bytes.NewReader(tt.body))
			r.Host = tt.host

			mockReq.EXPECT().BillingV1PaddleHook(ctx, tt.expectReq).Return(nil)

			if err := h.Billing(ctx, r); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Billing_Error(t *testing.T) {
	tests := []struct {
		name string

		host string
		path string
		body []byte

		expectReq   *hmhook.Hook
		expectError error
	}{
		{
			name: "request handler error",

			host: "hook.voipbin.net",
			path: "/v1.0/billing/paddle",
			body: []byte(`{"event_id":"evt_001","event_type":"transaction.completed"}`),

			expectReq: &hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/billing/paddle",
				ReceivedData: []byte(`{"event_id":"evt_001","event_type":"transaction.completed"}`),
			},
			expectError: fmt.Errorf("billing hook error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
			}

			ctx := context.Background()

			r, _ := http.NewRequest("POST", "http://"+tt.host+tt.path, bytes.NewReader(tt.body))
			r.Host = tt.host

			mockReq.EXPECT().BillingV1PaddleHook(ctx, tt.expectReq).Return(tt.expectError)

			if err := h.Billing(ctx, r); err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}
