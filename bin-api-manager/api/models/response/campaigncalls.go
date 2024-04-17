package response

import (
	cmcampaigncall "monorepo/bin-campaign-manager/models/campaigncall"
)

// BodyCampaigncallsGET is response body define for
// GET /v1.0/campaigncalls
type BodyCampaigncallsGET struct {
	Result []*cmcampaigncall.WebhookMessage `json:"result"`
	Pagination
}
