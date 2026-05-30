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
	autoAICallAuditEnabled bool,
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

	// Pre-generate the history ID so we can write it into the AI row at creation time
	var currentPromptHistoryID uuid.UUID
	if initPrompt != "" {
		currentPromptHistoryID = h.utilHandler.UUIDCreate()
	}

	res, err := h.dbCreate(ctx, customerID, name, detail, engineModel, parameter, engineKey, ragID,
		initPrompt, ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled,
		autoAICallAuditEnabled, currentPromptHistoryID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create ai")
	}

	if initPrompt != "" {
		if errHistory := h.db.AIPromptHistoryCreate(ctx, &aiprompthistory.AIPromptHistory{
			Identity: identity.Identity{
				ID:         currentPromptHistoryID,
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
	autoAICallAuditEnabled bool,
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

	// Pre-fetch unconditionally so all three branches can detect changes.
	preUpdateAI, errGet := h.db.AIGet(ctx, id)
	if errGet != nil {
		return nil, errors.Wrapf(errGet, "could not get current ai for update")
	}

	promptChanged := initPrompt != "" && initPrompt != preUpdateAI.InitPrompt
	promptCleared := initPrompt == "" && preUpdateAI.InitPrompt != ""

	switch {
	case promptChanged:
		historyID := h.utilHandler.UUIDCreate()
		fields := h.buildUpdateFields(name, detail, engineModel, parameter, engineKey, ragID, initPrompt,
			ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled, autoAICallAuditEnabled)
		fields[ai.FieldCurrentPromptHistoryID] = historyID
		if err := h.db.AIUpdate(ctx, id, fields); err != nil {
			return nil, errors.Wrapf(err, "could not update ai")
		}
		res, err := h.db.AIGet(ctx, id)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get updated ai")
		}
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, ai.EventTypeUpdated, res)
		if errHistory := h.db.AIPromptHistoryCreate(ctx, &aiprompthistory.AIPromptHistory{
			Identity: identity.Identity{
				ID:         historyID,
				CustomerID: res.CustomerID,
			},
			AIID:   id,
			Prompt: initPrompt,
		}); errHistory != nil {
			logrus.WithField("func", "Update").Errorf("Could not create prompt history. err: %v", errHistory)
		}
		return res, nil

	case promptCleared:
		fields := h.buildUpdateFields(name, detail, engineModel, parameter, engineKey, ragID, "",
			ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled, autoAICallAuditEnabled)
		fields[ai.FieldCurrentPromptHistoryID] = uuid.Nil
		if err := h.db.AIUpdate(ctx, id, fields); err != nil {
			return nil, errors.Wrapf(err, "could not update ai (clear prompt)")
		}
		res, err := h.db.AIGet(ctx, id)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get updated ai")
		}
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, ai.EventTypeUpdated, res)
		return res, nil

	default: // prompt unchanged
		return h.dbUpdate(ctx, id, name, detail, engineModel, parameter, engineKey, ragID, initPrompt,
			ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled, autoAICallAuditEnabled)
	}
}
