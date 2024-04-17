package response

import (
	omoutdial "monorepo/bin-outdial-manager/models/outdial"
	omoutdialtarget "monorepo/bin-outdial-manager/models/outdialtarget"
)

// BodyOutdialsGET is rquest body define for
// GET /v1.0/outdials
type BodyOutdialsGET struct {
	Result []*omoutdial.WebhookMessage `json:"result"`
	Pagination
}

// BodyOutdialsIDTargetsGET is rquest body define for
// GET /v1.0/outdials/{id}/targets
type BodyOutdialsIDTargetsGET struct {
	Result []*omoutdialtarget.WebhookMessage `json:"result"`
	Pagination
}
