package servicehandler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	hmhook "monorepo/bin-hook-manager/models/hook"
)

func Test_Billing(t *testing.T) {
	secret := "pdl_ntfset_test_secret"
	body := []byte(`{"event_id":"evt_001","event_type":"transaction.completed"}`)

	// Generate a valid signature with a fresh timestamp
	freshTS := strconv.FormatInt(time.Now().Unix(), 10)
	validMAC := hmacSHA256(secret, freshTS, body)
	validSignature := "ts=" + freshTS + ";h1=" + validMAC

	tests := []struct {
		name string

		host      string
		path      string
		body      []byte
		secret    string
		signature string

		expectReq *hmhook.Hook
	}{
		{
			name: "paddle webhook - valid secret and signature",

			host:      "hook.voipbin.net",
			path:      "/v1.0/billing/paddle",
			body:      body,
			secret:    secret,
			signature: validSignature,

			expectReq: &hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/billing/paddle",
				ReceivedData: body,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := serviceHandler{
				reqHandler:          mockReq,
				paddleWebhookSecret: tt.secret,
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
	secret := "pdl_ntfset_test_secret"
	body := []byte(`{"event_id":"evt_001","event_type":"transaction.completed"}`)

	// Generate a valid signature with a fresh timestamp
	freshTS := strconv.FormatInt(time.Now().Unix(), 10)
	validMAC := hmacSHA256(secret, freshTS, body)
	validSignature := "ts=" + freshTS + ";h1=" + validMAC

	tests := []struct {
		name string

		host      string
		path      string
		body      []byte
		secret    string
		signature string

		expectReq   *hmhook.Hook
		expectError error
	}{
		{
			name: "request handler error",

			host:      "hook.voipbin.net",
			path:      "/v1.0/billing/paddle",
			body:      body,
			secret:    secret,
			signature: validSignature,

			expectReq: &hmhook.Hook{
				ReceviedURI:  "hook.voipbin.net/v1.0/billing/paddle",
				ReceivedData: body,
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
				reqHandler:          mockReq,
				paddleWebhookSecret: tt.secret,
			}

			ctx := context.Background()

			r, _ := http.NewRequest("POST", "http://"+tt.host+tt.path, bytes.NewReader(tt.body))
			r.Host = tt.host
			if tt.signature != "" {
				r.Header.Set("Paddle-Signature", tt.signature)
			}

			mockReq.EXPECT().BillingV1PaddleHook(ctx, tt.expectReq).Return(tt.expectError)

			if err := h.Billing(ctx, r); err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func Test_Billing_EmptySecretRejected(t *testing.T) {
	body := []byte(`{"event_id":"evt_001","event_type":"transaction.completed"}`)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := serviceHandler{
		reqHandler:          mockReq,
		paddleWebhookSecret: "", // empty secret
	}

	ctx := context.Background()
	r, _ := http.NewRequest("POST", "http://hook.voipbin.net/v1.0/billing/paddle", bytes.NewReader(body))
	r.Host = "hook.voipbin.net"

	err := h.Billing(ctx, r)
	if err == nil {
		t.Fatal("Expected error when paddleWebhookSecret is empty, got nil")
	}

	// Verify it's a ValidationError (should return 400, not 500)
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("Expected *ValidationError, got %T: %v", err, err)
	}
}

func Test_Billing_SignatureVerification(t *testing.T) {
	body := []byte(`{"event_id":"evt_001","event_type":"transaction.completed"}`)
	secret := "pdl_ntfset_test_secret"

	// Generate a valid signature with a fresh timestamp
	freshTS := strconv.FormatInt(time.Now().Unix(), 10)
	validMAC := hmacSHA256(secret, freshTS, body)
	validSignature := "ts=" + freshTS + ";h1=" + validMAC

	tests := []struct {
		name        string
		secret      string
		signature   string
		expectCall  bool // whether reqHandler.BillingV1PaddleHook should be called
		expectError bool
	}{
		{
			name:        "valid signature - passes through",
			secret:      secret,
			signature:   validSignature,
			expectCall:  true,
			expectError: false,
		},
		{
			name:        "missing signature header",
			secret:      secret,
			signature:   "",
			expectError: true,
		},
		{
			name:        "invalid signature format",
			secret:      secret,
			signature:   "invalid-format",
			expectError: true,
		},
		{
			name:        "wrong hash",
			secret:      secret,
			signature:   "ts=" + freshTS + ";h1=0000000000000000000000000000000000000000000000000000000000000000",
			expectError: true,
		},
		{
			name:        "stale timestamp",
			secret:      secret,
			signature:   "ts=1234567890;h1=" + hmacSHA256(secret, "1234567890", body),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := serviceHandler{
				reqHandler:          mockReq,
				paddleWebhookSecret: tt.secret,
			}

			if tt.expectCall {
				mockReq.EXPECT().BillingV1PaddleHook(gomock.Any(), gomock.Any()).Return(nil)
			}

			ctx := context.Background()

			r, _ := http.NewRequest("POST", "http://hook.voipbin.net/v1.0/billing/paddle", bytes.NewReader(body))
			r.Host = "hook.voipbin.net"
			if tt.signature != "" {
				r.Header.Set("Paddle-Signature", tt.signature)
			}

			err := h.Billing(ctx, r)
			if (err != nil) != tt.expectError {
				t.Errorf("Billing() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func Test_verifyPaddleSignature(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"test":"data"}`)

	// Generate a valid signature with a fresh timestamp
	freshTS := strconv.FormatInt(time.Now().Unix(), 10)
	validMAC := hmacSHA256(secret, freshTS, body)

	tests := []struct {
		name      string
		signature string
		expectErr bool
	}{
		{
			name:      "valid signature",
			signature: "ts=" + freshTS + ";h1=" + validMAC,
			expectErr: false,
		},
		{
			name:      "invalid hash",
			signature: "ts=" + freshTS + ";h1=deadbeef",
			expectErr: true,
		},
		{
			name:      "missing ts",
			signature: "h1=" + validMAC,
			expectErr: true,
		},
		{
			name:      "missing h1",
			signature: "ts=" + freshTS,
			expectErr: true,
		},
		{
			name:      "empty string",
			signature: "",
			expectErr: true,
		},
		{
			name:      "stale timestamp rejected",
			signature: "ts=1234567890;h1=" + hmacSHA256(secret, "1234567890", body),
			expectErr: true,
		},
		{
			name:      "non-numeric timestamp",
			signature: "ts=abc;h1=" + validMAC,
			expectErr: true,
		},
		{
			name:      "future timestamp rejected",
			signature: "ts=" + strconv.FormatInt(time.Now().Unix()+600, 10) + ";h1=" + hmacSHA256(secret, strconv.FormatInt(time.Now().Unix()+600, 10), body),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyPaddleSignature(secret, tt.signature, body)
			if (err != nil) != tt.expectErr {
				t.Errorf("verifyPaddleSignature() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

// hmacSHA256 helper for test — computes HMAC-SHA256 of "ts:body"
func hmacSHA256(secret, ts string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + ":" + string(body)))
	return hex.EncodeToString(mac.Sum(nil))
}
