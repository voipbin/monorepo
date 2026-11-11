package settingshandler

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/gofrs/uuid"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-push-manager/models/notification"
	"monorepo/bin-push-manager/models/settings"
)

// checkOwner returns an error if the owner of the settings is not complete.
func checkOwner(customerID, agentID uuid.UUID) error {
	if customerID == uuid.Nil || agentID == uuid.Nil {
		return cerrors.InvalidArgument(commonoutline.ServiceNamePushManager, "INVALID_OWNER", "The customer id and the agent id are required.")
	}

	return nil
}

// checkChat returns an error if the chat id is empty.
func checkChat(chatID uuid.UUID) error {
	if chatID == uuid.Nil {
		return cerrors.InvalidArgument(commonoutline.ServiceNamePushManager, "INVALID_CHAT", "The chat id is required.")
	}

	return nil
}

// Get returns the notification settings of the agent.
func (h *settingsHandler) Get(ctx context.Context, customerID, agentID uuid.UUID) (*settings.Settings, error) {
	if err := checkOwner(customerID, agentID); err != nil {
		return nil, err
	}

	disabled, err := h.db.KindPrefListDisabledByAgent(ctx, customerID, agentID)
	if err != nil {
		return nil, fmt.Errorf("could not get the disabled kinds. err: %w", err)
	}

	muted, err := h.db.MuteListByAgent(ctx, customerID, agentID)
	if err != nil {
		return nil, fmt.Errorf("could not get the muted chats. err: %w", err)
	}

	return settings.New(disabled, muted), nil
}

// Update switches the kinds and returns the settings after the change.
func (h *settingsHandler) Update(ctx context.Context, customerID, agentID uuid.UUID, kinds map[notification.Kind]bool) (*settings.Settings, error) {
	if err := checkOwner(customerID, agentID); err != nil {
		return nil, err
	}

	// check everything first, so a bad request writes nothing
	list := make([]notification.Kind, 0, len(kinds))
	for k := range kinds {
		if !notification.IsSettingsKind(k) {
			return nil, cerrors.InvalidArgument(commonoutline.ServiceNamePushManager, "INVALID_KIND", "The notification kind can not be changed.")
		}
		list = append(list, k)
	}
	sort.Slice(list, func(i, j int) bool { return list[i] < list[j] })

	for _, k := range list {
		var err error
		if kinds[k] {
			err = h.db.KindPrefEnable(ctx, customerID, agentID, k)
		} else {
			err = h.db.KindPrefDisable(ctx, customerID, agentID, k)
		}
		if err != nil {
			return nil, fmt.Errorf("could not switch the kind. kind: %s, err: %w", k, err)
		}
	}

	return h.Get(ctx, customerID, agentID)
}

// MuteAdd mutes the chat for the agent.
func (h *settingsHandler) MuteAdd(ctx context.Context, customerID, agentID, chatID uuid.UUID) (*settings.Settings, error) {
	if err := checkOwner(customerID, agentID); err != nil {
		return nil, err
	}
	if err := checkChat(chatID); err != nil {
		return nil, err
	}

	if err := h.db.MuteCreate(ctx, customerID, agentID, chatID); err != nil {
		return nil, fmt.Errorf("could not mute the chat. err: %w", err)
	}

	return h.Get(ctx, customerID, agentID)
}

// MuteRemove unmutes the chat for the agent.
func (h *settingsHandler) MuteRemove(ctx context.Context, customerID, agentID, chatID uuid.UUID) (*settings.Settings, error) {
	if err := checkOwner(customerID, agentID); err != nil {
		return nil, err
	}
	if err := checkChat(chatID); err != nil {
		return nil, err
	}

	if err := h.db.MuteDelete(ctx, customerID, agentID, chatID); err != nil {
		return nil, fmt.Errorf("could not unmute the chat. err: %w", err)
	}

	return h.Get(ctx, customerID, agentID)
}

// FilterRecipients returns the agents that want the kind.
func (h *settingsHandler) FilterRecipients(ctx context.Context, customerID uuid.UUID, agentIDs []uuid.UUID, kind notification.Kind, chatID uuid.UUID) ([]uuid.UUID, error) {
	if len(agentIDs) == 0 {
		return []uuid.UUID{}, nil
	}

	off := map[uuid.UUID]bool{}

	disabled, err := h.db.KindPrefListDisabledByAgents(ctx, customerID, agentIDs, kind)
	if err != nil {
		return nil, fmt.Errorf("could not get the agents that switched the kind off. err: %w", err)
	}
	for _, id := range disabled {
		off[id] = true
	}

	if kind == notification.KindChatMessage {
		muted, err := h.db.MuteListMutedByAgents(ctx, customerID, agentIDs, chatID)
		if err != nil {
			return nil, fmt.Errorf("could not get the agents that muted the chat. err: %w", err)
		}
		for _, id := range muted {
			off[id] = true
		}
	}

	res := make([]uuid.UUID, 0, len(agentIDs))
	for _, id := range agentIDs {
		if !off[id] {
			res = append(res, id)
		}
	}

	return res, nil
}

// EventCustomerDeleted handles the customer-manager's customer_deleted event: the settings of the customer are
// deleted. Both tables are tried, and the errors are returned together.
func (h *settingsHandler) EventCustomerDeleted(ctx context.Context, c *cmcustomer.Customer) error {
	errKind := h.db.KindPrefDeleteByCustomerID(ctx, c.ID)
	errMute := h.db.MuteDeleteByCustomerID(ctx, c.ID)

	if err := errors.Join(errKind, errMute); err != nil {
		return fmt.Errorf("could not delete the settings of the customer. err: %w", err)
	}

	return nil
}

// EventAgentDeleted handles the agent-manager's agent_deleted event: the settings of the agent are deleted. Both
// tables are tried, and the errors are returned together.
func (h *settingsHandler) EventAgentDeleted(ctx context.Context, a *amagent.Agent) error {
	errKind := h.db.KindPrefDeleteByAgentID(ctx, a.ID)
	errMute := h.db.MuteDeleteByAgentID(ctx, a.ID)

	if err := errors.Join(errKind, errMute); err != nil {
		return fmt.Errorf("could not delete the settings of the agent. err: %w", err)
	}

	return nil
}
