package response

import rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"

// BodyDomainsGET is rquest body define for GET /domains
type BodyDomainsGET struct {
	Result []*rmdomain.WebhookMessage `json:"result"`
	Pagination
}
