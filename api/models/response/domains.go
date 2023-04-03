package response

import rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"

// BodyDomainsGET is rquest body define for
// GET /v1.0/domains
type BodyDomainsGET struct {
	Result []*rmdomain.WebhookMessage `json:"result"`
	Pagination
}
