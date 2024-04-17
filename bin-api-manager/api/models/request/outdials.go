package request

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

// BodyOutdialsPOST is rquest body define for
// POST /v1.0/outdials
type BodyOutdialsPOST struct {
	CampaignID uuid.UUID `json:"campaign_id"`
	Name       string    `json:"name"`
	Detail     string    `json:"detail"`
	Data       string    `json:"data"`
}

// ParamOutdialsGET is rquest param define for
// GET /v1.0/outdials
type ParamOutdialsGET struct {
	Pagination
}

// BodyOutdialsIDPUT is rquest body define for
// PUT /v1.0/outdials/<outdial-id>
type BodyOutdialsIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// BodyOutdialsIDCampaignIDPUT is rquest body define for
// PUT /outdials/<outdial-id>/campaign_id
type BodyOutdialsIDCampaignIDPUT struct {
	CampaignID uuid.UUID `json:"campaign_id"`
}

// BodyOutdialsIDDataPUT is rquest body define for
// PUT /v1.0/outdials/<outdial-id>/data
type BodyOutdialsIDDataPUT struct {
	Data string `json:"data"`
}

// BodyOutdialsIDTargetsPOST is rquest body define for
// POST /v1.0/outdials/<outdial-id>/targets
type BodyOutdialsIDTargetsPOST struct {
	Name         string                 `json:"name"`
	Detail       string                 `json:"detail"`
	Data         string                 `json:"data"`
	Destination0 *commonaddress.Address `json:"destination_0"`
	Destination1 *commonaddress.Address `json:"destination_1"`
	Destination2 *commonaddress.Address `json:"destination_2"`
	Destination3 *commonaddress.Address `json:"destination_3"`
	Destination4 *commonaddress.Address `json:"destination_4"`
}

// ParamOutdialsIDTargetsGET is rquest param define for
// GET /v1.0/outdials/<outdial-id>/targets
type ParamOutdialsIDTargetsGET struct {
	Pagination
}
