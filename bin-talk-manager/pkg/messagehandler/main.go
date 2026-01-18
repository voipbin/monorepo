package messagehandler

//go:generate mockgen -package messagehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	commonutil "monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/pkg/dbhandler"
)

// MessageHandler defines the interface for message business logic
type MessageHandler interface {
	MessageCreate(ctx context.Context, req MessageCreateRequest) (*message.Message, error)
	MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error)
	MessageList(ctx context.Context, filters map[message.Field]any, token string, size uint64) ([]*message.Message, error)
	MessageDelete(ctx context.Context, id uuid.UUID) (*message.Message, error)
}

// MessageCreateRequest defines the input for creating a message
type MessageCreateRequest struct {
	CustomerID uuid.UUID  `json:"customer_id"`
	ChatID     uuid.UUID  `json:"chat_id"`
	ParentID   *uuid.UUID `json:"parent_id"` // Optional - for threaded messages
	OwnerType  string     `json:"owner_type"`
	OwnerID    uuid.UUID  `json:"owner_id"`
	Type       string     `json:"type"`
	Text       string     `json:"text"`
	Medias     string     `json:"medias"` // JSON string
}

type messageHandler struct {
	dbHandler     dbhandler.DBHandler
	sockHandler   sockhandler.SockHandler
	notifyHandler notifyhandler.NotifyHandler
	utilHandler   commonutil.UtilHandler
}

// New creates a new MessageHandler instance
func New(
	dbHandler dbhandler.DBHandler,
	sockHandler sockhandler.SockHandler,
	notifyHandler notifyhandler.NotifyHandler,
	utilHandler commonutil.UtilHandler,
) MessageHandler {
	return &messageHandler{
		dbHandler:     dbHandler,
		sockHandler:   sockHandler,
		notifyHandler: notifyHandler,
		utilHandler:   utilHandler,
	}
}
