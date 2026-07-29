package aihandler

import (
	"context"
	stderrors "errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

// Create creates a new ai record.
func (h *aiHandler) dbCreate(
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
	currentPromptHistoryID uuid.UUID,
) (*ai.AI, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "dbCreate",
	})

	id := h.utilHandler.UUIDCreate()

	// create direct hash via direct-manager
	d, err := h.reqHandler.DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeAI, id)
	if err != nil {
		log.Errorf("Could not create direct hash. err: %v", err)
		return nil, fmt.Errorf("could not create direct hash: %w", err)
	}
	log.WithField("direct", d).Debugf("Created direct hash. direct_id: %s", d.ID)

	c := &ai.AI{
		Identity: identity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		Name:   name,
		Detail: detail,
		Type:   aiType,

		EngineModel: engineModel,
		Parameter:   parameter,
		EngineKey:   engineKey,
		RagID:       ragID,

		InitPrompt: initPrompt,

		CurrentPromptHistoryID: currentPromptHistoryID,

		TTSType:    ttsType,
		TTSVoiceID: ttsVoiceID,

		STTType:     sttType,
		STTLanguage: sttLanguage,

		ToolNames: toolNames,

		VADConfig:        vadConfig,
		SmartTurnEnabled: smartTurnEnabled,

		AutoAICallAuditEnabled: autoAICallAuditEnabled,

		DirectID:   d.ID,
		DirectHash: d.Hash,
	}

	if err := h.db.AICreate(ctx, c); err != nil {
		// cleanup orphaned direct
		if _, errDelete := h.reqHandler.DirectV1DirectDelete(ctx, d.ID); errDelete != nil {
			log.Errorf("Could not cleanup orphaned direct. direct_id: %s, err: %v", d.ID, errDelete)
		}
		return nil, errors.Wrapf(err, "could not create ai")
	}

	res, err := h.db.AIGet(ctx, c.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get created ai")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, ai.EventTypeCreated, res)

	return res, nil
}

// Get returns ai.
func (h *aiHandler) Get(ctx context.Context, id uuid.UUID) (*ai.AI, error) {
	res, err := h.db.AIGet(ctx, id)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameAIManager,
				"AI_NOT_FOUND",
				"The AI was not found.",
			).Wrap(err)
		}
		return nil, errors.Wrapf(err, "could not get ai")
	}

	return res, nil
}

// List returns list of ais.
func (h *aiHandler) List(ctx context.Context, size uint64, token string, filters map[ai.Field]any) ([]*ai.AI, error) {
	res, err := h.db.AIList(ctx, size, token, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ais")
	}

	return res, nil
}

// Delete deletes the ai.
func (h *aiHandler) Delete(ctx context.Context, id uuid.UUID) (*ai.AI, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Delete",
	})

	// get the ai to retrieve the direct_id before deletion
	a, err := h.db.AIGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get ai for delete")
	}
	log.WithField("ai", a).Debugf("Retrieved ai info. ai_id: %s", a.ID)

	// delete direct hash via direct-manager (best-effort, don't block ai deletion)
	if a.DirectID != uuid.Nil {
		if _, errDirect := h.reqHandler.DirectV1DirectDelete(ctx, a.DirectID); errDirect != nil {
			log.Errorf("Could not delete direct hash. direct_id: %s, err: %v", a.DirectID, errDirect)
		}
	}

	if err := h.db.AIDelete(ctx, id); err != nil {
		return nil, errors.Wrapf(err, "could not delete ai")
	}

	res, err := h.db.AIGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get deleted ai")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, ai.EventTypeDeleted, res)

	return res, nil
}

