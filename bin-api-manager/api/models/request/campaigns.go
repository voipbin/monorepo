package request

import (
	"github.com/gofrs/uuid"
	cacampaign "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// BodyCampaignsPOST is rquest body define for
// POST /v1.0/campaigns
type BodyCampaignsPOST struct {
	Name         string               `json:"name"`
	Detail       string               `json:"detail"`
	Type         cacampaign.Type      `json:"type"`
	ServiceLevel int                  `json:"service_level"`
	EndHandle    cacampaign.EndHandle `json:"end_handle"`

	Actions []fmaction.Action `json:"actions"` // this actions will be stored to the flow

	// resource info
	OutplanID      uuid.UUID `json:"outplan_id"`
	OutdialID      uuid.UUID `json:"outdial_id"`
	QueueID        uuid.UUID `json:"queue_id"`
	NextCampaignID uuid.UUID `json:"next_campaign_id"`
}

// ParamCampaignsGET is rquest param define for
// GET /v1.0/campaigns
type ParamCampaignsGET struct {
	Pagination
}

// BodyCampaignsIDPUT is rquest body define for
// PUT /v1.0/campaigns/<campaign-id>
type BodyCampaignsIDPUT struct {
	Name         string               `json:"name"`
	Detail       string               `json:"detail"`
	Type         cacampaign.Type      `json:"type"`
	ServiceLevel int                  `json:"service_level"`
	EndHandle    cacampaign.EndHandle `json:"end_handle"`
}

// BodyCampaignsIDStatusPUT is rquest body define for
// PUT /v1.0/campaigns/<campaign-id>/status
type BodyCampaignsIDStatusPUT struct {
	Status cacampaign.Status `json:"status"`
}

// BodyCampaignsIDServiceLevelPUT is rquest body define for
// PUT /v1.0/campaigns/<campaign-id>/service_level
type BodyCampaignsIDServiceLevelPUT struct {
	ServiceLevel int `json:"service_level"`
}

// BodyCampaignsIDActionsPUT is rquest body define for
// PUT /v1.0/campaigns/<campaign-id>/actions
type BodyCampaignsIDActionsPUT struct {
	Actions []fmaction.Action `json:"actions"`
}

// BodyCampaignsIDResourceInfoPUT is rquest body define for
// PUT /v1.0/campaigns/<campaign-id>/resource_info
type BodyCampaignsIDResourceInfoPUT struct {
	OutplanID      uuid.UUID `json:"outplan_id"`
	OutdialID      uuid.UUID `json:"outdial_id"`
	QueueID        uuid.UUID `json:"queue_id"`
	NextCampaignID uuid.UUID `json:"next_campaign_id"`
}

// BodyCampaignsIDNextCampaignIDPUT is rquest body define for
// PUT /v1.0/campaigns/<campaign-id>/next_campaign_id
type BodyCampaignsIDNextCampaignIDPUT struct {
	NextCampaignID uuid.UUID `json:"next_campaign_id"`
}

// ParamCampaignsIDCampaigncallsGET is rquest param define for
// GET /v1.0/campaigns/<campaign-id>/campaigncalls
type ParamCampaignsIDCampaigncallsGET struct {
	Pagination
}
