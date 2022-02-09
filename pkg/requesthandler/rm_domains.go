package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	rmrequest "gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// RMDomainCreate sends a request to registrar-manager
// to creating a domain.
// it returns created domain if it succeed.
func (r *requestHandler) RMV1DomainCreate(ctx context.Context, customerID uuid.UUID, domainName, name, detail string) (*rmdomain.Domain, error) {
	uri := "/v1/domains"

	data := &rmrequest.V1DataDomainsPost{
		CustomerID: customerID,
		DomainName: domainName,
		Name:       name,
		Detail:     detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRM(uri, rabbitmqhandler.RequestMethodPost, resourceRegistrarDomains, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res rmdomain.Domain
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RMDomainGet sends a request to registrar-manager
// to getting a detail domain info.
// it returns detail domain info if it succeed.
func (r *requestHandler) RMV1DomainGet(ctx context.Context, domainID uuid.UUID) (*rmdomain.Domain, error) {
	uri := fmt.Sprintf("/v1/domains/%s", domainID)

	tmp, err := r.sendRequestRM(uri, rabbitmqhandler.RequestMethodGet, resourceRegistrarDomains, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res rmdomain.Domain
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RMDomainDelete sends a request to registrar-manager
// to deleting the domain.
func (r *requestHandler) RMV1DomainDelete(ctx context.Context, domainID uuid.UUID) (*rmdomain.Domain, error) {
	uri := fmt.Sprintf("/v1/domains/%s", domainID)

	tmp, err := r.sendRequestRM(uri, rabbitmqhandler.RequestMethodDelete, resourceRegistrarDomains, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res rmdomain.Domain
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RMDomainUpdate sends a request to registrar-manager
// to update the detail domain info.
// it returns updated domain info if it succeed.
func (r *requestHandler) RMV1DomainUpdate(ctx context.Context, id uuid.UUID, name, detail string) (*rmdomain.Domain, error) {
	uri := fmt.Sprintf("/v1/domains/%s", id)

	data := &rmrequest.V1DataDomainsIDPut{
		Name:   name,
		Detail: detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestRM(uri, rabbitmqhandler.RequestMethodPut, resourceRegistrarDomains, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resDomain rmdomain.Domain
	if err := json.Unmarshal([]byte(res.Data), &resDomain); err != nil {
		return nil, err
	}

	return &resDomain, nil
}

// RMDomainGets sends a request to registrar-manager
// to getting a list of domain info.
// it returns detail list of domain info if it succeed.
func (r *requestHandler) RMV1DomainGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]rmdomain.Domain, error) {
	uri := fmt.Sprintf("/v1/domains?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	res, err := r.sendRequestRM(uri, rabbitmqhandler.RequestMethodGet, resourceRegistrarDomains, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var f []rmdomain.Domain
	if err := json.Unmarshal([]byte(res.Data), &f); err != nil {
		return nil, err
	}

	return f, nil
}
