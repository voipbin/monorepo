package response

import rmextension "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"

// BodyExtensionsGET is rquest body define for
// GET /v1.0/extensions
type BodyExtensionsGET struct {
	Result []*rmextension.WebhookMessage `json:"result"`
	Pagination
}
