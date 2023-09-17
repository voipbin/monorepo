package response

import rmtrunks "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/trunk"

// BodyTrunksGET is rquest body define for
// GET /v1.0/trunks
type BodyTrunksGET struct {
	Result []*rmtrunks.WebhookMessage `json:"result"`
	Pagination
}
