package response

import (
	omoutdial "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdial"
	omoutdialtarget "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"
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
