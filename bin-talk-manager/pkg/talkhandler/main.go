//go:generate mockgen -package talkhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

package talkhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/notifyhandler"

	"monorepo/bin-talk-manager/models/talk"
	"monorepo/bin-talk-manager/pkg/dbhandler"
)

// TalkHandler defines the talk business logic interface
type TalkHandler interface {
	TalkCreate(ctx context.Context, customerID uuid.UUID, talkType talk.Type) (*talk.Talk, error)
	TalkGet(ctx context.Context, id uuid.UUID) (*talk.Talk, error)
	TalkList(ctx context.Context, filters map[talk.Field]any, token string, size uint64) ([]*talk.Talk, error)
	TalkDelete(ctx context.Context, id uuid.UUID) error
}

type talkHandler struct {
	dbHandler     dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
}

// New creates a new TalkHandler
func New(
	dbHandler dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
) TalkHandler {
	return &talkHandler{
		dbHandler:     dbHandler,
		notifyHandler: notifyHandler,
	}
}
