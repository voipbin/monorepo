package linehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package linehandler -destination ./mock_linehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/accounthandler"
)

// LineHandler defines
type LineHandler interface {
	Setup(ctx context.Context, customerID uuid.UUID) error
	Send(ctx context.Context, cv *conversation.Conversation, text string, medias []media.Media) error
	Hook(ctx context.Context, customerID uuid.UUID, data []byte) ([]*conversation.Conversation, []*message.Message, error)

	GetParticipant(ctx context.Context, customerID uuid.UUID, id string) (*commonaddress.Address, error)
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
