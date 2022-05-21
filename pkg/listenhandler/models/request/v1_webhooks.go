package request

import (
	"encoding/json"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
)

// V1DataWebhooksPost is
// v1 data type request struct for
// /v1/webhooks POST
type V1DataWebhooksPost struct {
	CustomerID uuid.UUID        `json:"customer_id"` // customer's id
	DataType   webhook.DataType `json:"data_type"`   // application/json
	Data       webhook.Data     `json:"data"`
}

// V1DataWebhookDestinationsPost is
// /v1/webhook_destinations POST
type V1DataWebhookDestinationsPost struct {
	CustomerID uuid.UUID          `json:"customer_id"` // customer's id
	URI        string             `json:"uri"`         // send uri
	Method     webhook.MethodType `json:"method"`      // send method
	DataType   webhook.DataType   `json:"data_type"`   // application/json
	Data       json.RawMessage    `json:"data"`
}
