package whatsapphandler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/whatsapphandler"
)

var errDownstream = errors.New("downstream failure")

func Test_Hook_payloadErrors(t *testing.T) {
	const secret = "appsecret"

	tests := []struct {
		name string

		provider json.RawMessage
		body     []byte

		expectErr string
	}{
		{
			name:      "provider data that is not json",
			provider:  json.RawMessage(`not-json`),
			body:      samplePayload("wamid.e1", "15550000001", "hi"),
			expectErr: "invalid provider_data",
		},
		{
			name:      "a signed body that is not json",
			body:      []byte(`{broken`),
			expectErr: "unmarshal payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no downstream request is declared: a rejected payload must not reach the request handler
			h := whatsapphandler.NewWhatsAppHandler(requesthandler.NewMockRequestHandler(mc))

			ac := makeAccount(secret)
			if tt.provider != nil {
				ac.ProviderData = tt.provider
			}

			res, err := h.Hook(context.Background(), ac, tt.body, validSignature(t, secret, tt.body))
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Hook() error = %v, want it to contain %q", err, tt.expectErr)
			}
			if res != nil {
				t.Errorf("Hook() result = %v, want nil", res)
			}
		})
	}
}

// A failure while handling one message is logged and the message is skipped. The webhook itself is
// still accepted, so the platform does not retry it.
func Test_Hook_messageFailureIsSkipped(t *testing.T) {
	const (
		secret = "appsecret"
		wamid  = "wamid.fail"
		waID   = "15550000002"
	)

	cvID := uuid.FromStringOrNil("5b6f2a10-3c4d-4e8f-9a1b-2c3d4e5f6a70")

	tests := []struct {
		name  string
		setup func(rh *requesthandler.MockRequestHandler)
	}{
		{
			name: "the duplicate check fails",
			setup: func(rh *requesthandler.MockRequestHandler) {
				rh.EXPECT().ConversationV1MessageList(gomock.Any(), "", uint64(1), gomock.Any()).Return(nil, errDownstream)
			},
		},
		{
			name: "the conversation lookup fails",
			setup: func(rh *requesthandler.MockRequestHandler) {
				rh.EXPECT().ConversationV1MessageList(gomock.Any(), "", uint64(1), gomock.Any()).Return([]message.Message{}, nil)
				rh.EXPECT().ConversationV1ConversationList(gomock.Any(), "", uint64(1), gomock.Any()).Return(nil, errDownstream)
			},
		},
		{
			name: "the conversation creation fails",
			setup: func(rh *requesthandler.MockRequestHandler) {
				rh.EXPECT().ConversationV1MessageList(gomock.Any(), "", uint64(1), gomock.Any()).Return([]message.Message{}, nil)
				rh.EXPECT().ConversationV1ConversationList(gomock.Any(), "", uint64(1), gomock.Any()).Return([]conversation.Conversation{}, nil)
				rh.EXPECT().ConversationV1ConversationCreate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), conversation.TypeWhatsApp, waID, gomock.Any(), gomock.Any()).Return(nil, errDownstream)
			},
		},
		{
			name: "the account update of a new conversation fails",
			setup: func(rh *requesthandler.MockRequestHandler) {
				rh.EXPECT().ConversationV1MessageList(gomock.Any(), "", uint64(1), gomock.Any()).Return([]message.Message{}, nil)
				rh.EXPECT().ConversationV1ConversationList(gomock.Any(), "", uint64(1), gomock.Any()).Return([]conversation.Conversation{}, nil)
				rh.EXPECT().ConversationV1ConversationCreate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), conversation.TypeWhatsApp, waID, gomock.Any(), gomock.Any()).
					Return(&conversation.Conversation{Identity: commonidentity.Identity{ID: cvID}}, nil)
				rh.EXPECT().ConversationV1ConversationUpdate(gomock.Any(), cvID, gomock.Any()).Return(nil, errDownstream)
			},
		},
		{
			name: "the message creation fails",
			setup: func(rh *requesthandler.MockRequestHandler) {
				rh.EXPECT().ConversationV1MessageList(gomock.Any(), "", uint64(1), gomock.Any()).Return([]message.Message{}, nil)
				rh.EXPECT().ConversationV1ConversationList(gomock.Any(), "", uint64(1), gomock.Any()).
					Return([]conversation.Conversation{{Identity: commonidentity.Identity{ID: cvID}}}, nil)
				rh.EXPECT().ConversationV1MessageCreate(gomock.Any(), uuid.Nil, gomock.Any(), cvID, message.DirectionIncoming, message.StatusDone, message.ReferenceTypeWhatsApp, uuid.Nil, wamid, "hi", gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errDownstream)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			rh := requesthandler.NewMockRequestHandler(mc)
			tt.setup(rh)
			h := whatsapphandler.NewWhatsAppHandler(rh)

			body := samplePayload(wamid, waID, "hi")
			res, err := h.Hook(context.Background(), makeAccount(secret), body, validSignature(t, secret, body))
			if err != nil {
				t.Fatalf("Hook() error = %v, want nil", err)
			}
			if len(res) != 0 {
				t.Errorf("Hook() results = %d, want 0 for the skipped message", len(res))
			}
		})
	}
}

