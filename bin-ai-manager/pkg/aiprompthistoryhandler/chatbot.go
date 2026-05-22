package aiprompthistoryhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/aiprompthistory"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

// List returns history entries for the given AI in reverse chronological order.
func (h *aiprompthistoryHandler) List(ctx context.Context, aiID uuid.UUID, size uint64, token string) ([]*aiprompthistory.AIPromptHistory, error) {
	return h.db.AIPromptHistoryGetsByAIID(ctx, aiID, size, token)
}

// Get returns a single history entry, verifying it belongs to aiID.
func (h *aiprompthistoryHandler) Get(ctx context.Context, aiID uuid.UUID, historyID uuid.UUID) (*aiprompthistory.AIPromptHistory, error) {
	res, err := h.db.AIPromptHistoryGet(ctx, historyID)
	if err != nil {
		return nil, err
	}

	if res.AIID != aiID {
		return nil, dbhandler.ErrNotFound
	}

	return res, nil
}
