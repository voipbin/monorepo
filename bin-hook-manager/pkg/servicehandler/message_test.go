package servicehandler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	hmhook "monorepo/bin-hook-manager/models/hook"
)

func Test_Message(t *testing.T) {
	tests := []struct {
		name string

		host string
		path string
		body []byte

		expectReq *hmhook.Hook
	}{
		{
			name: "normal",

			host: "hook.voipbin.net",
			path: "/v1.0/messages",
			body: []byte(`{"key1":"val1"}`),

			expectReq: &hmhook.Hook{
				ReceivedURI:  "hook.voipbin.net/v1.0/messages",
				ReceivedData: []byte(`{"key1":"val1"}`),
			},
		},
		{
			name: "message telnyx",

			host: "hook.voipbin.net",
			path: "/v1.0/messages/telnyx",
			body: []byte(`{"key1":"val1"}`),

			expectReq: &hmhook.Hook{
				ReceivedURI:  "hook.voipbin.net/v1.0/messages/telnyx",
				ReceivedData: []byte(`{"key1":"val1"}`),
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

			mockReq.EXPECT().MessageV1Hook(ctx, tt.expectReq).Return(nil)

			if err := h.Message(ctx, r); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}

}

func Test_Message_Error(t *testing.T) {
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
			path: "/v1.0/messages",
			body: []byte(`{"key1":"val1"}`),

			expectReq: &hmhook.Hook{
				ReceivedURI:  "hook.voipbin.net/v1.0/messages",
				ReceivedData: []byte(`{"key1":"val1"}`),
			},
			expectError: fmt.Errorf("message hook error"),
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

			mockReq.EXPECT().MessageV1Hook(ctx, tt.expectReq).Return(tt.expectError)

			if err := h.Message(ctx, r); err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func Test_Message_twilio(t *testing.T) {
	const (
		host = "hook.voipbin.net"
		path = "/v1.0/messages/twilio"
		sig  = "fake-signature"
	)
	body := []byte("MessageSid=SM0123456789abcdef0123456789abcdef&MessageStatus=delivered")

	newRequest := func(t *testing.T, requestBody []byte, signature string) *http.Request {
		t.Helper()

		r, _ := http.NewRequest("POST", "http://"+host+path, bytes.NewReader(requestBody))
		r.Host = host
		if signature != "" {
			r.Header.Set("X-Twilio-Signature", signature)
		}

		return r
	}

	t.Run("the signature is forwarded as received", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := serviceHandler{reqHandler: mockReq}
		ctx := context.Background()

		mockReq.EXPECT().MessageV1Hook(ctx, &hmhook.Hook{
			ReceivedURI:       host + path,
			ReceivedData:      body,
			ReceivedSignature: sig,
		}).Return(nil)

		if err := h.Message(ctx, newRequest(t, body, sig)); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("the telnyx request is the same with or without the header", func(t *testing.T) {
		for _, header := range []string{"", sig} {
			mc := gomock.NewController(t)

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			r, _ := http.NewRequest("POST", "http://"+host+"/v1.0/messages/telnyx", bytes.NewReader(body))
			r.Host = host
			if header != "" {
				r.Header.Set("X-Twilio-Signature", header)
			}

			// the signature is never forwarded on the telnyx path
			mockReq.EXPECT().MessageV1Hook(ctx, &hmhook.Hook{ReceivedURI: host + "/v1.0/messages/telnyx", ReceivedData: body}).Return(nil)

			if err := h.Message(ctx, r); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			mc.Finish()
		}
	})

	t.Run("no signature header is refused before the request is forwarded", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := serviceHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)} // any call fails the test

		err := h.Message(context.Background(), newRequest(t, body, ""))
		var valErr *ValidationError
		if !errors.As(err, &valErr) {
			t.Errorf("Wrong match. expect: a validation error, got: %v", err)
		}
	})

	t.Run("a body that is too large is refused", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := serviceHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

		err := h.Message(context.Background(), newRequest(t, []byte(strings.Repeat("a", maxTwilioBodySize+1)), sig))
		var valErr *ValidationError
		if !errors.As(err, &valErr) {
			t.Errorf("Wrong match. expect: a validation error, got: %v", err)
		}
	})

	t.Run("a body of the largest size is forwarded", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := serviceHandler{reqHandler: mockReq}
		largest := []byte(strings.Repeat("a", maxTwilioBodySize))

		mockReq.EXPECT().MessageV1Hook(gomock.Any(), gomock.Any()).Return(nil)

		if err := h.Message(context.Background(), newRequest(t, largest, sig)); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("a refusal of the message-manager is a validation error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := serviceHandler{reqHandler: mockReq}

		mockReq.EXPECT().MessageV1Hook(gomock.Any(), gomock.Any()).Return(requesthandler.ErrUnauthorized)

		err := h.Message(context.Background(), newRequest(t, body, sig))
		var valErr *ValidationError
		if !errors.As(err, &valErr) {
			t.Errorf("Wrong match. expect: a validation error, got: %v", err)
		}
	})

	t.Run("another error keeps being a plain error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := serviceHandler{reqHandler: mockReq}

		mockReq.EXPECT().MessageV1Hook(gomock.Any(), gomock.Any()).Return(fmt.Errorf("rpc failed"))

		err := h.Message(context.Background(), newRequest(t, body, sig))
		var valErr *ValidationError
		if err == nil || errors.As(err, &valErr) {
			t.Errorf("Wrong match. expect: a plain error, got: %v", err)
		}
	})
}
