package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	mmmessage "gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
)

// messageGet validates the tag's ownership and returns the message info.
func (h *serviceHandler) messageGet(ctx context.Context, a *amagent.Agent, messageID uuid.UUID) (*mmmessage.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "messageGet",
		"customer_id": a.CustomerID,
		"tag_id":      messageID,
	})

	// send request
	res, err := h.reqHandler.MessageV1MessageGet(ctx, messageID)
	if err != nil {
		log.Errorf("Could not get message. err: %v", err)
		return nil, err
	}
	log.WithField("tag", res).Debug("Received result.")

	// create result
	return res, nil
}

// MessageGets sends a request to getting a list of messages
// It sends a request to the message-manager to getting a list of messages.
// it returns list of messages if it succeed.
func (h *serviceHandler) MessageGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*mmmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "MessageGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       "token",
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	// get messages
	tmps, err := h.reqHandler.MessageV1MessageGets(ctx, a.CustomerID, token, size)
	if err != nil {
		log.Infof("Could not get messages info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*mmmessage.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// MessageSend handles message send request.
// It sends a request to the message-manager to create(send) a new message.
// it returns created message information if it succeed.
func (h *serviceHandler) MessageSend(ctx context.Context, a *amagent.Agent, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*mmmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "MessageSend",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	if len(destinations) <= 0 {
		log.Errorf("The destination is empty. destinations: %d", len(destinations))
		return nil, fmt.Errorf("destination is empty")
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	// send message
	tmp, err := h.reqHandler.MessageV1MessageSend(ctx, uuid.Nil, a.CustomerID, source, destinations, text)
	if err != nil {
		log.Infof("Could not get send a message info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// MessageGet handles message get request.
// It sends a request to the message-manager to get a existed message.
// it returns a message information if it succeed.
func (h *serviceHandler) MessageGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*mmmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "MessageGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"message_id":  id,
	})

	// get message info
	tmp, err := h.messageGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get message info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionAll) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	if tmp.TMDelete != defaultTimestamp {
		log.WithField("message", tmp).Debugf("Deleted message.")
		return nil, fmt.Errorf("not found")
	}

	res := tmp.ConvertWebhookMessage()

	return res, nil
}

// MessageDelete handles message delete request.
// It sends a request to the message-manager to get a existed message.
// it returns a message information if it succeed.
func (h *serviceHandler) MessageDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*mmmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "MessageDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"message_id":  id,
	})

	// get message info
	m, err := h.messageGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get message info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, m.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	// delete message info
	tmp, err := h.reqHandler.MessageV1MessageDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not get message info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
