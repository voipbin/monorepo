package response

import omoutdial "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdial"

// BodyOutdialsGET is rquest body define for GET /outdials
type BodyOutdialsGET struct {
	Result []*omoutdial.WebhookMessage `json:"result"`
	Pagination
}
