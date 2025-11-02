package aihandler

import (
	"context"
	"fmt"
	"monorepo/bin-ai-manager/models/ai"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *aiHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	engineType ai.EngineType,
	engineModel ai.EngineModel,
	engineData map[string]any,
	engineKey string,
	initPrompt string,
	ttsType ai.TTSType,
	ttsVoiceID string,
	sttType ai.STTType,
) (*ai.AI, error) {

	if !ai.IsValidEngineModel(engineModel) {
		return nil, fmt.Errorf("invalid engine model: %s", engineModel)
	}

	res, err := h.dbCreate(ctx, customerID, name, detail, engineType, engineModel, engineData, engineKey, initPrompt, ttsType, ttsVoiceID, sttType)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create ai")
	}

	return res, nil
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
	engineKey string,
	initPrompt string,
	ttsType ai.TTSType,
	ttsVoiceID string,
	sttType ai.STTType,
) (*ai.AI, error) {

	if !ai.IsValidEngineModel(engineModel) {
		return nil, fmt.Errorf("invalid engine model: %s", engineModel)
	}

	res, err := h.dbUpdate(ctx, id, name, detail, engineType, engineModel, engineData, engineKey, initPrompt, ttsType, ttsVoiceID, sttType)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update ai")
	}

	return res, nil
}
