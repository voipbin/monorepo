package requesthandler

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/request"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	rmextension "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
)

// RMExtensionCreate sends a request to registrar-manager
// to creating a extension.
// it returns created extension if it succeed.
func (r *requestHandler) RMExtensionCreate(e *rmextension.Extension) (*rmextension.Extension, error) {
	uri := fmt.Sprintf("/v1/extensions")

	data := &request.RMV1DataExtensionsPost{
		UserID:    e.UserID,
		Name:      e.Name,
		Detail:    e.Detail,
		DomainID:  e.DomainID,
		Extension: e.Extension,
		Password:  e.Password,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRegistrar(uri, rabbitmqhandler.RequestMethodPost, resourceRegistrarExtensions, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res rmextension.Extension
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RMExtensionGet sends a request to registrar-manager
// to getting a detail extension info.
// it returns detail extension info if it succeed.
func (r *requestHandler) RMExtensionGet(extensionID uuid.UUID) (*rmextension.Extension, error) {
	uri := fmt.Sprintf("/v1/extensions/%s", extensionID)

	res, err := r.sendRequestRegistrar(uri, rabbitmqhandler.RequestMethodGet, resourceRegistrarExtensions, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var f rmextension.Extension
	if err := json.Unmarshal([]byte(res.Data), &f); err != nil {
		return nil, err
	}

	return &f, nil
}

// RMExtensionDelete sends a request to registrar-manager
// to deleting the domain.
func (r *requestHandler) RMExtensionDelete(extensionID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/extensions/%s", extensionID)

	res, err := r.sendRequestRegistrar(uri, rabbitmqhandler.RequestMethodDelete, resourceRegistrarExtensions, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// RMExtensionUpdate sends a request to registrar-manager
// to update the detail extension info.
// it returns updated extension info if it succeed.
func (r *requestHandler) RMExtensionUpdate(f *rmextension.Extension) (*rmextension.Extension, error) {
	uri := fmt.Sprintf("/v1/extensions/%s", f.ID)

	data := &request.RMV1DataExtensionsIDPut{
		Name:     f.Name,
		Detail:   f.Detail,
		Password: f.Password,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestRegistrar(uri, rabbitmqhandler.RequestMethodPut, resourceRegistrarExtensions, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resDomain rmextension.Extension
	if err := json.Unmarshal([]byte(res.Data), &resDomain); err != nil {
		return nil, err
	}

	return &resDomain, nil
}

// RMExtensionGets sends a request to registrar-manager
// to getting a list of extension info.
// it returns detail list of extension info if it succeed.
func (r *requestHandler) RMExtensionGets(domainID uuid.UUID, pageToken string, pageSize uint64) ([]rmextension.Extension, error) {
	uri := fmt.Sprintf("/v1/extensions?page_token=%s&page_size=%d&domain_id=%s", url.QueryEscape(pageToken), pageSize, domainID)

	res, err := r.sendRequestRegistrar(uri, rabbitmqhandler.RequestMethodGet, resourceRegistrarExtensions, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var f []rmextension.Extension
	if err := json.Unmarshal([]byte(res.Data), &f); err != nil {
		return nil, err
	}

	return f, nil
}
