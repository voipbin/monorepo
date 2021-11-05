package requesthandler

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	rmrequest "gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/listenhandler/models/request"
)

// RMDomainCreate sends a request to registrar-manager
// to creating a domain.
// it returns created domain if it succeed.
func (r *requestHandler) RMDomainCreate(userID uint64, domainName, name, detail string) (*rmdomain.Domain, error) {
	uri := "/v1/domains"

	data := &rmrequest.V1DataDomainsPost{
		UserID:     userID,
		DomainName: domainName,
		Name:       name,
		Detail:     detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRegistrar(uri, rabbitmqhandler.RequestMethodPost, resourceRegistrarDomains, requestTimeoutDefault, 0, ContentTypeJSON, m)
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
func (r *requestHandler) RMDomainGet(domainID uuid.UUID) (*rmdomain.Domain, error) {
	uri := fmt.Sprintf("/v1/domains/%s", domainID)

	res, err := r.sendRequestRegistrar(uri, rabbitmqhandler.RequestMethodGet, resourceRegistrarDomains, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var f rmdomain.Domain
	if err := json.Unmarshal([]byte(res.Data), &f); err != nil {
		return nil, err
	}

	return &f, nil
}

// RMDomainDelete sends a request to registrar-manager
// to deleting the domain.
func (r *requestHandler) RMDomainDelete(domainID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/domains/%s", domainID)

	res, err := r.sendRequestRegistrar(uri, rabbitmqhandler.RequestMethodDelete, resourceRegistrarDomains, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case res == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}

	return nil
}

// RMDomainUpdate sends a request to registrar-manager
// to update the detail domain info.
// it returns updated domain info if it succeed.
func (r *requestHandler) RMDomainUpdate(f *rmdomain.Domain) (*rmdomain.Domain, error) {
	uri := fmt.Sprintf("/v1/domains/%s", f.ID)

	data := &rmrequest.V1DataDomainsIDPut{
		Name:   f.Name,
		Detail: f.Detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestRegistrar(uri, rabbitmqhandler.RequestMethodPut, resourceRegistrarDomains, requestTimeoutDefault, 0, ContentTypeJSON, m)
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
func (r *requestHandler) RMDomainGets(userID uint64, pageToken string, pageSize uint64) ([]rmdomain.Domain, error) {
	uri := fmt.Sprintf("/v1/domains?page_token=%s&page_size=%d&user_id=%d", url.QueryEscape(pageToken), pageSize, userID)

	res, err := r.sendRequestRegistrar(uri, rabbitmqhandler.RequestMethodGet, resourceRegistrarDomains, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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
