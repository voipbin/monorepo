package webhookhandler

import (
	"encoding/json"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
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
