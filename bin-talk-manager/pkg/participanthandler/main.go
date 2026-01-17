package participanthandler

import (
	"context"

	"github.com/gofrs/uuid"

	commonnotify "monorepo/bin-common-handler/pkg/notifyhandler"
	commonutil "monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-talk-manager/models/participant"
)

//go:generate mockgen -package participanthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

// ParticipantHandler defines the interface for participant business logic operations
type ParticipantHandler interface {
	// ParticipantAdd adds a participant to a chat (upsert behavior)
	ParticipantAdd(ctx context.Context, customerID, chatID, ownerID uuid.UUID, ownerType string) (*participant.Participant, error)

	// ParticipantList returns all participants for a talk
	ParticipantList(ctx context.Context, customerID, chatID uuid.UUID) ([]*participant.Participant, error)

	// ParticipantListWithFilters returns participants matching filter criteria
	ParticipantListWithFilters(ctx context.Context, filters map[participant.Field]any, token string, size uint64) ([]*participant.Participant, error)

	// ParticipantRemove removes a participant from a chat (hard delete)
	ParticipantRemove(ctx context.Context, customerID, participantID uuid.UUID) error
}

// participantHandler implements ParticipantHandler interface
type participantHandler struct {
	dbHandler     DBHandler
	sockHandler   SockHandler
	notifyHandler commonnotify.NotifyHandler
	utilHandler   commonutil.UtilHandler
}

// New creates a new ParticipantHandler instance
func New(dbHandler DBHandler, sockHandler SockHandler, notifyHandler commonnotify.NotifyHandler, utilHandler commonutil.UtilHandler) ParticipantHandler {
	return &participantHandler{
		dbHandler:     dbHandler,
		sockHandler:   sockHandler,
		notifyHandler: notifyHandler,
		utilHandler:   utilHandler,
	}
}

// DBHandler defines the interface for database operations
type DBHandler interface {
	ParticipantCreate(ctx context.Context, p *participant.Participant) error
	ParticipantGet(ctx context.Context, id uuid.UUID) (*participant.Participant, error)
	ParticipantList(ctx context.Context, filters map[participant.Field]any) ([]*participant.Participant, error)
	ParticipantDelete(ctx context.Context, id uuid.UUID) error
}

// SockHandler defines the interface for RabbitMQ socket operations
type SockHandler interface {
	// Add methods as needed for inter-service communication
}
