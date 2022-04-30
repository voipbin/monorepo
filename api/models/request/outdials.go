package request

import (
	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
)

// BodyOutdialsPOST is rquest body define for POST /outdials
type BodyOutdialsPOST struct {
	CampaignID uuid.UUID `json:"campaign_id"`
	Name       string    `json:"name"`
	Detail     string    `json:"detail"`
	Data       string    `json:"data"`
}

// ParamOutdialsGET is rquest param define for GET /outdials
type ParamOutdialsGET struct {
	Pagination
}

// BodyOutdialsIDPUT is rquest body define for PUT /outdials/{id}
type BodyOutdialsIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// BodyOutdialsIDCampaignIDPUT is rquest body define for PUT /outdials/{id}/campaign_id
type BodyOutdialsIDCampaignIDPUT struct {
	CampaignID uuid.UUID `json:"campaign_id"`
}

// BodyOutdialsIDDataPUT is rquest body define for PUT /outdials/{id}/data
type BodyOutdialsIDDataPUT struct {
	Data string `json:"data"`
}

// BodyOutdialsIDTargetsPOST is rquest body define for POST /outdials/{id}/targets
type BodyOutdialsIDTargetsPOST struct {
	Name         string             `json:"name"`
	Detail       string             `json:"detail"`
	Data         string             `json:"data"`
	Destination0 *cmaddress.Address `json:"destination_0"`
	Destination1 *cmaddress.Address `json:"destination_1"`
	Destination2 *cmaddress.Address `json:"destination_2"`
	Destination3 *cmaddress.Address `json:"destination_3"`
	Destination4 *cmaddress.Address `json:"destination_4"`
}

// ParamOutdialsIDTargetsGET is rquest param define for GET /outdials/{id}/targets
type ParamOutdialsIDTargetsGET struct {
	Pagination
}
