package response

import tmtag "gitlab.com/voipbin/bin-manager/tag-manager.git/models/tag"

// BodyTagsGET is response body define for
// GET /v1.0/tags
type BodyTagsGET struct {
	Result []*tmtag.WebhookMessage `json:"result"`
	Pagination
}