func Test_Hook_signatureWithoutPrefix(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := whatsapphandler.NewWhatsAppHandler(requesthandler.NewMockRequestHandler(mc))

	body := samplePayload("wamid.noprefix", "15550000003", "hi")
	_, err := h.Hook(context.Background(), makeAccount("appsecret"), body, "0123456789abcdef")
	if err == nil || !strings.Contains(err.Error(), "signature missing sha256= prefix") {
		t.Errorf("Hook() error = %v, want the missing prefix error", err)
	}
}

func Test_Send_failures(t *testing.T) {
	validProvider := json.RawMessage(`{"phone_number_id":"123","app_secret":"sec"}`)

	tests := []struct {
		name string

		provider json.RawMessage
		// server builds the stub of the Meta API, nil keeps the default base address
		server func() (base string, closeFn func())

		expectErr string
	}{
		{
			name:      "provider data that is not json",
			provider:  json.RawMessage(`not-json`),
			expectErr: "invalid provider_data",
		},
		{
			name:     "an address that cannot form a request",
			provider: validProvider,
			server: func() (string, func()) {
				return "http://bad host", func() {}
			},
			expectErr: "create request",
		},
		{
			name:     "the server is not reachable",
			provider: validProvider,
			server: func() (string, func()) {
				ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
				ts.Close()
				return ts.URL, func() {}
			},
			expectErr: "http request",
		},
		{
			name:     "a response that is not json",
			provider: validProvider,
			server: func() (string, func()) {
				ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte("not json"))
				}))
				return ts.URL, ts.Close
			},
			expectErr: "decode response",
		},
		{
			name:     "a response without messages",
			provider: validProvider,
			server: func() (string, func()) {
				ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte(`{"messages":[]}`))
				}))
				return ts.URL, ts.Close
			},
			expectErr: "no messages in response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			if tt.server != nil {
				base, closeFn := tt.server()
				defer closeFn()

				orig := whatsapphandler.GraphAPIBase
				whatsapphandler.GraphAPIBase = base
				defer func() { whatsapphandler.GraphAPIBase = orig }()
			}

			h := whatsapphandler.NewWhatsAppHandler(requesthandler.NewMockRequestHandler(mc))
			ac := &account.Account{ProviderData: tt.provider, Token: "token"}

			wamid, err := h.Send(context.Background(), &conversation.Conversation{DialogID: "15550000004"}, ac, "hello")
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Send() error = %v, want it to contain %q", err, tt.expectErr)
			}
			if wamid != "" {
				t.Errorf("Send() wamid = %q, want empty on failure", wamid)
			}
		})
	}
}

func Test_Teardown(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no request is declared: the teardown must not call anything
	h := whatsapphandler.NewWhatsAppHandler(requesthandler.NewMockRequestHandler(mc))

	if err := h.Teardown(context.Background(), makeAccount("appsecret")); err != nil {
		t.Errorf("Teardown() error = %v, want nil", err)
	}
}
