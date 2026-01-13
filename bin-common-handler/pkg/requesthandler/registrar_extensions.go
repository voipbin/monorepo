package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	rmextension "monorepo/bin-registrar-manager/models/extension"
	rmrequest "monorepo/bin-registrar-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// RegistrarV1ExtensionCreate sends a request to registrar-manager
// to creating a extension.
// it returns created extension if it succeed.
func (r *requestHandler) RegistrarV1ExtensionCreate(ctx context.Context, customerID uuid.UUID, ext string, password string, name, detail string) (*rmextension.Extension, error) {
	uri := "/v1/extensions"

	data := &rmrequest.V1DataExtensionsPost{
		CustomerID: customerID,
		Extension:  ext,
		Password:   password,
		Name:       name,
		Detail:     detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodPost, "registrar/extension", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmextension.Extension
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RegistrarV1ExtensionGet sends a request to registrar-manager
// to getting a detail extension info.
// it returns detail extension info if it succeed.
func (r *requestHandler) RegistrarV1ExtensionGet(ctx context.Context, extensionID uuid.UUID) (*rmextension.Extension, error) {
	uri := fmt.Sprintf("/v1/extensions/%s", extensionID)

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodGet, "registrar/extension", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res rmextension.Extension
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RegistrarV1ExtensionDelete sends a request to registrar-manager
// to deleting the domain.
func (r *requestHandler) RegistrarV1ExtensionDelete(ctx context.Context, extensionID uuid.UUID) (*rmextension.Extension, error) {
	uri := fmt.Sprintf("/v1/extensions/%s", extensionID)

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodDelete, "registrar/extension", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res rmextension.Extension
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RegistrarV1ExtensionUpdate sends a request to registrar-manager
// to update the detail extension info.
// it returns updated extension info if it succeed.
func (r *requestHandler) RegistrarV1ExtensionUpdate(ctx context.Context, id uuid.UUID, name string, detail string, password string) (*rmextension.Extension, error) {
	uri := fmt.Sprintf("/v1/extensions/%s", id)

	data := &rmrequest.V1DataExtensionsIDPut{
		Name:     name,
		Detail:   detail,
		Password: password,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodPut, "registrar/extension", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmextension.Extension
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RegistrarV1ExtensionGets sends a request to registrar-manager
// to getting a list of extension info.
// it returns detail list of extension info if it succeed.
func (r *requestHandler) RegistrarV1ExtensionGets(ctx context.Context, pageToken string, pageSize uint64, filters map[rmextension.Field]any) ([]rmextension.Extension, error) {
	uri := fmt.Sprintf("/v1/extensions?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodGet, "registrar/extension", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []rmextension.Extension
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// RegistrarV1ExtensionGetByExtension sends a request to registrar-manager
// to getting a detail extension info of the given extension.
// it returns detail extension info if it succeed.
func (r *requestHandler) RegistrarV1ExtensionGetByExtension(ctx context.Context, customerID uuid.UUID, extension string) (*rmextension.Extension, error) {
	uri := fmt.Sprintf("/v1/extensions/extension/%s?customer_id=%s", extension, customerID.String())

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodGet, "registrar/extension", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res rmextension.Extension
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
