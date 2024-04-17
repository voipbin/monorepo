package servicehandler

import (
	"context"
	"fmt"

	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	cvmedia "monorepo/bin-conversation-manager/models/media"
	cvmessage "monorepo/bin-conversation-manager/models/message"

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

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// get tmp
	tmp, err := h.reqHandler.ConversationV1ConversationGetsByCustomerID(ctx, a.CustomerID, token, size)
	if err != nil {
		log.Errorf("Could not get campaigns info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaigns info. err: %v", err)
	}

	// create result
	res := []*cvconversation.WebhookMessage{}
	for _, f := range tmp {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
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

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionAll) {
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

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionAll) {
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

// ConversationMessageGetsByConversationID gets the list of conversation's messages of the given conversation id.
// It returns list of conversation messages if it succeed.
func (h *serviceHandler) ConversationMessageGetsByConversationID(
	ctx context.Context,
	a *amagent.Agent,
	conversationID uuid.UUID,
	size uint64,
	token string,
) ([]*cvmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ConversationMessageGetsByConversationID",
		"customer_id":     a.CustomerID,
		"conversation_id": conversationID,
		"username":        a.Username,
		"size":            size,
		"token":           token,
	})
	log.Debug("Getting a conversation messages.")

	// get conversation to check the permission
	c, err := h.conversationGet(ctx, a, conversationID)
	if err != nil {
		log.Errorf("Could not get conversation info. err: %v", err)
		return nil, fmt.Errorf("could not verify the conversation. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// get tmp
	tmp, err := h.reqHandler.ConversationV1ConversationMessageGetsByConversationID(ctx, conversationID, token, size)
	if err != nil {
		log.Errorf("Could not get conversation messages info from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("could not get conversation messages info. err: %v", err)
	}

	// create result
	res := []*cvmessage.WebhookMessage{}
	for _, f := range tmp {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// ConversationMessageSend send a message to the conversation.
func (h *serviceHandler) ConversationMessageSend(
	ctx context.Context,
	a *amagent.Agent,
	conversationID uuid.UUID,
	text string,
	medias []cvmedia.Media,
) (*cvmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ConversationMessageSend",
		"customer_id":     a.CustomerID,
		"conversation_id": conversationID,
	})
	log.Debugf("Sending a message. conversation_id: %s", conversationID)

	// get conversation to check the permission
	c, err := h.conversationGet(ctx, a, conversationID)
	if err != nil {
		log.Errorf("Could not get conversation info. err: %v", err)
		return nil, fmt.Errorf("could not verify the conversation. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ConversationV1MessageSend(ctx, conversationID, text, medias)
	if err != nil {
		log.Errorf("Could not send the message correctly. err: %v", err)
		return nil, fmt.Errorf("could not send the message. err: %v", err)
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
