package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	chatchat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
)

// chatGet validates the chat's ownership and returns the chat info.
func (h *serviceHandler) chatGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatchat.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "chatGet",
			"customer_id": u.ID,
			"chat_id":     id,
		},
	)

	// send request
	tmp, err := h.reqHandler.ChatV1ChatGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the chat info. err: %v", err)
		return nil, err
	}
	log.WithField("chat", tmp).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmp.CustomerID {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatCreate is a service handler for chat creation.
func (h *serviceHandler) ChatCreate(
	ctx context.Context,
	u *cscustomer.Customer,
	chatType chatchat.Type,
	ownerID uuid.UUID,
	participantIDs []uuid.UUID,
	name string,
	detail string,
) (*chatchat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatCreate",
		"customer_id": u.ID,
		"name":        name,
	})

	log.Debug("Creating a new chat.")
	tmp, err := h.reqHandler.ChatV1ChatCreate(
		ctx,
		u.ID,
		chatType,
		ownerID,
		participantIDs,
		name,
		detail,
	)
	if err != nil {
		log.Errorf("Could not create a new chat. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatGetsByCustomerID gets the list of chats of the given customer id.
// It returns list of chats if it succeed.
func (h *serviceHandler) ChatGetsByCustomerID(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*chatchat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatGetsByCustomerID",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a chats.")

	if token == "" {
		token = getCurTime()
	}

	// get chats
	tmps, err := h.reqHandler.ChatV1ChatGetsByCustomerID(ctx, u.ID, token, size)
	if err != nil {
		log.Errorf("Could not get chats info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chats info. err: %v", err)
	}

	// create result
	res := []*chatchat.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// ChatGet gets the chat of the given id.
// It returns chat if it succeed.
func (h *serviceHandler) ChatGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatchat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatGet",
		"customer_id": u.ID,
		"username":    u.Username,
		"chat_id":     id,
	})
	log.Debug("Getting a chat.")

	// get chat
	res, err := h.chatGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chat info. err: %v", err)
	}

	return res, nil
}

// ChatDelete deletes the chat.
func (h *serviceHandler) ChatDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatchat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatDelete",
		"customer_id": u.ID,
		"username":    u.Username,
		"chat_id":     id,
	})
	log.Debug("Deleting a chat.")

	// get chat
	_, err := h.chatGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chat info. err: %v", err)
	}

	tmp, err := h.reqHandler.ChatV1ChatDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the chat. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatUpdateBasicInfo updates the chat's basic info.
// It returns updated chat if it succeed.
func (h *serviceHandler) ChatUpdateBasicInfo(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, name, detail string) (*chatchat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatUpdateBasicInfo",
		"customer_id": u.ID,
		"username":    u.Username,
		"chat_id":     id,
	})
	log.Debug("Updating a chat.")

	// get chat
	_, err := h.chatGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chat info. err: %v", err)
	}

	tmp, err := h.reqHandler.ChatV1ChatUpdateBasicInfo(ctx, id, name, detail)
	if err != nil {
		logrus.Errorf("Could not update the chat. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatUpdateOwnerID updates the chat's status.
// It returns updated chat if it succeed.
func (h *serviceHandler) ChatUpdateOwnerID(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, ownerID uuid.UUID) (*chatchat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatUpdateOwnerID",
		"customer_id": u.ID,
		"username":    u.Username,
		"chat_id":     id,
	})
	log.Debug("Updating an chat.")

	// get chat
	_, err := h.chatGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chat info. err: %v", err)
	}

	tmp, err := h.reqHandler.ChatV1ChatUpdateOwnerID(ctx, id, ownerID)
	if err != nil {
		logrus.Errorf("Could not update the chat. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatAddParticipantID add the given participant id to the chat.
// It returns updated chat if it succeed.
func (h *serviceHandler) ChatAddParticipantID(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, participantID uuid.UUID) (*chatchat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatAddParticipantID",
		"customer_id": u.ID,
		"username":    u.Username,
		"chat_id":     id,
	})
	log.Debug("Adding the participant id to the chat.")

	// get chat
	_, err := h.chatGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chat info. err: %v", err)
	}

	tmp, err := h.reqHandler.ChatV1ChatAddParticipantID(ctx, id, participantID)
	if err != nil {
		logrus.Errorf("Could not add the participant id to the chat. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatRemoveParticipantID removes the given participant id from the chat.
// It returns updated chat if it succeed.
func (h *serviceHandler) ChatRemoveParticipantID(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, participantID uuid.UUID) (*chatchat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatRemoveParticipantID",
		"customer_id": u.ID,
		"username":    u.Username,
		"chat_id":     id,
	})
	log.Debug("Removing the participant id from the chat.")

	// get chat
	_, err := h.chatGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chat info. err: %v", err)
	}

	tmp, err := h.reqHandler.ChatV1ChatRemoveParticipantID(ctx, id, participantID)
	if err != nil {
		logrus.Errorf("Could not remove the participant id from the chat. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
