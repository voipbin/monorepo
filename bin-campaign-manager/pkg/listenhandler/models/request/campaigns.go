package request

import (
	"github.com/gofrs/uuid"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
)

// V1DataCampaignsPost is
// v1 data type request struct for
// /v1/campaigns POST
type V1DataCampaignsPost struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Type campaign.Type `json:"type"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	ServiceLevel int                `json:"service_level"`
	EndHandle    campaign.EndHandle `json:"end_handle"`

	// action settings
	Actions []fmaction.Action `json:"actions"`

	// resource info
	OutplanID uuid.UUID `json:"outplan_id"`
	OutdialID uuid.UUID `json:"outdial_id"`
	QueueID   uuid.UUID `json:"queue_id"`

	NextCampaignID uuid.UUID `json:"next_campaign_id"`
}

// V1DataCampaignsIDPut is
// v1 data type request struct for
// /v1/campaigns/<campaign-id> PUT
type V1DataCampaignsIDPut struct {
	Name         string             `json:"name,omitempty"`
	Detail       string             `json:"detail,omitempty"`
	Type         campaign.Type      `json:"type"`
	ServiceLevel int                `json:"service_level,omitempty"`
	EndHandle    campaign.EndHandle `json:"end_handle,omitempty"`
}

// V1DataCampaignsIDStatusPut is
// v1 data type request struct for
// /v1/campaigns/<campaign-id>/status PUT
type V1DataCampaignsIDStatusPut struct {
	Status campaign.Status `json:"status"`
}

// V1DataCampaignsIDServiceLevelPut is
// v1 data type request struct for
// /v1/campaigns/<campaign-id>/service_level PUT
type V1DataCampaignsIDServiceLevelPut struct {
	ServiceLevel int `json:"service_level"`
}

// V1DataCampaignsIDActionsPut is
// v1 data type request struct for
// /v1/campaigns/<campaign-id>/actions PUT
type V1DataCampaignsIDActionsPut struct {
	Actions []fmaction.Action `json:"actions"`
}

// V1DataCampaignsIDResourceInfoPut is
// v1 data type request struct for
// /v1/campaigns/<campaign-id>/resource_info PUT
type V1DataCampaignsIDResourceInfoPut struct {
	OutplanID      uuid.UUID `json:"outplan_id"`
	OutdialID      uuid.UUID `json:"outdial_id"`
	QueueID        uuid.UUID `json:"queue_id"`
	NextCampaignID uuid.UUID `json:"next_campaign_id"`
}

// V1DataCampaignsIDNextCampaignIDPut is
// v1 data type request struct for
// /v1/campaigns/<campaign-id>/next_campaign_id PUT
type V1DataCampaignsIDNextCampaignIDPut struct {
	NextCampaignID uuid.UUID `json:"next_campaign_id"`
}
