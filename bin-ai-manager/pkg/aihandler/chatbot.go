package aihandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aiprompthistory"
	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-common-handler/models/identity"
)

func (h *aiHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	engineModel ai.EngineModel,
	parameter map[string]any,
	engineKey string,
	ragID uuid.UUID,
	initPrompt string,
	ttsType ai.TTSType,
	ttsVoiceID string,
	sttType ai.STTType,
	sttLanguage string,
	toolNames []tool.ToolName,
	vadConfig *ai.VADConfig,
	smartTurnEnabled bool,
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

	if err := vadConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid vad_config: %w", err)
	}

	res, err := h.dbCreate(ctx, customerID, name, detail, engineModel, parameter, engineKey, ragID, initPrompt, ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create ai")
	}

	if initPrompt != "" {
		if errHistory := h.db.AIPromptHistoryCreate(ctx, &aiprompthistory.AIPromptHistory{
			Identity: identity.Identity{
				ID:         h.utilHandler.UUIDCreate(),
				CustomerID: res.CustomerID,
			},
			AIID:   res.ID,
			Prompt: initPrompt,
		}); errHistory != nil {
			logrus.WithField("func", "Create").Errorf("Could not create prompt history. err: %v", errHistory)
		}
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
	ragID uuid.UUID,
	initPrompt string,
	ttsType ai.TTSType,
	ttsVoiceID string,
	sttType ai.STTType,
	sttLanguage string,
	toolNames []tool.ToolName,
	vadConfig *ai.VADConfig,
	smartTurnEnabled bool,
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

	if err := vadConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid vad_config: %w", err)
	}

	// Pre-fetch the current AI to detect prompt change (only when initPrompt is non-empty)
	var preUpdateAI *ai.AI
	if initPrompt != "" {
		var errGet error
		preUpdateAI, errGet = h.db.AIGet(ctx, id)
		if errGet != nil {
			return nil, errors.Wrapf(errGet, "could not get current ai for prompt history")
		}
	}

	res, err := h.dbUpdate(ctx, id, name, detail, engineModel, parameter, engineKey, ragID, initPrompt, ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update ai")
	}

	// Record history if prompt changed
	if preUpdateAI != nil && initPrompt != preUpdateAI.InitPrompt {
		if errHistory := h.db.AIPromptHistoryCreate(ctx, &aiprompthistory.AIPromptHistory{
			Identity: identity.Identity{
				ID:         h.utilHandler.UUIDCreate(),
				CustomerID: preUpdateAI.CustomerID,
			},
			AIID:   preUpdateAI.ID,
			Prompt: initPrompt,
		}); errHistory != nil {
			logrus.WithField("func", "Update").Errorf("Could not create prompt history. err: %v", errHistory)
		}
	}

	return res, nil
}
