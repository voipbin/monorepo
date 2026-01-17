//go:generate mockgen -package reactionhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

package reactionhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/pkg/dbhandler"
	commonsock "monorepo/bin-common-handler/pkg/sockhandler"
	commonnotify "monorepo/bin-common-handler/pkg/notifyhandler"
)

// ReactionHandler defines business logic for reactions
type ReactionHandler interface {
	ReactionAdd(ctx context.Context, messageID uuid.UUID, emoji, ownerType string, ownerID uuid.UUID) (*message.Message, error)
	ReactionRemove(ctx context.Context, messageID uuid.UUID, emoji, ownerType string, ownerID uuid.UUID) (*message.Message, error)
}

type reactionHandler struct {
	dbHandler     dbhandler.DBHandler
	sockHandler   commonsock.SockHandler
	notifyHandler commonnotify.NotifyHandler
}

// New creates a new ReactionHandler
func New(db dbhandler.DBHandler, sock commonsock.SockHandler, notify commonnotify.NotifyHandler) ReactionHandler {
	return &reactionHandler{
		dbHandler:     db,
		sockHandler:   sock,
		notifyHandler: notify,
	}
}
