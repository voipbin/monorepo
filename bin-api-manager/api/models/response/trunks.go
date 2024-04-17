package response

import rmtrunks "monorepo/bin-registrar-manager/models/trunk"

// BodyTrunksGET is rquest body define for
// GET /v1.0/trunks
type BodyTrunksGET struct {
	Result []*rmtrunks.WebhookMessage `json:"result"`
	Pagination
}
