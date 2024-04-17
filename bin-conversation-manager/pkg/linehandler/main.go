package linehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package linehandler -destination ./mock_linehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
)

// LineHandler defines
type LineHandler interface {
	Setup(ctx context.Context, ac *account.Account) error
	Send(ctx context.Context, cv *conversation.Conversation, ac *account.Account, text string, medias []media.Media) error
	Hook(ctx context.Context, ac *account.Account, data []byte) ([]*conversation.Conversation, []*message.Message, error)

	GetParticipant(ctx context.Context, ac *account.Account, id string) (*commonaddress.Address, error)
}

// lineHandler defines
type lineHandler struct {
}

// NewLineHandler defines
func NewLineHandler() LineHandler {

	return &lineHandler{}
}
