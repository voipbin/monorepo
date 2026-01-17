//go:generate mockgen -package chathandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

package chathandler

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/pkg/dbhandler"
	"monorepo/bin-talk-manager/pkg/participanthandler"
)

// ChatHandler defines the chat business logic interface
type ChatHandler interface {
	ChatCreate(ctx context.Context, customerID uuid.UUID, chatType chat.Type, name string, detail string, creatorType string, creatorID uuid.UUID) (*chat.Chat, error)
	ChatGet(ctx context.Context, id uuid.UUID) (*chat.Chat, error)
	ChatList(ctx context.Context, filters map[chat.Field]any, token string, size uint64) ([]*chat.Chat, error)
	ChatDelete(ctx context.Context, id uuid.UUID) (*chat.Chat, error)
}

type chatHandler struct {
	dbHandler          dbhandler.DBHandler
	participantHandler participanthandler.ParticipantHandler
	notifyHandler      notifyhandler.NotifyHandler
	utilHandler        utilhandler.UtilHandler
}

// New creates a new ChatHandler
func New(
	dbHandler dbhandler.DBHandler,
	participantHandler participanthandler.ParticipantHandler,
	notifyHandler notifyhandler.NotifyHandler,
	utilHandler utilhandler.UtilHandler,
) ChatHandler {
	return &chatHandler{
		dbHandler:          dbHandler,
		participantHandler: participantHandler,
		notifyHandler:      notifyHandler,
		utilHandler:        utilHandler,
	}
}
