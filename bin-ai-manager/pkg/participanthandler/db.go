package participanthandler

import (
	"context"

	"github.com/gofrs/uuid"
)

func (h *participantHandler) Create(ctx context.Context, aicallID uuid.UUID, aiID uuid.UUID) error {
	return h.db.ParticipantCreate(ctx, aicallID, aiID)
}
