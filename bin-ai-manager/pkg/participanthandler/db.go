package participanthandler

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/participant"
)

func (h *participantHandler) Create(ctx context.Context, aicallID uuid.UUID, aiID uuid.UUID) error {
	return h.db.ParticipantCreate(ctx, aicallID, aiID)
}

func (h *participantHandler) ListByAIcallID(ctx context.Context, aicallID uuid.UUID, size uint64, token string) ([]*participant.Participant, error) {
	return h.db.ParticipantListByAIcallID(ctx, aicallID, size, token)
}

func (h *participantHandler) ListByAIID(ctx context.Context, aiID uuid.UUID, size uint64, token string) ([]*participant.Participant, error) {
	return h.db.ParticipantListByAIID(ctx, aiID, size, token)
}
