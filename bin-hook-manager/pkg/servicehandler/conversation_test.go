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

func Test_Conversation(t *testing.T) {
	tests := []struct {
		name string

		method string
		host   string
		path   string
		body   []byte

		expectReq *hmhook.Hook
	}{
		{
			name: "normal POST",

			method: "POST",
			host:   "hook.voipbin.net",
			path:   "/v1.0/conversation",
			body:   []byte(`{"key1":"val1"}`),

			expectReq: &hmhook.Hook{
				ReceviedURI:      "hook.voipbin.net/v1.0/conversation",
				ReceivedData:     []byte(`{"key1":"val1"}`),
				ReceivedMethod:   "POST",
				ReceivedSignature: "",
			},
		},
		{
			name: "POST conversation with path",

			method: "POST",
			host:   "hook.voipbin.net",
			path:   "/v1.0/conversation/customers/id/line",
			body:   []byte(`{"test":"data"}`),

			expectReq: &hmhook.Hook{
				ReceviedURI:      "hook.voipbin.net/v1.0/conversation/customers/id/line",
				ReceivedData:     []byte(`{"test":"data"}`),
				ReceivedMethod:   "POST",
				ReceivedSignature: "",
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

			r, _ := http.NewRequest(tt.method, "http://"+tt.host+tt.path, bytes.NewReader(tt.body))
			r.Host = tt.host

			mockReq.EXPECT().ConversationV1Hook(ctx, tt.expectReq).Return(nil)

			challenge, err := h.Conversation(ctx, r)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if challenge != "" {
				t.Errorf("Wrong match. expect empty challenge, got: %v", challenge)
			}
		})
	}
}

func Test_Conversation_GET(t *testing.T) {
	tests := []struct {
		name string

		host              string
		path              string
		signature         string
		expectReq         *hmhook.Hook
		expectChallenge   string
	}{
		{
			name: "GET with challenge",

			host:      "hook.voipbin.net",
			path:      "/v1.0/conversation/customers/id/whatsapp?hub.mode=subscribe&hub.challenge=chal123&hub.verify_token=token",
			signature: "",
			expectReq: &hmhook.Hook{
				ReceviedURI:      "hook.voipbin.net/v1.0/conversation/customers/id/whatsapp?hub.mode=subscribe&hub.challenge=chal123&hub.verify_token=token",
				ReceivedData:     []byte{},
				ReceivedMethod:   "GET",
				ReceivedSignature: "",
			},
			expectChallenge: "chal123",
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

			r, _ := http.NewRequest("GET", "http://"+tt.host+tt.path, nil)
			r.Host = tt.host
			if tt.signature != "" {
				r.Header.Set("X-Hub-Signature-256", tt.signature)
			}

			mockReq.EXPECT().ConversationV1HookGet(ctx, tt.expectReq).Return(tt.expectChallenge, nil)

			challenge, err := h.Conversation(ctx, r)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if challenge != tt.expectChallenge {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChallenge, challenge)
			}
		})
	}
}

func Test_Conversation_POST_WithSignature(t *testing.T) {
	tests := []struct {
		name string

		host      string
		path      string
		body      []byte
		signature string

		expectReq *hmhook.Hook
	}{
		{
			name: "POST with X-Hub-Signature-256",

			host:      "hook.voipbin.net",
			path:      "/v1.0/conversation/customers/id/whatsapp",
			body:      []byte(`{"object":"whatsapp_business_account"}`),
			signature: "sha256=abc123",

			expectReq: &hmhook.Hook{
				ReceviedURI:      "hook.voipbin.net/v1.0/conversation/customers/id/whatsapp",
				ReceivedData:     []byte(`{"object":"whatsapp_business_account"}`),
				ReceivedMethod:   "POST",
				ReceivedSignature: "sha256=abc123",
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
			r.Header.Set("X-Hub-Signature-256", tt.signature)

			mockReq.EXPECT().ConversationV1Hook(ctx, tt.expectReq).Return(nil)

			challenge, err := h.Conversation(ctx, r)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if challenge != "" {
				t.Errorf("Wrong match. expect empty challenge, got: %v", challenge)
			}
		})
	}
}

func Test_Conversation_Error(t *testing.T) {
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
			path: "/v1.0/conversation",
			body: []byte(`{"key1":"val1"}`),

			expectReq: &hmhook.Hook{
				ReceviedURI:      "hook.voipbin.net/v1.0/conversation",
				ReceivedData:     []byte(`{"key1":"val1"}`),
				ReceivedMethod:   "POST",
				ReceivedSignature: "",
			},
			expectError: fmt.Errorf("request handler error"),
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

			mockReq.EXPECT().ConversationV1Hook(ctx, tt.expectReq).Return(tt.expectError)

			if _, err := h.Conversation(ctx, r); err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}
