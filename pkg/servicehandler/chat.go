package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	chatchat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
)

// chatGet validates the chat's ownership and returns the chat info.
func (h *serviceHandler) chatGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatchat.Chat, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "chatGet",
		"customer_id": a.CustomerID,
		"chat_id":     id,
	})

	// send request
	res, err := h.reqHandler.ChatV1ChatGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the chat info. err: %v", err)
		return nil, err
	}
	log.WithField("chat", res).Debug("Received result.")

	return res, nil
}

// ChatCreate is a service handler for chat creation.
func (h *serviceHandler) ChatCreate(
	ctx context.Context,
	a *amagent.Agent,
	chatType chatchat.Type,
	ownerID uuid.UUID,
	participantIDs []uuid.UUID,
	name string,
	detail string,
) (*chatchat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatCreate",
		"customer_id": a.CustomerID,
		"name":        name,
	})
	log.Debug("Creating a new chat.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ChatV1ChatCreate(
		ctx,
		a.CustomerID,
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
func (h *serviceHandler) ChatGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*chatchat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatGetsByCustomerID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a chats.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		return nil, fmt.Errorf("agent has no permission")
	}

	// get chats
	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false",
	}
	tmps, err := h.reqHandler.ChatV1ChatGets(ctx, token, size, filters)
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
func (h *serviceHandler) ChatGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatchat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"chat_id":     id,
	})
	log.Debug("Getting a chat.")

	// get chat
	tmp, err := h.chatGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chat info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionAll) {
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ChatDelete deletes the chat.
func (h *serviceHandler) ChatDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatchat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"chat_id":     id,
	})
	log.Debug("Deleting a chat.")

	// get chat
	c, err := h.chatGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chat info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("agent has no permission")
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
func (h *serviceHandler) ChatUpdateBasicInfo(ctx context.Context, a *amagent.Agent, id uuid.UUID, name, detail string) (*chatchat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatUpdateBasicInfo",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"chat_id":     id,
	})
	log.Debug("Updating a chat.")

	// get chat
	c, err := h.chatGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chat info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionAll) {
		return nil, fmt.Errorf("agent has no permission")
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
func (h *serviceHandler) ChatUpdateOwnerID(ctx context.Context, a *amagent.Agent, id uuid.UUID, ownerID uuid.UUID) (*chatchat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatUpdateOwnerID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"chat_id":     id,
	})
	log.Debug("Updating an chat.")

	// get chat
	c, err := h.chatGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chat info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("agent has no permission")
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
func (h *serviceHandler) ChatAddParticipantID(ctx context.Context, a *amagent.Agent, id uuid.UUID, participantID uuid.UUID) (*chatchat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatAddParticipantID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"chat_id":     id,
	})
	log.Debug("Adding the participant id to the chat.")

	// get chat
	c, err := h.chatGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chat info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionAll) {
		return nil, fmt.Errorf("agent has no permission")
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
func (h *serviceHandler) ChatRemoveParticipantID(ctx context.Context, a *amagent.Agent, id uuid.UUID, participantID uuid.UUID) (*chatchat.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ChatRemoveParticipantID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"chat_id":     id,
	})
	log.Debug("Removing the participant id from the chat.")

	// get chat
	c, err := h.chatGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get chat info from the chat-manager. err: %v", err)
		return nil, fmt.Errorf("could not find chat info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionAll) {
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ChatV1ChatRemoveParticipantID(ctx, id, participantID)
	if err != nil {
		logrus.Errorf("Could not remove the participant id from the chat. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
