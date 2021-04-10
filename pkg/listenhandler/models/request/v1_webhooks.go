package request

import (
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
)

// V1DataWebhooksPost is
// v1 data type request struct for
// /v1/webhooks POST
type V1DataWebhooksPost struct {
	webhook.Webhook
}