// Update updates the ai info
func (h *aiHandler) dbUpdate(
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
	ttsVoice string,
	sttType ai.STTType,
	sttLanguage string,
	toolNames []tool.ToolName,
	vadConfig *ai.VADConfig,
	smartTurnEnabled bool,
	autoAICallAuditEnabled bool,
) (*ai.AI, error) {
	fields := h.buildUpdateFields(name, detail, aiType, engineModel, parameter, engineKey, ragID, initPrompt,
		ttsType, ttsVoice, sttType, sttLanguage, toolNames, vadConfig, smartTurnEnabled, autoAICallAuditEnabled)

	if err := h.db.AIUpdate(ctx, id, fields); err != nil {
		return nil, errors.Wrapf(err, "could not update ai")
	}

	res, err := h.db.AIGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated ai")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, ai.EventTypeUpdated, res)

	return res, nil
}

// ActivateInsight makes the given Insight AI the customer's active one,
// deactivating whichever Insight AI was active before.
func (h *aiHandler) ActivateInsight(ctx context.Context, id uuid.UUID) (*ai.AI, error) {
	res, previous, err := h.db.AIActivateInsight(ctx, id)
	if err != nil {
		switch {
		case stderrors.Is(err, dbhandler.ErrNotFound):
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameAIManager,
				"AI_NOT_FOUND",
				"The AI was not found.",
			).Wrap(err)

		case stderrors.Is(err, dbhandler.ErrNotInsightAI):
			return nil, cerrors.InvalidArgument(
				commonoutline.ServiceNameAIManager,
				"AI_NOT_INSIGHT_TYPE",
				"Only an Insight AI can be activated.",
			).Wrap(err)

		case dbhandler.IsErrDuplicate(err):
			// The unique index on ai_ais.active_insight_key rejected the write,
			// which means another activation for this customer committed between
			// our read and our write. Retrying resolves it.
			return nil, cerrors.AlreadyExists(
				commonoutline.ServiceNameAIManager,
				"AI_INSIGHT_ACTIVATION_CONFLICT",
				"Another activation is in progress for this customer's Insight AI; retry.",
			).Wrap(err)
		}

		return nil, errors.Wrapf(err, "could not activate insight ai")
	}

	// The displaced AI flipped is_insight_active true -> false, which is just as
	// user-visible as the activation itself. Without this event, subscribers
	// keep a stale is_insight_active=true for it forever. Published first so
	// subscribers never observe two active Insight AIs.
	if previous != nil {
		h.notifyHandler.PublishWebhookEvent(ctx, previous.CustomerID, ai.EventTypeUpdated, previous)
	}

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, ai.EventTypeUpdated, res)

	return res, nil
}

// buildUpdateFields builds the AI field map used for AIUpdate calls.
func (h *aiHandler) buildUpdateFields(
	name, detail string,
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
) map[ai.Field]any {
	res := map[ai.Field]any{
		ai.FieldName:                   name,
		ai.FieldDetail:                 detail,
		ai.FieldType:                   aiType,
		ai.FieldEngineModel:            engineModel,
		ai.FieldParameter:              parameter,
		ai.FieldEngineKey:              engineKey,
		ai.FieldRagID:                  ragID,
		ai.FieldInitPrompt:             initPrompt,
		ai.FieldTTSType:                ttsType,
		ai.FieldTTSVoiceID:             ttsVoiceID,
		ai.FieldSTTType:                sttType,
		ai.FieldSTTLanguage:            sttLanguage,
		ai.FieldToolNames:              toolNames,
		ai.FieldVADConfig:              vadConfig,
		ai.FieldSmartTurnEnabled:       smartTurnEnabled,
		ai.FieldAutoAICallAuditEnabled: autoAICallAuditEnabled,
	}

	// Any row that is not (or is no longer) an Insight AI must not keep an
	// is_insight_active flag. Written unconditionally in the same UPDATE rather
	// than "only if it was previously true": a read-then-write would race a
	// concurrent AIActivateInsight and could silently undo a just-won activation.
	if aiType != ai.TypeInsight {
		res[ai.FieldIsInsightActive] = false
	}

	return res
}
