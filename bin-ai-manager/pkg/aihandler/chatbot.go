package aihandler

import (
	"context"
	"fmt"
	"strings"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/tool"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *aiHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	engineModel ai.EngineModel,
	parameter map[string]any,
	engineKey string,
	initPrompt string,
	ttsType ai.TTSType,
	ttsVoiceID string,
	sttType ai.STTType,
	toolNames []tool.ToolName,
	vadConfig *ai.VADConfig,
) (*ai.AI, error) {

	if !ai.IsValidEngineModel(engineModel) {
		return nil, fmt.Errorf("invalid engine model: %s", engineModel)
	}

	if !ttsType.IsValid() {
		return nil, fmt.Errorf("invalid tts_type: %s. valid values: %s", ttsType, strings.Join(ttsType.ValidValues(), ", "))
	}

	if !sttType.IsValid() {
		return nil, fmt.Errorf("invalid stt_type: %s. valid values: %s", sttType, strings.Join(sttType.ValidValues(), ", "))
	}

	res, err := h.dbCreate(ctx, customerID, name, detail, engineModel, parameter, engineKey, initPrompt, ttsType, ttsVoiceID, sttType, toolNames, vadConfig)
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
	engineModel ai.EngineModel,
	parameter map[string]any,
	engineKey string,
	initPrompt string,
	ttsType ai.TTSType,
	ttsVoiceID string,
	sttType ai.STTType,
	toolNames []tool.ToolName,
	vadConfig *ai.VADConfig,
) (*ai.AI, error) {

	if !ai.IsValidEngineModel(engineModel) {
		return nil, fmt.Errorf("invalid engine model: %s", engineModel)
	}

	if !ttsType.IsValid() {
		return nil, fmt.Errorf("invalid tts_type: %s. valid values: %s", ttsType, strings.Join(ttsType.ValidValues(), ", "))
	}

	if !sttType.IsValid() {
		return nil, fmt.Errorf("invalid stt_type: %s. valid values: %s", sttType, strings.Join(sttType.ValidValues(), ", "))
	}

	res, err := h.dbUpdate(ctx, id, name, detail, engineModel, parameter, engineKey, initPrompt, ttsType, ttsVoiceID, sttType, toolNames, vadConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update ai")
	}

	return res, nil
}
