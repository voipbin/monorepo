package response

import tmtag "monorepo/bin-tag-manager/models/tag"

// BodyTagsGET is response body define for
// GET /v1.0/tags
type BodyTagsGET struct {
	Result []*tmtag.WebhookMessage `json:"result"`
	Pagination
}
