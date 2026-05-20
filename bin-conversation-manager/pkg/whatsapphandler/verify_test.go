package whatsapphandler_test

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/pkg/whatsapphandler"
)

func TestVerifyWebhook_Success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := whatsapphandler.NewWhatsAppHandler(requesthandler.NewMockRequestHandler(mc))
	ac := &account.Account{}
	ac.Secret = "my-secret-token"

	got, err := h.VerifyWebhook(context.Background(), ac, "subscribe", "my-secret-token", "challenge-xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "challenge-xyz" {
		t.Fatalf("expected challenge-xyz, got %q", got)
	}
}

func TestVerifyWebhook_WrongToken(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := whatsapphandler.NewWhatsAppHandler(requesthandler.NewMockRequestHandler(mc))
	ac := &account.Account{}
	ac.Secret = "correct-secret"

	_, err := h.VerifyWebhook(context.Background(), ac, "subscribe", "wrong-token", "challenge")
	if err == nil {
		t.Fatal("expected error for wrong verify_token")
	}
}

func TestVerifyWebhook_WrongMode(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := whatsapphandler.NewWhatsAppHandler(requesthandler.NewMockRequestHandler(mc))
	ac := &account.Account{}
	ac.Secret = "secret"

	_, err := h.VerifyWebhook(context.Background(), ac, "unsubscribe", "secret", "challenge")
	if err == nil {
		t.Fatal("expected error for wrong hub.mode")
	}
}
