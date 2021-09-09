package request

import (
	"encoding/json"

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
	Method     webhook.MethodType `json:"method"`      // webhook method
	WebhookURI string             `json:"webhook_uri"` // webhook destination uri
	DataType   webhook.DataType   `json:"data_type"`
	Data       WebhookData        `json:"data"`
}
