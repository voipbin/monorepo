package response

import rmextension "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"

// BodyExtensionsGET is rquest body define for GET /extensions
type BodyExtensionsGET struct {
	Result []*rmextension.WebhookMessage `json:"result"`
	Pagination
}
