package smshandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package smshandler -destination ./mock_smshandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/accounthandler"
)

// SMSHandler defines
type SMSHandler interface {
	Send(ctx context.Context, cv *conversation.Conversation, transactionID string, text string) error
	Event(ctx context.Context, data []byte) ([]*message.Message, *commonaddress.Address, error)
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
