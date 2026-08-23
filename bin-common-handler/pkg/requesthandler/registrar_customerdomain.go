package requesthandler

import (
	"context"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	rmcustomerdomain "monorepo/bin-registrar-manager/models/customerdomain"
)

// RegistrarV1CustomerDomainGetByRealm sends a request to registrar-manager
// to getting a detail customer domain info of the given realm.
// it returns detail customer domain info if it succeed.
func (r *requestHandler) RegistrarV1CustomerDomainGetByRealm(ctx context.Context, realm string) (*rmcustomerdomain.CustomerDomain, error) {
	uri := fmt.Sprintf("/v1/customer_domains/realm/%s", url.PathEscape(realm))

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodGet, "registrar/customer_domains/realm", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res rmcustomerdomain.CustomerDomain
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
