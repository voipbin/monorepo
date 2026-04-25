package servicehandler

import (
	"context"
	"fmt"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonaddress "monorepo/bin-common-handler/models/address"

	mmmessage "monorepo/bin-message-manager/models/message"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// messageGet validates the tag's ownership and returns the message info.
func (h *serviceHandler) messageGet(ctx context.Context, messageID uuid.UUID) (*mmmessage.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "messageGet",
		"tag_id": messageID,
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
func (h *serviceHandler) MessageList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*mmmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "MessageGets",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"size":        size,
		"token":       "token",
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// get messages
	filters := map[mmmessage.Field]any{
		mmmessage.FieldCustomerID: a.CustomerID,
	}
	tmps, err := h.reqHandler.MessageV1MessageList(ctx, token, size, filters)
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
func (h *serviceHandler) MessageSend(ctx context.Context, a *auth.AuthIdentity, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*mmmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "MessageSend",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if len(destinations) <= 0 {
		log.Errorf("The destination is empty. destinations: %d", len(destinations))
		return nil, fmt.Errorf("destination is empty")
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
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
func (h *serviceHandler) MessageGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*mmmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "MessageGet",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"message_id":  id,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// get message info
	tmp, err := h.messageGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get message info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	if tmp.TMDelete != nil {
		log.WithField("message", tmp).Debugf("Deleted message.")
		return nil, fmt.Errorf("not found")
	}

	res := tmp.ConvertWebhookMessage()

	return res, nil
}

// MessageDelete handles message delete request.
// It sends a request to the message-manager to get a existed message.
// it returns a message information if it succeed.
func (h *serviceHandler) MessageDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*mmmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "MessageDelete",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"message_id":  id,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// get message info
	m, err := h.messageGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get message info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, m.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
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
