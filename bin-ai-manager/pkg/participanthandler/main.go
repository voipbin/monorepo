package participanthandler

//go:generate mockgen -package participanthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/pkg/dbhandler"
)

// ParticipantHandler manages aicall participant records.
type ParticipantHandler interface {
	Create(ctx context.Context, aicallID uuid.UUID, aiID uuid.UUID) error
}

type participantHandler struct {
	db dbhandler.DBHandler
}

// New returns a new ParticipantHandler backed by db.
func New(db dbhandler.DBHandler) ParticipantHandler {
	return &participantHandler{db: db}
}
