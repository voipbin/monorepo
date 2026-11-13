package servicehandler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	hmhook "monorepo/bin-hook-manager/models/hook"
)

func Test_Billing(t *testing.T) {
	body := []byte(`{"event_id":"evt_001","event_type":"transaction.completed"}`)

	tests := []struct {
		name string

		host      string
		path      string
		body      []byte
		signature string

		expectReq *hmhook.Hook
	}{
		{
			name: "paddle webhook - signature header is forwarded",

			host:      "hook.voipbin.net",
			path:      "/v1.0/billing/paddle",
			body:      body,
			signature: "ts=1700000000;h1=0123456789abcdef",

			expectReq: &hmhook.Hook{
				ReceivedURI:  "hook.voipbin.net/v1.0/billing/paddle",
				ReceivedData: body,

				ReceivedSignature: "ts=1700000000;h1=0123456789abcdef",
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
			if tt.signature != "" {
				r.Header.Set("Paddle-Signature", tt.signature)
			}

			mockReq.EXPECT().BillingV1PaddleHook(ctx, tt.expectReq).Return(nil)

			if err := h.Billing(ctx, r); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Billing_Error(t *testing.T) {
	body := []byte(`{"event_id":"evt_001","event_type":"transaction.completed"}`)
	signature := "ts=1700000000;h1=0123456789abcdef"

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
	}

	ctx := context.Background()

	r, _ := http.NewRequest("POST", "http://hook.voipbin.net/v1.0/billing/paddle", bytes.NewReader(body))
	r.Host = "hook.voipbin.net"
	r.Header.Set("Paddle-Signature", signature)

	expectReq := &hmhook.Hook{
		ReceivedURI:       "hook.voipbin.net/v1.0/billing/paddle",
		ReceivedData:      body,
		ReceivedSignature: signature,
	}
	mockReq.EXPECT().BillingV1PaddleHook(ctx, expectReq).Return(fmt.Errorf("billing hook error"))

	if err := h.Billing(ctx, r); err == nil {
		t.Error("Expected error, got nil")
	}
}

// The hook service checks that the header exists. It does not verify the signature: every header
// value is forwarded as received and the billing service decides.
func Test_Billing_HeaderPresence(t *testing.T) {
	body := []byte(`{"event_id":"evt_001","event_type":"transaction.completed"}`)

	tests := []struct {
		name string

		signature string

		expectCall  bool // whether reqHandler.BillingV1PaddleHook should be called
		expectError bool
	}{
		{name: "well formed header is forwarded", signature: "ts=1700000000;h1=0123456789abcdef", expectCall: true},
		{name: "missing header is rejected", signature: "", expectError: true},
		{name: "malformed header is forwarded", signature: "invalid-format", expectCall: true},
		{name: "wrong hash is forwarded", signature: "ts=1700000000;h1=0000000000000000000000000000000000000000000000000000000000000000", expectCall: true},
		{name: "stale timestamp is forwarded", signature: "ts=1234567890;h1=00", expectCall: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
			}

			if tt.expectCall {
				mockReq.EXPECT().BillingV1PaddleHook(gomock.Any(), gomock.Any()).Return(nil)
			}

			r, _ := http.NewRequest("POST", "http://hook.voipbin.net/v1.0/billing/paddle", bytes.NewReader(body))
			r.Host = "hook.voipbin.net"
			if tt.signature != "" {
				r.Header.Set("Paddle-Signature", tt.signature)
			}

			err := h.Billing(context.Background(), r)
			if (err != nil) != tt.expectError {
				t.Errorf("Billing() error = %v, expectError %v", err, tt.expectError)
			}

			// a missing header is a client error (400), never a server error
			if tt.expectError {
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Errorf("Wrong match. expect: ValidationError, got: %T %v", err, err)
				}
			}
		})
	}
}

func Test_Billing_SignatureForwarded(t *testing.T) {
	body := []byte(`{"event_id":"evt_001","event_type":"transaction.completed"}`)

	// The header carries an extra unknown field and the parts are in the reverse order. The
	// value must reach billing byte for byte.
	signature := "h1=0123456789abcdef;ts=1700000000;x=Y"

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
	}

	ctx := context.Background()
	r, _ := http.NewRequest("POST", "http://hook.voipbin.net/v1.0/billing/paddle", bytes.NewReader(body))
	r.Host = "hook.voipbin.net"
	r.Header.Set("Paddle-Signature", signature)

	expectReq := &hmhook.Hook{
		ReceivedURI:       "hook.voipbin.net/v1.0/billing/paddle",
		ReceivedData:      body,
		ReceivedSignature: signature,
	}
	mockReq.EXPECT().BillingV1PaddleHook(ctx, expectReq).Return(nil)

	if err := h.Billing(ctx, r); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_Billing_BillingRejection(t *testing.T) {
	body := []byte(`{"event_id":"evt_001","event_type":"transaction.completed"}`)
	signature := "ts=1700000000;h1=0123456789abcdef"

	tests := []struct {
		name string

		billingErr error

		expectValidation bool
	}{
		{
			name:             "unauthorized is a validation error",
			billingErr:       requesthandler.ErrUnauthorized,
			expectValidation: true,
		},
		{
			name:             "wrapped unauthorized is a validation error",
			billingErr:       fmt.Errorf("wrapped: %w", requesthandler.ErrUnauthorized),
			expectValidation: true,
		},
		{
			name:             "bad request keeps the current behavior",
			billingErr:       requesthandler.ErrBadRequest,
			expectValidation: false,
		},
		{
			name:             "internal keeps the current behavior",
			billingErr:       requesthandler.ErrInternal,
			expectValidation: false,
		},
		{
			name:             "other error keeps the current behavior",
			billingErr:       fmt.Errorf("billing hook error"),
			expectValidation: false,
		},
		{
			name:             "timeout keeps the current behavior",
			billingErr:       context.DeadlineExceeded,
			expectValidation: false,
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
			r, _ := http.NewRequest("POST", "http://hook.voipbin.net/v1.0/billing/paddle", bytes.NewReader(body))
			r.Host = "hook.voipbin.net"
			r.Header.Set("Paddle-Signature", signature)

			mockReq.EXPECT().BillingV1PaddleHook(ctx, gomock.Any()).Return(tt.billingErr)

			err := h.Billing(ctx, r)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			var valErr *ValidationError
			if got := errors.As(err, &valErr); got != tt.expectValidation {
				t.Errorf("Wrong match. expect ValidationError: %v, got: %v (%T)", tt.expectValidation, got, err)
			}
		})
	}
}

func Test_Billing_NotForwardedWithoutHeader(t *testing.T) {
	body := []byte(`{"event_id":"evt_001"}`)

	mc := gomock.NewController(t)
	defer mc.Finish()

	// No call to BillingV1PaddleHook is expected: gomock fails the test on any call.
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
	}

	r, _ := http.NewRequest("POST", "http://hook.voipbin.net/v1.0/billing/paddle", bytes.NewReader(body))
	r.Host = "hook.voipbin.net"

	if err := h.Billing(context.Background(), r); err == nil {
		t.Error("Expected error, got nil")
	}
}
