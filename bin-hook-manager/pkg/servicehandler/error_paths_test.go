package servicehandler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	hmhook "monorepo/bin-hook-manager/models/hook"
)

var errReadFailed = errors.New("synthetic read failure")

// failingBodies returns request bodies that fail on the first read and after a partial read.
func failingBodies() map[string]func() io.Reader {
	return map[string]func() io.Reader{
		"immediate failure": func() io.Reader { return iotest.ErrReader(errReadFailed) },
		"failure after partial read": func() io.Reader {
			return io.MultiReader(strings.NewReader(`{"partial":`), iotest.ErrReader(errReadFailed))
		},
	}
}

func Test_BodyReadError(t *testing.T) {
	tests := []struct {
		name string

		method string
		call   func(h *serviceHandler, r *http.Request) error
	}{
		{
			name:   "email",
			method: http.MethodPost,
			call: func(h *serviceHandler, r *http.Request) error {
				return h.Email(context.Background(), r)
			},
		},
		{
			name:   "message",
			method: http.MethodPost,
			call: func(h *serviceHandler, r *http.Request) error {
				return h.Message(context.Background(), r)
			},
		},
		{
			name:   "conversation post",
			method: http.MethodPost,
			call: func(h *serviceHandler, r *http.Request) error {
				_, err := h.Conversation(context.Background(), r)
				return err
			},
		},
		{
			name:   "conversation get",
			method: http.MethodGet,
			call: func(h *serviceHandler, r *http.Request) error {
				res, err := h.Conversation(context.Background(), r)
				if res != "" {
					t.Errorf("Wrong match. expect: empty challenge, got: %s", res)
				}
				return err
			},
		},
	}

	for _, tt := range tests {
		for bodyName, body := range failingBodies() {
			t.Run(tt.name+" "+bodyName, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()

				// no expectation is set on the request handler: it must not be reached.
				mockReq := requesthandler.NewMockRequestHandler(mc)
				h := &serviceHandler{reqHandler: mockReq}

				r, _ := http.NewRequest(tt.method, "http://hook.example.com/v1.0/test", body())
				r.Host = "hook.example.com"

				err := tt.call(h, r)
				if err == nil {
					t.Fatalf("Wrong match. expect: error, got: nil")
				}
				if !errors.Is(err, errReadFailed) {
					t.Errorf("Wrong match. expect: wrapped read failure, got: %v", err)
				}
				if !strings.Contains(err.Error(), "could not read request body") {
					t.Errorf("Wrong match. expect: read body message, got: %v", err)
				}
			})
		}
	}
}

func Test_Billing_BodyReadError(t *testing.T) {
	for name, body := range failingBodies() {
		t.Run(name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq, paddleWebhookSecret: "synthetic-secret"}

			r, _ := http.NewRequest(http.MethodPost, "http://hook.example.com/v1.0/billing/paddle", body())

			err := h.Billing(context.Background(), r)

			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("Wrong match. expect: ValidationError, got: %v", err)
			}
			if !errors.Is(err, errReadFailed) {
				t.Errorf("Wrong match. expect: wrapped read failure, got: %v", err)
			}
			if !strings.Contains(err.Error(), "could not read request body") {
				t.Errorf("Wrong match. expect: read body message, got: %v", err)
			}
		})
	}
}

func Test_Billing_BodySizeBoundary(t *testing.T) {
	tests := []struct {
		name string

		size int

		expectMessage string
	}{
		{
			name:          "exactly the maximum size passes the size check",
			size:          maxWebhookBodySize,
			expectMessage: "missing Paddle-Signature header",
		},
		{
			name:          "one byte over the maximum size is rejected",
			size:          maxWebhookBodySize + 1,
			expectMessage: "request body exceeds maximum size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq, paddleWebhookSecret: "synthetic-secret"}

			r, _ := http.NewRequest(http.MethodPost, "http://hook.example.com/v1.0/billing/paddle", bytes.NewReader(make([]byte, tt.size)))

			err := h.Billing(context.Background(), r)

			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("Wrong match. expect: ValidationError, got: %v", err)
			}
			if !strings.Contains(err.Error(), tt.expectMessage) {
				t.Errorf("Wrong match. expect message containing: %s, got: %v", tt.expectMessage, err)
			}
		})
	}
}

func Test_verifyPaddleSignature_MissingFields(t *testing.T) {
	freshTS := fmt.Sprintf("%d", time.Now().Unix())
	body := []byte(`{"event":"synthetic"}`)
	validMAC := hmacSHA256("synthetic-secret", freshTS, body)

	tests := []struct {
		name string

		signature string
	}{
		{name: "empty h1 value", signature: "ts=" + freshTS + ";h1="},
		{name: "empty ts value", signature: "ts=;h1=" + validMAC},
		{name: "neither field present", signature: "x=1;y=2"},
		{name: "h1 present without ts", signature: "h1=" + validMAC + ";x=1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyPaddleSignature("synthetic-secret", tt.signature, body)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if !strings.Contains(err.Error(), "missing ts or h1") {
				t.Errorf("Wrong match. expect: missing ts or h1, got: %v", err)
			}
		})
	}
}

func Test_Conversation_GetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	r, _ := http.NewRequest(http.MethodGet, "http://hook.example.com/v1.0/conversation/line?hub.challenge=abc", nil)
	r.Host = "hook.example.com"

	expectReq := &hmhook.Hook{
		ReceivedURI:    "hook.example.com/v1.0/conversation/line?hub.challenge=abc",
		ReceivedData:   []byte{},
		ReceivedMethod: http.MethodGet,
	}
	errHook := fmt.Errorf("synthetic hook failure")
	// a challenge is returned together with the error to prove it is not passed through.
	mockReq.EXPECT().ConversationV1HookGet(ctx, expectReq).Return("leaked-challenge", errHook)

	res, err := h.Conversation(ctx, r)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if !errors.Is(err, errHook) {
		t.Errorf("Wrong match. expect: wrapped hook error, got: %v", err)
	}
	if res != "" {
		t.Errorf("Wrong match. expect: empty challenge, got: %s", res)
	}
}
