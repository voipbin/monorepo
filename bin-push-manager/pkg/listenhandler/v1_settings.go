package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-push-manager/pkg/listenhandler/models/request"
)

// parseOwner returns the customer id and the agent id of the query of the uri. A missing or broken id is the nil
// id, which the handler rejects with a typed error.
func parseOwner(uri string) (customerID, agentID uuid.UUID, err error) {
	u, err := url.Parse(uri)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	q := u.Query()

	return uuid.FromStringOrNil(q.Get("customer_id")), uuid.FromStringOrNil(q.Get("agent_id")), nil
}

// processV1SettingsGet handles GET /v1/settings?customer_id=<id>&agent_id=<id> request
func (h *listenHandler) processV1SettingsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "processV1SettingsGet")

	customerID, agentID, err := parseOwner(m.URI)
	if err != nil {
		log.Errorf("Could not parse the uri. err: %v", err)
		return simpleResponse(400), nil
	}

	res, err := h.settingsHandler.Get(ctx, customerID, agentID)
	if err != nil {
		return nil, err
	}

	return jsonResponse(res), nil
}

// processV1SettingsPut handles PUT /v1/settings request
func (h *listenHandler) processV1SettingsPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "processV1SettingsPut")

	var req request.V1DataSettingsPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal the request. err: %v", err)
		return simpleResponse(400), nil
	}

	res, err := h.settingsHandler.Update(ctx, req.CustomerID, req.AgentID, req.Kinds)
	if err != nil {
		return nil, err
	}

	return jsonResponse(res), nil
}

// processV1MutesIDPut handles PUT /v1/mutes/<chat-id>?customer_id=<id>&agent_id=<id> request
func (h *listenHandler) processV1MutesIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "processV1MutesIDPut")

	customerID, agentID, err := parseOwner(m.URI)
	if err != nil {
		log.Errorf("Could not parse the uri. err: %v", err)
		return simpleResponse(400), nil
	}

	chatID, ok := parseChatID(m.URI)
	if !ok {
		return simpleResponse(400), nil
	}

	res, err := h.settingsHandler.MuteAdd(ctx, customerID, agentID, chatID)
	if err != nil {
		return nil, err
	}

	return jsonResponse(res), nil
}

// processV1MutesIDDelete handles DELETE /v1/mutes/<chat-id>?customer_id=<id>&agent_id=<id> request
func (h *listenHandler) processV1MutesIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "processV1MutesIDDelete")

	customerID, agentID, err := parseOwner(m.URI)
	if err != nil {
		log.Errorf("Could not parse the uri. err: %v", err)
		return simpleResponse(400), nil
	}

	chatID, ok := parseChatID(m.URI)
	if !ok {
		return simpleResponse(400), nil
	}

	res, err := h.settingsHandler.MuteRemove(ctx, customerID, agentID, chatID)
	if err != nil {
		return nil, err
	}

	return jsonResponse(res), nil
}

// parseChatID returns the chat id of the path /v1/mutes/<chat-id>. The path is already matched by the route.
func parseChatID(uri string) (uuid.UUID, bool) {
	u, err := url.Parse(uri)
	if err != nil {
		return uuid.Nil, false
	}

	// u.Path is /v1/mutes/<chat-id>
	const prefix = "/v1/mutes/"
	if len(u.Path) <= len(prefix) {
		return uuid.Nil, false
	}
	id := uuid.FromStringOrNil(u.Path[len(prefix):])

	return id, id != uuid.Nil
}
