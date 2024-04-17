package response

import (
	cacampaign "monorepo/bin-campaign-manager/models/campaign"
	cacampaigncall "monorepo/bin-campaign-manager/models/campaigncall"
)

// BodyCampaignsGET is rquest body define for
// GET /v1.0/campaigns
type BodyCampaignsGET struct {
	Result []*cacampaign.WebhookMessage `json:"result"`
	Pagination
}

// BodyCampaignsIDCampaigncallsGET is rquest body define for
// GET /v1.0/campaigns/{id}/campaigncalls
type BodyCampaignsIDCampaigncallsGET struct {
	Result []*cacampaigncall.WebhookMessage `json:"result"`
	Pagination
}
