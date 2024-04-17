package response

import rmextension "monorepo/bin-registrar-manager/models/extension"

// BodyExtensionsGET is rquest body define for
// GET /v1.0/extensions
type BodyExtensionsGET struct {
	Result []*rmextension.WebhookMessage `json:"result"`
	Pagination
}
