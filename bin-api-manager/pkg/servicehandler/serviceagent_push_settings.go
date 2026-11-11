package servicehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/models/auth"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	pmnotification "monorepo/bin-push-manager/models/notification"
	pmsettings "monorepo/bin-push-manager/models/settings"
)

// ServiceAgentPushSettingsGet returns the push notification settings of the calling agent.
//
// The owner always comes from the authenticated identity. The settings hold ids and booleans only, so they are
// returned as they are.
func (h *serviceHandler) ServiceAgentPushSettingsGet(ctx context.Context, a *auth.AuthIdentity) (*pmsettings.Settings, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentPushSettingsGet",
		"customer_id": a.CustomerID,
	})

	agentID, err := h.serviceAgentStarCheck(ctx, a)
	if err != nil {
		log.Infof("The agent is not allowed. err: %v", err)
		return nil, err
	}

	res, err := h.reqHandler.PushV1SettingsGet(ctx, a.CustomerID, agentID)
	if err != nil {
		log.Errorf("Could not get the push settings. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceAgentPushSettingsUpdate switches push notification kinds of the calling agent.
//
// The kinds are checked by push-manager before anything is written. Its typed 400 (a kind that can not be
// switched) is returned as it is, so the caller gets a 400 with the reason.
func (h *serviceHandler) ServiceAgentPushSettingsUpdate(ctx context.Context, a *auth.AuthIdentity, kinds map[pmnotification.Kind]bool) (*pmsettings.Settings, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentPushSettingsUpdate",
		"customer_id": a.CustomerID,
	})

	agentID, err := h.serviceAgentStarCheck(ctx, a)
	if err != nil {
		log.Infof("The agent is not allowed. err: %v", err)
		return nil, err
	}

	res, err := h.reqHandler.PushV1SettingsUpdate(ctx, a.CustomerID, agentID, kinds)
	if err != nil {
		log.Errorf("Could not update the push settings. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceAgentPushMuteAdd mutes the chat for the calling agent.
//
// The chat must belong to the customer of the caller; the caller does not have to take part in it, because a
// public chat is readable by every agent of the customer. A chat that does not exist, that belongs to another
// customer and that can not be read are all reported as the same not found: the lookup does not tell them apart.
func (h *serviceHandler) ServiceAgentPushMuteAdd(ctx context.Context, a *auth.AuthIdentity, chatID uuid.UUID) (*pmsettings.Settings, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentPushMuteAdd",
		"customer_id": a.CustomerID,
		"chat_id":     chatID,
	})

	agentID, err := h.serviceAgentStarCheck(ctx, a)
	if err != nil {
		log.Infof("The agent is not allowed. err: %v", err)
		return nil, err
	}

	chat, err := h.talkGet(ctx, chatID)
	if err != nil {
		log.Infof("Could not get the chat. err: %v", err)
		return nil, newChatNotFound()
	}
	if chat == nil || chat.CustomerID != a.CustomerID {
		return nil, newChatNotFound()
	}

	res, err := h.reqHandler.PushV1MuteAdd(ctx, a.CustomerID, agentID, chatID)
	if err != nil {
		log.Errorf("Could not mute the chat. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceAgentPushMuteRemove unmutes the chat for the calling agent. It does not look at the chat: unmuting a
// chat that is not muted, or that is gone, succeeds.
func (h *serviceHandler) ServiceAgentPushMuteRemove(ctx context.Context, a *auth.AuthIdentity, chatID uuid.UUID) (*pmsettings.Settings, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentPushMuteRemove",
		"customer_id": a.CustomerID,
		"chat_id":     chatID,
	})

	agentID, err := h.serviceAgentStarCheck(ctx, a)
	if err != nil {
		log.Infof("The agent is not allowed. err: %v", err)
		return nil, err
	}

	res, err := h.reqHandler.PushV1MuteRemove(ctx, a.CustomerID, agentID, chatID)
	if err != nil {
		log.Errorf("Could not unmute the chat. err: %v", err)
		return nil, err
	}

	return res, nil
}

// newChatNotFound returns the canonical "chat not found" error of the mute call.
func newChatNotFound() error {
	return cerrors.NotFound(commonoutline.ServiceNameTalkManager, "CHAT_NOT_FOUND", "The chat was not found.")
}
