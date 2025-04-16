package linehandler

//go:generate mockgen -package linehandler -destination ./mock_linehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
)

// LineHandler defines
type LineHandler interface {
	Setup(ctx context.Context, ac *account.Account) error
	Send(ctx context.Context, cv *conversation.Conversation, ac *account.Account, text string, medias []media.Media) error
	Hook(ctx context.Context, ac *account.Account, data []byte) error

	GetPeer(ctx context.Context, ac *account.Account, userID string) (*commonaddress.Address, error)
}

// lineHandler defines
type lineHandler struct {
	reqHandler requesthandler.RequestHandler
}

// NewLineHandler defines
func NewLineHandler(requestHandler requesthandler.RequestHandler) LineHandler {

	return &lineHandler{
		reqHandler: requestHandler,
	}
}
