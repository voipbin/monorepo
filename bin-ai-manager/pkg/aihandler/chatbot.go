package aihandler

import (
	"context"
	"fmt"
	"monorepo/bin-ai-manager/models/ai"

	"github.com/gofrs/uuid"
)

func (h *aiHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	engineType ai.EngineType,
	engineModel ai.EngineModel,
	engineData map[string]any,
	initPrompt string,
) (*ai.AI, error) {

	target := ai.GetEngineModelTarget(engineModel)
	if target == ai.EngineModelTargetNone {
		return nil, fmt.Errorf("invalid engine model: %s", engineModel)
	}

	return h.dbCreate(ctx, customerID, name, detail, engineType, engineModel, engineData, initPrompt)
}

// Update updates the ai info
func (h *aiHandler) Update(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	engineType ai.EngineType,
	engineModel ai.EngineModel,
	engineData map[string]any,
	initPrompt string,
) (*ai.AI, error) {

	target := ai.GetEngineModelTarget(engineModel)
	if target == ai.EngineModelTargetNone {
		return nil, fmt.Errorf("invalid engine model: %s", engineModel)
	}

	return h.dbUpdate(ctx, id, name, detail, engineType, engineModel, engineData, initPrompt)
}
