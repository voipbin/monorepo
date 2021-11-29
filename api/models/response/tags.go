package response

import "gitlab.com/voipbin/bin-manager/api-manager.git/models/tag"

// BodyTagsGET is response body define for GET /tags
type BodyTagsGET struct {
	Result []*tag.Tag `json:"result"`
	Pagination
}
