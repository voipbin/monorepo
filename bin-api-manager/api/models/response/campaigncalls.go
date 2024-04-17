package response

import (
	cmcampaigncall "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
)

// BodyCampaigncallsGET is response body define for
// GET /v1.0/campaigncalls
type BodyCampaigncallsGET struct {
	Result []*cmcampaigncall.WebhookMessage `json:"result"`
	Pagination
}
