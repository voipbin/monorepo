package whatsapphandler_test

import (
	"context"
	"encoding/json"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/pkg/whatsapphandler"
)

func TestSetup_MissingPhoneNumberID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := whatsapphandler.NewWhatsAppHandler(requesthandler.NewMockRequestHandler(mc))
	ac := &account.Account{}
	ac.ProviderData = json.RawMessage(`{"phone_number_id":"","app_secret":"sec"}`)

	if err := h.Setup(context.Background(), ac); err == nil {
		t.Fatal("expected error for missing phone_number_id")
	}
}

func TestSetup_ValidData(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := whatsapphandler.NewWhatsAppHandler(requesthandler.NewMockRequestHandler(mc))
	ac := &account.Account{}
	ac.ProviderData = json.RawMessage(`{"phone_number_id":"12345","app_secret":"secret"}`)

	if err := h.Setup(context.Background(), ac); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetup_InvalidJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := whatsapphandler.NewWhatsAppHandler(requesthandler.NewMockRequestHandler(mc))
	ac := &account.Account{}
	ac.ProviderData = json.RawMessage(`not-json`)

	if err := h.Setup(context.Background(), ac); err == nil {
		t.Fatal("expected error for invalid JSON provider_data")
	}
}
