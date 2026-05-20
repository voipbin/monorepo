package whatsapphandler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/whatsapphandler"

	"github.com/gofrs/uuid"
)

func TestSend_Success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// Stub the Meta Cloud API with a local test server.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]string{{"id": "wamid.test123"}},
		})
	}))
	defer ts.Close()

	// Override GraphAPIBase to point at the test server.
	orig := whatsapphandler.GraphAPIBase
	whatsapphandler.GraphAPIBase = ts.URL
	defer func() { whatsapphandler.GraphAPIBase = orig }()

	h := whatsapphandler.NewWhatsAppHandler(requesthandler.NewMockRequestHandler(mc))

	ac := &account.Account{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		},
		Token: "test-bearer-token",
	}
	ac.ProviderData = json.RawMessage(`{"phone_number_id":"1234567890","app_secret":"secret"}`)

	cv := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		},
		DialogID: "15551234567",
	}

	wamid, err := h.Send(context.Background(), cv, ac, "Hello!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wamid != "wamid.test123" {
		t.Fatalf("expected wamid.test123, got %q", wamid)
	}
}

func TestSend_APIError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// Return HTTP 500 from the stub.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer ts.Close()

	orig := whatsapphandler.GraphAPIBase
	whatsapphandler.GraphAPIBase = ts.URL
	defer func() { whatsapphandler.GraphAPIBase = orig }()

	h := whatsapphandler.NewWhatsAppHandler(requesthandler.NewMockRequestHandler(mc))

	ac := &account.Account{}
	ac.ProviderData = json.RawMessage(`{"phone_number_id":"123","app_secret":"sec"}`)
	ac.Token = "token"

	cv := &conversation.Conversation{DialogID: "15550000000"}

	_, err := h.Send(context.Background(), cv, ac, "Hello")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}
