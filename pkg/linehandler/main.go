package linehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package linehandler -destination ./mock_linehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/accounthandler"
)

// LineHandler defines
type LineHandler interface {
	Setup(ctx context.Context, customerID uuid.UUID) error
	Send(ctx context.Context, customerID uuid.UUID, destination string, text string, medias []media.Media) error
	Event(ctx context.Context, customerID uuid.UUID, data []byte) ([]*conversation.Conversation, []*message.Message, error)
}

// lineHandler defines
type lineHandler struct {
	accountHandler accounthandler.AccountHandler
}

// NewLineHandler defines
func NewLineHandler(accountHandler accounthandler.AccountHandler) LineHandler {

	return &lineHandler{
		accountHandler: accountHandler,
	}
}
