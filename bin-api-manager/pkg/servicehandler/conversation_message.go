package servicehandler

import (
	"context"
	"fmt"

	cvmedia "monorepo/bin-conversation-manager/models/media"
	cvmessage "monorepo/bin-conversation-manager/models/message"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

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
	c, err := h.conversationGet(ctx, conversationID)
	if err != nil {
		log.Errorf("Could not get conversation info. err: %v", err)
		return nil, fmt.Errorf("could not verify the conversation. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmps, err := h.conversationMessageGetsByConversationID(ctx, a, conversationID, size, token)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get conversation messages.")
	}

	// create result
	res := []*cvmessage.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// ConversationMessageGetsByConversationID gets the list of conversation's messages of the given conversation id.
// It returns list of conversation messages if it succeed.
func (h *serviceHandler) conversationMessageGetsByConversationID(
	ctx context.Context,
	a *amagent.Agent,
	conversationID uuid.UUID,
	size uint64,
	token string,
) ([]cvmessage.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conversationMessageGetsByConversationID",
		"customer_id":     a.CustomerID,
		"conversation_id": conversationID,
		"username":        a.Username,
		"size":            size,
		"token":           token,
	})
	log.Debug("Getting a conversation messages.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	filters := map[cvmessage.Field]any{
		cvmessage.FieldDeleted:        false,
		cvmessage.FieldConversationID: conversationID,
	}

	tmps, err := h.reqHandler.ConversationV1MessageGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get conversation messages info from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("could not get conversation messages info. err: %v", err)
	}

	return tmps, nil
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
	c, err := h.conversationGet(ctx, conversationID)
	if err != nil {
		log.Errorf("Could not get conversation info. err: %v", err)
		return nil, fmt.Errorf("could not verify the conversation. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.conversationMessageSend(ctx, a, conversationID, text, medias)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not send the message.")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// conversationMessageSend send a message to the conversation.
func (h *serviceHandler) conversationMessageSend(
	ctx context.Context,
	a *amagent.Agent,
	conversationID uuid.UUID,
	text string,
	medias []cvmedia.Media,
) (*cvmessage.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conversationMessageSend",
		"agent":           a,
		"conversation_id": conversationID,
		"text":            text,
		"medias":          medias,
	})
	log.Debugf("Sending a message. conversation_id: %s", conversationID)

	res, err := h.reqHandler.ConversationV1MessageSend(ctx, conversationID, text, medias)
	if err != nil {
		log.Errorf("Could not send the message correctly. err: %v", err)
		return nil, fmt.Errorf("could not send the message. err: %v", err)
	}

	return res, nil
}
