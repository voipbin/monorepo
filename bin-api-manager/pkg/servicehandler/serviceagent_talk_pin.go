package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	tkpin "monorepo/bin-talk-manager/models/pin"
)

// pinOwnerTypeAgent is the owner type of the pins of an agent.
const pinOwnerTypeAgent = "agent"

// serviceAgentTalkPinCheck is the check every pin endpoint starts with. It returns the agent id.
func (h *serviceHandler) serviceAgentTalkPinCheck(ctx context.Context, a *auth.AuthIdentity) (uuid.UUID, error) {
	if !a.IsAgent() {
		return uuid.Nil, serviceerrors.ErrAuthenticationRequired
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		return uuid.Nil, serviceerrors.ErrPermissionDenied
	}

	// a pin without an owner would belong to nobody
	agentID := a.AgentID()
	if agentID == uuid.Nil {
		return uuid.Nil, serviceerrors.ErrPermissionDenied
	}

	return agentID, nil
}

// ServiceAgentTalkPinCreate pins a chat or a message for the calling agent. The call is idempotent.
// The agent needs access to the chat of the target; a deleted message is reported as not found.
func (h *serviceHandler) ServiceAgentTalkPinCreate(ctx context.Context, a *auth.AuthIdentity, targetType tkpin.TargetType, targetID uuid.UUID) (*tkpin.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentTalkPinCreate",
		"target_type": targetType,
		"target_id":   targetID,
	})

	agentID, err := h.serviceAgentTalkPinCheck(ctx, a)
	if err != nil {
		return nil, err
	}

	if !targetType.IsValid() {
		return nil, fmt.Errorf("%w: target_type must be chat or message", serviceerrors.ErrInvalidArgument)
	}
	if targetID == uuid.Nil {
		return nil, fmt.Errorf("%w: target_id is required", serviceerrors.ErrInvalidArgument)
	}

	chatID := targetID
	if targetType == tkpin.TargetTypeMessage {
		m, errGet := h.talkMessageGet(ctx, targetID)
		if errGet != nil {
			return nil, errors.Wrapf(errGet, "Could not get the message.")
		}
		if m.TMDelete != nil {
			return nil, serviceerrors.ErrNotFound
		}
		chatID = m.ChatID
	}

	if !h.canAccessChat(ctx, agentID, a.CustomerID, chatID) {
		log.Info("The agent has no access to the chat.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.TalkV1PinCreate(ctx, a.CustomerID, pinOwnerTypeAgent, agentID, targetType, targetID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create the pin.")
	}

	res, err := tmp.ConvertWebhookMessage()
	if err != nil {
		return nil, errors.Wrapf(err, "Could not convert the pin.")
	}

	return res, nil
}

// ServiceAgentTalkPinList returns the pins of the calling agent, newest first. targetType and
// chatID filter the result when they are set. The talk-manager hides the pins whose target the
// agent can no longer see.
func (h *serviceHandler) ServiceAgentTalkPinList(ctx context.Context, a *auth.AuthIdentity, targetType tkpin.TargetType, chatID uuid.UUID, size uint64, token string) ([]*tkpin.WebhookMessage, error) {
	agentID, err := h.serviceAgentTalkPinCheck(ctx, a)
	if err != nil {
		return nil, err
	}

	if targetType != "" && !targetType.IsValid() {
		return nil, fmt.Errorf("%w: target_type must be chat or message", serviceerrors.ErrInvalidArgument)
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	tmps, err := h.reqHandler.TalkV1PinList(ctx, a.CustomerID, pinOwnerTypeAgent, agentID, targetType, chatID, token, size)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get the pins.")
	}

	res := []*tkpin.WebhookMessage{}
	for _, p := range tmps {
		wm, errConv := p.ConvertWebhookMessage()
		if errConv != nil {
			return nil, errors.Wrapf(errConv, "Could not convert the pin.")
		}
		res = append(res, wm)
	}

	return res, nil
}

// ServiceAgentTalkPinDelete removes a pin of the calling agent. The pin of another agent is
// reported as not found by the talk-manager.
func (h *serviceHandler) ServiceAgentTalkPinDelete(ctx context.Context, a *auth.AuthIdentity, pinID uuid.UUID) (*tkpin.WebhookMessage, error) {
	agentID, err := h.serviceAgentTalkPinCheck(ctx, a)
	if err != nil {
		return nil, err
	}

	if pinID == uuid.Nil {
		return nil, fmt.Errorf("%w: pin id is required", serviceerrors.ErrInvalidArgument)
	}

	tmp, err := h.reqHandler.TalkV1PinDelete(ctx, pinID, a.CustomerID, pinOwnerTypeAgent, agentID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not delete the pin.")
	}

	res, err := tmp.ConvertWebhookMessage()
	if err != nil {
		return nil, errors.Wrapf(err, "Could not convert the pin.")
	}

	return res, nil
}
