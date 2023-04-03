package response

import amtag "gitlab.com/voipbin/bin-manager/agent-manager.git/models/tag"

// BodyTagsGET is response body define for
// GET /v1.0/tags
type BodyTagsGET struct {
	Result []*amtag.WebhookMessage `json:"result"`
	Pagination
}
