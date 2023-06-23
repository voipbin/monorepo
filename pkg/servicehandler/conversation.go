package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	cvconversation "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	cvmedia "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	cvmessage "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
)

// conversationGet validates the conversation's ownership and returns the conversation info.
func (h *serviceHandler) conversationGet(ctx context.Context, u *cscustomer.Customer, conversationID uuid.UUID) (*cvconversation.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conversationGet",
		"customer_id":     u.ID,
		"conversation_id": conversationID,
	})

	// send request
	tmp, err := h.reqHandler.ConversationV1ConversationGet(ctx, conversationID)
	if err != nil {
		log.Errorf("Could not get the conversation info. err: %v", err)
		return nil, err
	}
	log.WithField("conversation", tmp).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmp.CustomerID {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConversationGetsByCustomerID gets the list of conversations of the given customer id.
// It returns list of conversations if it succeed.
func (h *serviceHandler) ConversationGetsByCustomerID(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*cvconversation.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConversationGetsByCustomerID",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a conversations.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// get tmp
	tmp, err := h.reqHandler.ConversationV1ConversationGetsByCustomerID(ctx, u.ID, token, size)
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
func (h *serviceHandler) ConversationGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*cvconversation.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ConversationGet",
		"customer_id":     u.ID,
		"username":        u.Username,
		"conversation_id": id,
	})
	log.Debug("Getting an conversation.")

	// get campaign
	res, err := h.conversationGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get conversation info from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("could not find conversation info. err: %v", err)
	}

	return res, nil
}

// ConversationUpdate update the conversation of the given id.
// It returns updated conversation if it succeed.
func (h *serviceHandler) ConversationUpdate(ctx context.Context, u *cscustomer.Customer, conversationID uuid.UUID, name string, detail string) (*cvconversation.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ConversationUpdate",
		"customer_id":     u.ID,
		"username":        u.Username,
		"conversation_id": conversationID,
	})
	log.Debug("Updating the conversation.")

	// get campaign
	_, err := h.conversationGet(ctx, u, conversationID)
	if err != nil {
		log.Errorf("Could not get conversation info from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("could not find conversation info. err: %v", err)
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
	u *cscustomer.Customer,
	conversationID uuid.UUID,
	size uint64,
	token string,
) ([]*cvmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ConversationMessageGetsByConversationID",
		"customer_id":     u.ID,
		"conversation_id": conversationID,
		"username":        u.Username,
		"size":            size,
		"token":           token,
	})
	log.Debug("Getting a conversation messages.")

	// get conversation to check the permission
	_, err := h.conversationGet(ctx, u, conversationID)
	if err != nil {
		log.Errorf("Could not get conversation info. err: %v", err)
		return nil, fmt.Errorf("could not verify the conversation. err: %v", err)
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
	u *cscustomer.Customer,
	conversationID uuid.UUID,
	text string,
	medias []cvmedia.Media,
) (*cvmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ConversationMessageSend",
		"customer_id":     u.ID,
		"conversation_id": conversationID,
	})
	log.Debugf("Sending a message. conversation_id: %s", conversationID)

	// get conversation to check the permission
	_, err := h.conversationGet(ctx, u, conversationID)
	if err != nil {
		log.Errorf("Could not get conversation info. err: %v", err)
		return nil, fmt.Errorf("could not verify the conversation. err: %v", err)
	}

	tmp, err := h.reqHandler.ConversationV1MessageSend(ctx, conversationID, text, medias)
	if err != nil {
		log.Errorf("Could not send the message correctly. err: %v", err)
		return nil, fmt.Errorf("could not send the message. err: %v", err)
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
