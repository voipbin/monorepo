package response

import "gitlab.com/voipbin/bin-manager/api-manager.git/models/extension"

// BodyExtensionsGET is rquest body define for GET /extensions
type BodyExtensionsGET struct {
	Result []*extension.Extension `json:"result"`
	Pagination
}
