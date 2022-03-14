package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	mmmessage "gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
)

// messageGet validates the tag's ownership and returns the message info.
func (h *serviceHandler) messageGet(ctx context.Context, u *cscustomer.Customer, messageID uuid.UUID) (*mmmessage.Message, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "messageGet",
			"customer_id": u.ID,
			"tag_id":      messageID,
		},
	)

	// send request
	res, err := h.reqHandler.MMV1MessageGet(ctx, messageID)
	if err != nil {
		log.Errorf("Could not get message. err: %v", err)
		return nil, err
	}
	log.WithField("tag", res).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != res.CustomerID {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	return res, nil
}

// MessageGets sends a request to getting a list of messages
// It sends a request to the message-manager to getting a list of messages.
// it returns list of messages if it succeed.
func (h *serviceHandler) MessageGets(u *cscustomer.Customer, size uint64, token string) ([]*mmmessage.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "MessageGets",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       "token",
	})

	// get messages
	tmps, err := h.reqHandler.MMV1MessageGets(ctx, u.ID, token, size)
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
func (h *serviceHandler) MessageSend(u *cscustomer.Customer, source *cmaddress.Address, destinations []cmaddress.Address, text string) (*mmmessage.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "MessageSend",
		"customer_id": u.ID,
		"username":    u.Username,
	})

	if len(destinations) <= 0 {
		log.Errorf("The destination is empty. destinations: %d", len(destinations))
		return nil, fmt.Errorf("destination is empty")
	}

	// send message
	tmp, err := h.reqHandler.MMV1MessageSend(ctx, u.ID, source, destinations, text)
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
func (h *serviceHandler) MessageGet(u *cscustomer.Customer, id uuid.UUID) (*mmmessage.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "MessageGet",
		"customer_id": u.ID,
		"username":    u.Username,
		"message_id":  id,
	})

	// get message info
	tmp, err := h.messageGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get message info. err: %v", err)
		return nil, err
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
func (h *serviceHandler) MessageDelete(u *cscustomer.Customer, id uuid.UUID) (*mmmessage.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "MessageDelete",
		"customer_id": u.ID,
		"username":    u.Username,
		"message_id":  id,
	})

	// get message info
	_, err := h.messageGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get message info. err: %v", err)
		return nil, err
	}

	// delete message info
	tmp, err := h.reqHandler.MMV1MessageDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not get message info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
