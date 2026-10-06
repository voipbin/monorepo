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
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// errInvalidEngineModel returns the typed INVALID_ENGINE_MODEL error (HTTP 400).
// The message is customer-facing: it must stay free of internal routing terms.
func errInvalidEngineModel(engineModel ai.EngineModel) error {
	return cerrors.InvalidArgument(
		commonoutline.ServiceNameAIManager,
		"INVALID_ENGINE_MODEL",
		fmt.Sprintf("invalid engine_model: %q is not in the supported model list and does not use an allowed provider prefix. See GET /ai_models for the valid values.", engineModel),
	)
}

// errEngineKeyRequired returns the typed ENGINE_KEY_REQUIRED error (HTTP 400).
func errEngineKeyRequired() error {
	return cerrors.InvalidArgument(
		commonoutline.ServiceNameAIManager,
		"ENGINE_KEY_REQUIRED",
		"An API key is required for custom OpenRouter models.",
	)
}

// errInvalidCustomModelID returns INVALID_ENGINE_MODEL for a malformed custom model ID.
// The input is deliberately not echoed: a pasted key must never be reflected back.
func errInvalidCustomModelID() error {
	return cerrors.InvalidArgument(
		commonoutline.ServiceNameAIManager,
		"INVALID_ENGINE_MODEL",
		"invalid engine_model: the OpenRouter model ID is not valid. Use the vendor/model-name form with letters, digits, '.', '_' and '-' only. Variant suffixes such as ':free' and router IDs such as 'openrouter/auto' are not supported.",
	)
}

// engineValidationError converts an ai.ValidateEngine error to the typed customer-facing error.
func engineValidationError(engineModel ai.EngineModel, err error) error {
	switch {
	case errors.Is(err, ai.ErrEngineKeyRequired):
		return errEngineKeyRequired()
	case errors.Is(err, ai.ErrInvalidEngineModel):
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(string(engineModel))), ai.EngineModelPrefixCustomOpenRouter) {
			return errInvalidCustomModelID()
		}
		return errInvalidEngineModel(engineModel)
	}

	return err
}

func (h *aiHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	aiType ai.Type,
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

	if err := ai.ValidateEngine(engineModel, engineKey, true); err != nil {
		return nil, engineValidationError(engineModel, err)
	}

	if aiType == ai.TypeNone {
		aiType = ai.TypeNormal
	}
	if !aiType.IsValid() {
		return nil, fmt.Errorf("invalid type: %s. valid values: %s", aiType, strings.Join(aiType.ValidValues(), ", "))
	}

	if err := ai.ValidateToolNames(aiType, toolNames); err != nil {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_TOOL_NAMES", err.Error()).Wrap(err)
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

	res, err := h.dbCreate(ctx, customerID, name, detail, aiType, engineModel, parameter, engineKey, ragID,
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
	aiType ai.Type,
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

	// Only a changed engine model is validated for validity, so an unchanged legacy value keeps saving.
	// The final key state is always checked: a custom model with an empty key is rejected.
	if err := ai.ValidateEngine(engineModel, engineKey, engineModel != preUpdateAI.EngineModel); err != nil {
		return nil, engineValidationError(engineModel, err)
	}

	// A caller omitting the type field (aiType == TypeNone) means "leave it
	// unchanged", not "reset to normal" -- unlike Create, Update must not
	// silently downgrade an existing Insight AI to Normal on a partial PUT
	// that doesn't touch the type field.
	if aiType == ai.TypeNone {
		aiType = preUpdateAI.Type
	}
	if aiType == ai.TypeNone {
		aiType = ai.TypeNormal
	}
	if !aiType.IsValid() {
		return nil, fmt.Errorf("invalid type: %s. valid values: %s", aiType, strings.Join(aiType.ValidValues(), ", "))
	}

	if err := ai.ValidateToolNames(aiType, toolNames); err != nil {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_TOOL_NAMES", err.Error()).Wrap(err)
	}

	promptChanged := initPrompt != "" && initPrompt != preUpdateAI.InitPrompt
	promptCleared := initPrompt == "" && preUpdateAI.InitPrompt != ""

	switch {
	case promptChanged:
		historyID := h.utilHandler.UUIDCreate()
		fields := h.buildUpdateFields(name, detail, aiType, engineModel, parameter, engineKey, ragID, initPrompt,
			ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled, autoAICallAuditEnabled)
		fields[ai.FieldCurrentPromptHistoryID] = historyID
		if err := h.db.AIUpdate(ctx, id, fields); err != nil {
			return nil, errors.Wrapf(err, "could not update ai")
		}
		res, err := h.db.AIGet(ctx, id)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get updated ai")
		}
		h.publishAIEvent(ctx, ai.EventTypeUpdated, res)
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
		fields := h.buildUpdateFields(name, detail, aiType, engineModel, parameter, engineKey, ragID, "",
			ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled, autoAICallAuditEnabled)
		fields[ai.FieldCurrentPromptHistoryID] = uuid.Nil
		if err := h.db.AIUpdate(ctx, id, fields); err != nil {
			return nil, errors.Wrapf(err, "could not update ai (clear prompt)")
		}
		res, err := h.db.AIGet(ctx, id)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get updated ai")
		}
		h.publishAIEvent(ctx, ai.EventTypeUpdated, res)
		return res, nil

	default: // prompt unchanged
		return h.dbUpdate(ctx, id, name, detail, aiType, engineModel, parameter, engineKey, ragID, initPrompt,
			ttsType, ttsVoiceID, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled, autoAICallAuditEnabled)
	}
}
