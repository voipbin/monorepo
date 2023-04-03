package response

import (
	cacampaign "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	cacampaigncall "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
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
