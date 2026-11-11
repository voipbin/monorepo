package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"

	pmnotification "monorepo/bin-push-manager/models/notification"
	pmsettings "monorepo/bin-push-manager/models/settings"
	pmrequest "monorepo/bin-push-manager/pkg/listenhandler/models/request"
)

// PushV1SettingsGet sends a request to push-manager to get the notification settings of the agent.
// An agent that never changed anything gets every kind on and no muted chat.
func (r *requestHandler) PushV1SettingsGet(ctx context.Context, customerID, agentID uuid.UUID) (*pmsettings.Settings, error) {
	uri := fmt.Sprintf("/v1/settings?customer_id=%s&agent_id=%s", customerID, agentID)

	tmp, err := r.sendRequestPush(ctx, uri, sock.RequestMethodGet, "push/settings", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	return parsePushSettings(tmp)
}

// PushV1SettingsUpdate sends a request to push-manager to switch the notification kinds of the agent: true is
// on, false is off. A kind that is not in the map is not changed. It returns the settings after the change.
func (r *requestHandler) PushV1SettingsUpdate(ctx context.Context, customerID, agentID uuid.UUID, kinds map[pmnotification.Kind]bool) (*pmsettings.Settings, error) {
	uri := "/v1/settings"

	m, err := json.Marshal(&pmrequest.V1DataSettingsPut{
		CustomerID: customerID,
		AgentID:    agentID,
		Kinds:      kinds,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestPush(ctx, uri, sock.RequestMethodPut, "push/settings", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	return parsePushSettings(tmp)
}

// PushV1MuteAdd sends a request to push-manager to mute the chat for the agent. Muting a muted chat succeeds.
// It returns the settings after the change.
func (r *requestHandler) PushV1MuteAdd(ctx context.Context, customerID, agentID, chatID uuid.UUID) (*pmsettings.Settings, error) {
	uri := fmt.Sprintf("/v1/mutes/%s?customer_id=%s&agent_id=%s", chatID, customerID, agentID)

	tmp, err := r.sendRequestPush(ctx, uri, sock.RequestMethodPut, "push/mutes/<chat-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	return parsePushSettings(tmp)
}

// PushV1MuteRemove sends a request to push-manager to unmute the chat for the agent. Unmuting a chat that is not
// muted succeeds. It returns the settings after the change.
func (r *requestHandler) PushV1MuteRemove(ctx context.Context, customerID, agentID, chatID uuid.UUID) (*pmsettings.Settings, error) {
	uri := fmt.Sprintf("/v1/mutes/%s?customer_id=%s&agent_id=%s", chatID, customerID, agentID)

	tmp, err := r.sendRequestPush(ctx, uri, sock.RequestMethodDelete, "push/mutes/<chat-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	return parsePushSettings(tmp)
}

// parsePushSettings parses the settings of a response.
func parsePushSettings(tmp *sock.Response) (*pmsettings.Settings, error) {
	var res pmsettings.Settings
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
