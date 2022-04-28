package response

import cacampaign "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"

// BodyCampaignsGET is rquest body define for GET /campaigns
type BodyCampaignsGET struct {
	Result []*cacampaign.WebhookMessage `json:"result"`
	Pagination
}
