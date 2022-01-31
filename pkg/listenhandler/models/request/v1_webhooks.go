package request

import (
	"encoding/json"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
)

// WebhookData defines
type WebhookData struct {
	Type string          `json:"type"` // message type
	Data json.RawMessage `json:"data"` // data
}

// V1DataWebhooksPost is
// v1 data type request struct for
// /v1/webhooks POST
type V1DataWebhooksPost struct {
	CustomerID uuid.UUID `json:"customer_id"` // customer's id
	Method     webhook.MethodType
	WebhookURI string
	DataType   webhook.DataType `json:"data_type"`
	Data       WebhookData      `json:"data"`
}
