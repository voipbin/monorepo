package webhookhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-webhook-manager/models/webhook"

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

// publishRoutingKeyedEvent computes routing keys for the given event data and publishes to the
// new topic exchange with each key. Best-effort: logs and returns on parse/RPC failure without
// blocking the primary (fanout) delivery path above it.
//
// CRITICAL (envelope unwrapping): the `data` param here is the SAME json.RawMessage that
// SendWebhookToCustomer/SendWebhookToURI receive, which is ALWAYS a nested wire envelope --
// `{"type":"<resource>_<verb>","data":{...actual resource fields incl. customer_id/owner_id...}}`
// -- because every caller of those two methods builds it via `json.Marshal(webhook.Data{Type,
// Data})` (see bin-webhook-manager/models/webhook/webhook.go and
// bin-webhook-manager/pkg/listenhandler/v1_webhooks.go). It is NEVER a flat resource object.
// This was the actual production bug found during VOIP-1258 post-deploy verification
// (2026-07-14): an earlier version of this function unmarshaled `data` directly into
// webhookOwnerData, so customer_id/owner_id were always absent at the top level and always
// parsed as uuid.Nil -- createRoutingKeys then returned an empty slice and NOTHING was ever
// published to the topic exchange, even though the fanout (old-path) delivery succeeded and
// looked completely fine. The pre-VOIP-1258 consumer-side code (bin-api-manager's
// processEventWebhookManagerWebhookPublished/createTopics, now deleted) did this unwrap
// correctly: unmarshal into webhook.Data{Type, Data}, use .Type as the message type, and parse
// owner fields from .Data (not from the outer envelope). This function now does the same.
func (h *webhookHandler) publishRoutingKeyedEvent(ctx context.Context, _ string, data json.RawMessage) {
	log := logrus.WithFields(logrus.Fields{"func": "publishRoutingKeyedEvent"})

	envelope := &webhook.Data{}
	if err := json.Unmarshal(data, envelope); err != nil {
		log.Errorf("Could not unmarshal the webhook envelope for routing key computation. err: %v", err)
		return
	}
	eventType := envelope.Type
	log = log.WithField("event_type", eventType)

	d, err := parseWebhookOwnerData(envelope.Data)
	if err != nil {
		log.Errorf("Could not parse owner data for routing key computation. err: %v", err)
		return
	}

	// messageType/resource parsing: eventType is the wire event type e.g. "call_updated".
	// resource = first underscore-delimited segment, matching createTopics()'s existing
	// convention (webhookmanager.go:111-120 in the pre-migration bin-api-manager code).
	tmps := strings.SplitN(eventType, "_", 2)
	if len(tmps) < 2 {
		log.Errorf("Wrong event type format for routing key. event_type: %s", eventType)
		return
	}
	resource := tmps[0]

	var keys []string
	switch resource {
	case "chat", "chatmessage", "chatparticipant":
		keys = h.createRoutingKeysForChat(ctx, d, eventType)
	default:
		keys = createRoutingKeys(d, resource, eventType)
	}

	for _, key := range keys {
		h.topicNotifyHandler.PublishEventWithRoutingKey(ctx, eventType, key, envelope.Data)
	}
}
