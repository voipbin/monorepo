package whatsapphandler

//go:generate mockgen -package whatsapphandler -destination ./mock_whatsapphandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"
)

// HookResult contains the conversation and message produced from a WhatsApp inbound message.
type HookResult struct {
	Conversation *conversation.Conversation
	Message      *message.Message
}

// WhatsAppHandler defines the WhatsApp provider operations.
type WhatsAppHandler interface {
	Setup(ctx context.Context, ac *account.Account) error
	Teardown(ctx context.Context, ac *account.Account) error
	Send(ctx context.Context, cv *conversation.Conversation, ac *account.Account, text string) (wamid string, err error)
	Hook(ctx context.Context, ac *account.Account, rawData []byte, signature string) ([]*HookResult, error)
	VerifyWebhook(ctx context.Context, ac *account.Account, mode string, verifyToken string, challenge string) (string, error)
}

type whatsappHandler struct {
	reqHandler requesthandler.RequestHandler
}

// NewWhatsAppHandler returns a new WhatsAppHandler.
func NewWhatsAppHandler(reqHandler requesthandler.RequestHandler) WhatsAppHandler {
	return &whatsappHandler{reqHandler: reqHandler}
}
