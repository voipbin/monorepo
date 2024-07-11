package servicehandler

import (
	"context"
	"fmt"

	cvconversation "monorepo/bin-conversation-manager/models/conversation"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// conversationGet validates the conversation's ownership and returns the conversation info.
func (h *serviceHandler) conversationGet(ctx context.Context, a *amagent.Agent, conversationID uuid.UUID) (*cvconversation.Conversation, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conversationGet",
		"customer_id":     a.CustomerID,
		"conversation_id": conversationID,
	})

	// send request
	res, err := h.reqHandler.ConversationV1ConversationGet(ctx, conversationID)
	if err != nil {
		log.Errorf("Could not get the conversation info. err: %v", err)
		return nil, err
	}
	log.WithField("conversation", res).Debug("Received result.")

	return res, nil
}

// ConversationGetsByCustomerID gets the list of conversations of the given customer id.
// It returns list of conversations if it succeed.
func (h *serviceHandler) ConversationGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cvconversation.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConversationGetsByCustomerID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a conversations.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerManager|amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	filters := map[string]string{
		"deleted":     "false",
		"customer_id": a.CustomerID.String(),
	}

	tmps, err := h.conversationGets(ctx, a, size, token, filters)
	if err != nil {
		log.Errorf("Could not get conversations. err: %v", err)
		return nil, errors.Wrapf(err, "Could not get conversations.")
	}

	// create result
	res := []*cvconversation.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// conversationGets gets the list of conversations.
// It returns list of conversations if it succeed.
func (h *serviceHandler) conversationGets(ctx context.Context, a *amagent.Agent, size uint64, token string, filters map[string]string) ([]cvconversation.Conversation, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "ConversationGetsByCustomerID",
		"agent": a,
		"size":  size,
		"token": token,
	})
	log.Debug("Getting a conversations.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// gets
	res, err := h.reqHandler.ConversationV1ConversationGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get campaigns info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaigns info. err: %v", err)
	}

	return res, nil
}

// ConversationGet gets the conversation of the given id.
// It returns conversation if it succeed.
func (h *serviceHandler) ConversationGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cvconversation.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ConversationGet",
		"customer_id":     a.CustomerID,
		"username":        a.Username,
		"conversation_id": id,
	})
	log.Debug("Getting an conversation.")

	// get campaign
	tmp, err := h.conversationGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get conversation info from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("could not find conversation info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerManager|amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConversationUpdate update the conversation of the given id.
// It returns updated conversation if it succeed.
func (h *serviceHandler) ConversationUpdate(ctx context.Context, a *amagent.Agent, conversationID uuid.UUID, name string, detail string) (*cvconversation.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ConversationUpdate",
		"customer_id":     a.CustomerID,
		"username":        a.Username,
		"conversation_id": conversationID,
	})
	log.Debug("Updating the conversation.")

	// get campaign
	c, err := h.conversationGet(ctx, a, conversationID)
	if err != nil {
		log.Errorf("Could not get conversation info from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("could not find conversation info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerManager|amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ConversationV1ConversationUpdate(ctx, conversationID, name, detail)
	if err != nil {
		log.Errorf("Could not update the conversation. err: %v", err)
		return nil, errors.Wrap(err, "could not update the conversation")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
