//go:generate mockgen -package chathandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

package chathandler

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/notifyhandler"

	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/pkg/dbhandler"
)

// ChatHandler defines the chat business logic interface
type ChatHandler interface {
	ChatCreate(ctx context.Context, customerID uuid.UUID, chatType chat.Type) (*chat.Chat, error)
	ChatGet(ctx context.Context, id uuid.UUID) (*chat.Chat, error)
	ChatList(ctx context.Context, filters map[chat.Field]any, token string, size uint64) ([]*chat.Chat, error)
	ChatDelete(ctx context.Context, id uuid.UUID) (*chat.Chat, error)
}

type chatHandler struct {
	dbHandler     dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
}

// New creates a new ChatHandler
func New(
	dbHandler dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
) ChatHandler {
	return &chatHandler{
		dbHandler:     dbHandler,
		notifyHandler: notifyHandler,
	}
}
