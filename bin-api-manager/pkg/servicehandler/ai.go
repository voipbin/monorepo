package servicehandler

import (
	"context"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	amai "monorepo/bin-ai-manager/models/ai"
	amtool "monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// aiGet returns the ai info.
func (h *serviceHandler) aiGet(ctx context.Context, id uuid.UUID) (*amai.AI, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "aiGet",
		"ai_id": id,
	})

	// send request
	res, err := h.reqHandler.AIV1AIGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the resource info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// AICreate is a service handler for AI creation.
func (h *serviceHandler) AICreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	name string,
	detail string,
	aiType amai.Type,
	engineModel amai.EngineModel,
	parameter map[string]any,
	engineKey string,
	ragID uuid.UUID,
	initPrompt string,
	ttsType amai.TTSType,
	ttsVoiceID string,
	sttType amai.STTType,
	sttLanguage string,
	toolNames []amtool.ToolName,
	autoAICallAuditEnabled bool,
) (*amai.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":         "AICreate",
		"customer_id":  a.CustomerID,
		"name":         name,
		"detail":       detail,
		"engine_model": engineModel,
		"parameter":    parameter,
		"engine_key":   engineKey,
		"rag_id":       ragID,
		"init_prompt":  initPrompt,
		"tts_type":     ttsType,
		"tts_voice_id": ttsVoiceID,
		"stt_type":     sttType,
		"stt_language": sttLanguage,
		"tool_names":   toolNames,

		"auto_aicall_audit_enabled": autoAICallAuditEnabled,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// validate RAG ownership if ragID is provided
	if ragID != uuid.Nil {
		rag, err := h.reqHandler.RagV1RagGet(ctx, ragID)
		if err != nil {
			log.Errorf("Could not get RAG. err: %v", err)
			return nil, fmt.Errorf("%w: could not validate knowledge base", serviceerrors.ErrInternal)
		}
		log.WithField("rag", rag).Debugf("Retrieved RAG info. rag_id: %s", rag.ID)

		if rag.CustomerID != a.CustomerID {
			log.Infof("RAG customer_id mismatch. rag_customer_id: %s, agent_customer_id: %s", rag.CustomerID, a.CustomerID)
			return nil, fmt.Errorf("%w: knowledge base does not belong to this customer", serviceerrors.ErrPermissionDenied)
		}
	}

	tmp, err := h.reqHandler.AIV1AICreate(
		ctx,
		a.CustomerID,
		name,
		detail,
		aiType,
		engineModel,
		parameter,
		engineKey,
		ragID,
		initPrompt,
		ttsType,
		ttsVoiceID,
		sttType,
		sttLanguage,
		toolNames,
		autoAICallAuditEnabled,
	)
	if err != nil {
		log.Errorf("Could not create a new ai. err: %v", err)
		return nil, err
	}
	log.WithField("ai", tmp).Debug("Created a new ai.")

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AIDirectHashRegenerate regenerates the direct hash for the AI.
func (h *serviceHandler) AIDirectHashRegenerate(ctx context.Context, a *auth.AuthIdentity, aiID uuid.UUID) (*amai.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "AIDirectHashRegenerate",
		"customer_id": a.CustomerID,
		"ai_id":       aiID,
	})
	log.Debug("Regenerating AI direct hash.")

	c, err := h.aiGet(ctx, aiID)
	if err != nil {
		log.Errorf("Could not get ai info. err: %v", err)
		return nil, fmt.Errorf("%w: could not find ai info", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.AIV1AIDirectHashRegenerate(ctx, aiID)
	if err != nil {
		log.Errorf("Could not regenerate AI direct hash. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AIGetsByCustomerID gets the list of AIs of the given customer id.
// It returns list of AIs if it succeed.
func (h *serviceHandler) AIGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*amai.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "AIGetsByCustomerID",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// filters
	filters := map[string]string{
		"deleted":     "false", // we don't need deleted items
		"customer_id": a.CustomerID.String(),
	}

	// Convert string filters to typed filters
	typedFilters, err := h.convertAIFilters(filters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return nil, err
	}

	tmps, err := h.reqHandler.AIV1AIList(ctx, token, size, typedFilters)
	if err != nil {
		log.Errorf("Could not get AIs info from the chatobt manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find chats info", err)
	}

	// create result
	res := []*amai.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// convertAIFilters converts map[string]string to map[amai.Field]any
func (h *serviceHandler) convertAIFilters(filters map[string]string) (map[amai.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, amai.AI{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[amai.Field]any, len(typed))
	for k, v := range typed {
		result[amai.Field(k)] = v
	}

	return result, nil
}

// AIGet gets the AI of the given id.
// It returns AI if it succeed.
func (h *serviceHandler) AIGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amai.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "AIGet",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"ai_id":       id,
	})

	tmp, err := h.aiGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get ai info from the chatobt manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find ai info", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AIDelete deletes the ai.
func (h *serviceHandler) AIDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amai.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "AIDelete",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"ai_id":       id,
	})
	log.Debug("Deleting an ai.")

	// get chat
	c, err := h.aiGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get ai info from the ai-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find ai info", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.AIV1AIDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the ai. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AIUpdate is a service handler for ai update.
func (h *serviceHandler) AIUpdate(
	ctx context.Context,
	a *auth.AuthIdentity,
	id uuid.UUID,
	name string,
	detail string,
	aiType amai.Type,
	engineModel amai.EngineModel,
	parameter map[string]any,
	engineKey string,
	ragID uuid.UUID,
	initPrompt string,
	ttsType amai.TTSType,
	ttsVoiceID string,
	sttType amai.STTType,
	sttLanguage string,
	toolNames []amtool.ToolName,
	autoAICallAuditEnabled bool,
) (*amai.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":         "AIUpdate",
		"customer_id":  a.CustomerID,
		"id":           id,
		"name":         name,
		"detail":       detail,
		"engine_model": engineModel,
		"parameter":    parameter,
		"engine_key":   engineKey,
		"rag_id":       ragID,
		"init_prompt":  initPrompt,
		"tts_type":     ttsType,
		"tts_voice_id": ttsVoiceID,
		"stt_type":     sttType,
		"stt_language": sttLanguage,
		"tool_names":   toolNames,

		"auto_aicall_audit_enabled": autoAICallAuditEnabled,
	})

	// get chat
	c, err := h.aiGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get ai info from the ai-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find ai info", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// validate RAG ownership if ragID is provided
	if ragID != uuid.Nil {
		rag, err := h.reqHandler.RagV1RagGet(ctx, ragID)
		if err != nil {
			log.Errorf("Could not get RAG. err: %v", err)
			return nil, fmt.Errorf("%w: could not validate knowledge base", serviceerrors.ErrInternal)
		}
		log.WithField("rag", rag).Debugf("Retrieved RAG info. rag_id: %s", rag.ID)

		if rag.CustomerID != a.CustomerID {
			log.Infof("RAG customer_id mismatch. rag_customer_id: %s, agent_customer_id: %s", rag.CustomerID, a.CustomerID)
			return nil, fmt.Errorf("%w: knowledge base does not belong to this customer", serviceerrors.ErrPermissionDenied)
		}
	}

	tmp, err := h.reqHandler.AIV1AIUpdate(
		ctx,
		id,
		name,
		detail,
		aiType,
		engineModel,
		parameter,
		engineKey,
		ragID,
		initPrompt,
		ttsType,
		ttsVoiceID,
		sttType,
		sttLanguage,
		toolNames,
		autoAICallAuditEnabled,
	)
	if err != nil {
		log.Errorf("Could not update the ai. err: %v", err)
		return nil, err
	}
	log.WithField("ai", tmp).Debugf("Updated ai info. ai_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
