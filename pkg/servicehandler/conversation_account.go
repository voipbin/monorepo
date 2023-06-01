package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	cvaccount "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
)

// conversationAccountGet validates the conversation account's ownership and returns the conversation account info.
func (h *serviceHandler) conversationAccountGet(ctx context.Context, u *cscustomer.Customer, conversationAccountID uuid.UUID) (*cvaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                    "conversationAccountGet",
		"customer_id":             u.ID,
		"conversation_account_id": conversationAccountID,
	},
	)

	// send request
	tmp, err := h.reqHandler.ConversationV1AccountGet(ctx, conversationAccountID)
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

// ConversationAccountGetsByCustomerID gets the list of conversation accounts of the given customer id.
// It returns list of conversation accounts if it succeed.
func (h *serviceHandler) ConversationAccountGetsByCustomerID(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*cvaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConversationAccountGetsByCustomerID",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a conversation accounts.")

	if token == "" {
		token = h.utilHandler.GetCurTime()
	}

	// get tmp
	tmp, err := h.reqHandler.ConversationV1AccountGetsByCustomerID(ctx, u.ID, token, size)
	if err != nil {
		log.Errorf("Could not get conversation account infos from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("could not find conversation accounts info. err: %v", err)
	}

	// create result
	res := []*cvaccount.WebhookMessage{}
	for _, f := range tmp {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// ConversationAccountGet gets the conversation of the given id.
// It returns conversation account if it succeed.
func (h *serviceHandler) ConversationAccountGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*cvaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                    "ConversationAccountGet",
		"customer_id":             u.ID,
		"username":                u.Username,
		"conversation_account_id": id,
	})
	log.Debug("Getting an conversation account.")

	// get campaign
	res, err := h.conversationAccountGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get conversation account info from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("could not find conversation account info. err: %v", err)
	}

	return res, nil
}

// ConversationAccountCreate creates a new conversation account
func (h *serviceHandler) ConversationAccountCreate(
	ctx context.Context,
	u *cscustomer.Customer,
	accountType cvaccount.Type,
	name string,
	detail string,
	secret string,
	token string,
) (*cvaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConversationAccountCreate",
		"customer_id": u.ID,
		"username":    u.Username,
	})
	log.Debug("Creating a new conversation account.")

	tmp, err := h.reqHandler.ConversationV1AccountCreate(ctx, u.ID, accountType, name, detail, secret, token)
	if err != nil {
		log.Errorf("Could not create a new conversation account. err: %v", err)
		return nil, errors.Wrap(err, "could not create a new conversation account")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConversationAccountUpdate updates the conversation account
func (h *serviceHandler) ConversationAccountUpdate(
	ctx context.Context,
	u *cscustomer.Customer,
	accountID uuid.UUID,
	name string,
	detail string,
	secret string,
	token string,
) (*cvaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConversationAccountUpdate",
		"customer_id": u.ID,
		"username":    u.Username,
	})
	log.Debug("Creating a new conversation account.")

	// get campaign
	_, err := h.conversationAccountGet(ctx, u, accountID)
	if err != nil {
		log.Errorf("Could not get conversation info from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("could not find conversation info. err: %v", err)
	}

	tmp, err := h.reqHandler.ConversationV1AccountUpdate(ctx, accountID, name, detail, secret, token)
	if err != nil {
		log.Errorf("Could not update the conversation account. err: %v", err)
		return nil, errors.Wrap(err, "could not update the conversation account")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConversationAccountDelete deletes the conversation account
func (h *serviceHandler) ConversationAccountDelete(ctx context.Context, u *cscustomer.Customer, accountID uuid.UUID) (*cvaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConversationAccountDelete",
		"customer_id": u.ID,
		"username":    u.Username,
	})
	log.Debug("Creating a new conversation account.")

	// get campaign
	_, err := h.conversationAccountGet(ctx, u, accountID)
	if err != nil {
		log.Errorf("Could not get conversation info from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("could not find conversation info. err: %v", err)
	}

	tmp, err := h.reqHandler.ConversationV1AccountDelete(ctx, accountID)
	if err != nil {
		log.Errorf("Could not update the conversation account. err: %v", err)
		return nil, errors.Wrap(err, "could not update the conversation account")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
