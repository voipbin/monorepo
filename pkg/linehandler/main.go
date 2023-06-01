package linehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package linehandler -destination ./mock_linehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
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
