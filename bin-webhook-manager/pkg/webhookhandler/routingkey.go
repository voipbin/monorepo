package webhookhandler

import (
	"context"
	"encoding/json"
	"fmt"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// webhookOwnerData mirrors bin-api-manager's commonWebhookData struct (ported per VOIP-1258 §6
// harder part 1). Used to extract customer_id/agent_id (owner) from the event data payload
// BEFORE publish, so the routing key can be computed at publish time instead of at consumption
// time.
type webhookOwnerData struct {
	commonidentity.Identity
	commonidentity.Owner
	AIcallID uuid.UUID `json:"aicall_id"`
	ChatID   uuid.UUID `json:"chat_id"`
}

// parseWebhookOwnerData unmarshals the event data payload to extract customer_id/owner_id/
// aicall_id/chat_id. Best-effort: returns zero-value fields (not an error) if optional fields
// are absent, matching createTopics()'s existing tolerance for partial data.
func parseWebhookOwnerData(data json.RawMessage) (*webhookOwnerData, error) {
	d := &webhookOwnerData{}
	if err := json.Unmarshal(data, d); err != nil {
		return nil, err
	}
	return d, nil
}

// createRoutingKeys generates AMQP routing keys for the given event, scope-first
// (VOIP-1258 §5): "<scope>.<scope_id>.<resource>.<message_type>.<resource_id>".
// The agent_id wire prefix is unchanged (an agent_id -> owner_id rename was considered and
// reverted, see design doc §5).
func createRoutingKeys(d *webhookOwnerData, resource string, messageType string) []string {
	res := []string{}

	if d.CustomerID != uuid.Nil {
		res = append(res, fmt.Sprintf("customer_id.%s.%s.%s.%s", d.CustomerID, resource, messageType, d.ID))
	}
	if d.OwnerID != uuid.Nil {
		res = append(res, fmt.Sprintf("agent_id.%s.%s.%s.%s", d.OwnerID, resource, messageType, d.ID))
	}

	return res
}

// createRoutingKeysForChat generates routing keys for chat/chatmessage/chatparticipant events,
// including one agent_id key per chat participant (fan-out). This RPC call was moved here from
// bin-api-manager's createTopics() per VOIP-1258 §6 harder part 2 -- it now runs ONCE at publish
// time regardless of subscriber count, instead of once per pod after the fact.
func (h *webhookHandler) createRoutingKeysForChat(ctx context.Context, d *webhookOwnerData, messageType string) []string {
	log := logrus.WithFields(logrus.Fields{"func": "createRoutingKeysForChat"})

	res := []string{}

	chatID := d.ChatID
	if chatID == uuid.Nil {
		chatID = d.ID
	}
	if chatID == uuid.Nil {
		return res
	}

	if d.CustomerID != uuid.Nil {
		res = append(res, fmt.Sprintf("customer_id.%s.talk.%s.%s", d.CustomerID, messageType, chatID))
	}
	if d.OwnerID != uuid.Nil {
		res = append(res, fmt.Sprintf("agent_id.%s.talk.%s.%s", d.OwnerID, messageType, chatID))
	}

	participants, err := h.reqHandler.TalkV1ParticipantList(ctx, chatID)
	if err != nil {
		log.Errorf("Could not get chat participants, publishing customer/agent-scoped keys only. err: %v", err)
		return res
	}

	// NOTE: participants is []*tkparticipant.Participant (pointer slice) -- verified against
	// bin-common-handler/pkg/requesthandler/main.go:1394. p is a pointer here, not a value.
	for _, p := range participants {
		if p.OwnerID == d.OwnerID {
			continue // already added above
		}
		res = append(res, fmt.Sprintf("agent_id.%s.talk.%s.%s", p.OwnerID, messageType, chatID))
	}

	return res
}
