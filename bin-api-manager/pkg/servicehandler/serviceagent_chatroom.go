package servicehandler

import (
	"context"
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	chatchatroom "monorepo/bin-chat-manager/models/chatroom"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// ServiceAgentChatroomGets sends a request to chat-manager
// to getting the given agent's list of chatrooms.
// it returns list of chatrooms if it succeed.
func (h *serviceHandler) ServiceAgentChatroomGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*chatchatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentChatroomGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// filters
	filters := map[string]string{
		"owner_id":    a.ID.String(),
		"customer_id": a.CustomerID.String(),
		"deleted":     "false", // we don't need deleted items
	}

	res, err := h.chatroomGetsByFilters(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not chatrooms info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceAgentChatroomGets sends a request to chat-manager
// to getting the given agent's list of chatrooms.
// it returns list of chatrooms if it succeed.
func (h *serviceHandler) ServiceAgentChatroomGet(ctx context.Context, a *amagent.Agent, chatroomID uuid.UUID) (*chatchatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentChatroomGets",
		"agent":       a,
		"chatroom_id": chatroomID,
	})

	tmp, err := h.chatroomGet(ctx, chatroomID)
	if err != nil {
		log.Errorf("Could not get chatroom info. err: %v", err)
		return nil, err
	}

	if a.ID != tmp.OwnerID {
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentChatroomDelete sends a request to chat-manager
// to getting the given agent's list of chatrooms.
// it returns list of chatrooms if it succeed.
func (h *serviceHandler) ServiceAgentChatroomDelete(ctx context.Context, a *amagent.Agent, chatroomID uuid.UUID) (*chatchatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentChatroomDelete",
		"agent":       a,
		"chatroom_id": chatroomID,
	})

	tmp, err := h.chatroomGet(ctx, chatroomID)
	if err != nil {
		log.Errorf("Could not get chatroom info. err: %v", err)
		return nil, err
	}

	if a.ID != tmp.OwnerID {
		return nil, fmt.Errorf("user has no permission")
	}

	tmpRes, err := h.chatroomDelete(ctx, chatroomID)
	if err != nil {
		log.Errorf("Could not delete chatroom info. err: %v", err)
		return nil, err
	}

	res := tmpRes.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentChatroomCreate creates the chatroom message of the given chatroom id.
// It returns created chatroommessages if it succeed.
func (h *serviceHandler) ServiceAgentChatroomCreate(ctx context.Context, a *amagent.Agent, participantIDs []uuid.UUID, name string, detail string) (*chatchatroom.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ServiceAgentChatroomCreate",
		"agent":           a,
		"participant_ids": participantIDs,
		"name":            name,
		"detail":          detail,
	})

	res, err := h.ChatroomCreate(ctx, a, participantIDs, name, detail)
	if err != nil {
		log.Errorf("Could not create chatroom info. err: %v", err)
		return nil, err
	}

	return res, nil
}
