package servicehandler

import (
	"context"
	"fmt"

	amai "monorepo/bin-ai-manager/models/ai"

	amagent "monorepo/bin-agent-manager/models/agent"

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
	a *amagent.Agent,
	name string,
	detail string,
	engineType amai.EngineType,
	engineModel amai.EngineModel,
	engineData map[string]any,
	initPrompt string,
) (*amai.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "AICreate",
		"customer_id":  a.CustomerID,
		"name":         name,
		"detail":       detail,
		"engine_type":  engineType,
		"engine_model": engineModel,
		"engine_data":  engineData,
		"init_prompt":  initPrompt,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.AIV1AICreate(
		ctx,
		a.CustomerID,
		name,
		detail,
		engineType,
		engineModel,
		engineData,
		initPrompt,
	)
	if err != nil {
		log.Errorf("Could not create a new ai. err: %v", err)
		return nil, err
	}
	log.WithField("ai", tmp).Debug("Created a new ai.")

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AIGetsByCustomerID gets the list of AIs of the given customer id.
// It returns list of AIs if it succeed.
func (h *serviceHandler) AIGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*amai.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AIGetsByCustomerID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	// filters
	filters := map[string]string{
		"deleted": "false", // we don't need deleted items
	}

	tmps, err := h.reqHandler.AIV1AIGetsByCustomerID(ctx, a.CustomerID, token, size, filters)
	if err != nil {
		log.Errorf("Could not get AIs info from the chatobt manager. err: %v", err)
		return nil, fmt.Errorf("could not find chats info. err: %v", err)
	}

	// create result
	res := []*amai.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// AIGet gets the AI of the given id.
// It returns AI if it succeed.
func (h *serviceHandler) AIGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*amai.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AIGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"ai_id":       id,
	})

	tmp, err := h.aiGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get ai info from the chatobt manager. err: %v", err)
		return nil, fmt.Errorf("could not find ai info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AIDelete deletes the ai.
func (h *serviceHandler) AIDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*amai.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AIDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"ai_id":       id,
	})
	log.Debug("Deleting an ai.")

	// get chat
	c, err := h.aiGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get ai info from the ai-manager. err: %v", err)
		return nil, fmt.Errorf("could not find ai info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
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
	a *amagent.Agent,
	id uuid.UUID,
	name string,
	detail string,
	engineType amai.EngineType,
	engineModel amai.EngineModel,
	engineData map[string]any,
	initPrompt string,
) (*amai.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "AIUpdate",
		"customer_id":  a.CustomerID,
		"id":           id,
		"name":         name,
		"detail":       detail,
		"engine_type":  engineType,
		"engine_model": engineModel,
		"init_prompt":  initPrompt,
	})

	// get chat
	c, err := h.aiGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get ai info from the ai-manager. err: %v", err)
		return nil, fmt.Errorf("could not find ai info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.AIV1AIUpdate(
		ctx,
		id,
		name,
		detail,
		engineType,
		engineModel,
		engineData,
		initPrompt,
	)
	if err != nil {
		log.Errorf("Could not update the ai. err: %v", err)
		return nil, err
	}
	log.WithField("ai", tmp).Debugf("Updated ai info. ai_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
