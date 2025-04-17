package smshandler

//go:generate mockgen -package smshandler -destination ./mock_smshandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/accounthandler"

	"github.com/gofrs/uuid"
)

// SMSHandler defines
type SMSHandler interface {
	Send(ctx context.Context, cv *conversation.Conversation, messageID uuid.UUID, text string) error
}

// smsHandler defines
type smsHandler struct {
	reqHandler     requesthandler.RequestHandler
	accountHandler accounthandler.AccountHandler
}

// NewSMSHandler defines
func NewSMSHandler(reqHandler requesthandler.RequestHandler, accountHandler accounthandler.AccountHandler) SMSHandler {

	return &smsHandler{
		reqHandler:     reqHandler,
		accountHandler: accountHandler,
	}
}
