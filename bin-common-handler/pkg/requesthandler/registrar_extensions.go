package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	rmextension "monorepo/bin-registrar-manager/models/extension"
	rmextensiondirect "monorepo/bin-registrar-manager/models/extensiondirect"
	rmrequest "monorepo/bin-registrar-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

type extensionCountByCustomerRequest struct {
	CustomerID uuid.UUID `json:"customer_id"`
}

type extensionCountByCustomerResponse struct {
	Count int `json:"count"`
}

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

// RegistrarV1ExtensionList sends a request to registrar-manager
// to getting a list of extension info.
// it returns detail list of extension info if it succeed.
func (r *requestHandler) RegistrarV1ExtensionList(ctx context.Context, pageToken string, pageSize uint64, filters map[rmextension.Field]any) ([]rmextension.Extension, error) {
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
func (r *requestHandler) RegistrarV1ExtensionGetByExtension(ctx context.Context, extension string, filters map[rmextension.Field]any) (*rmextension.Extension, error) {
	uri := fmt.Sprintf("/v1/extensions/extension/%s", extension)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodGet, "registrar/extension", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmextension.Extension
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RegistrarV1ExtensionCountByCustomerID sends a request to registrar-manager
// to get the count of extensions for the given customer.
func (r *requestHandler) RegistrarV1ExtensionCountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error) {
	uri := "/v1/extensions/count_by_customer"

	m, err := json.Marshal(extensionCountByCustomerRequest{
		CustomerID: customerID,
	})
	if err != nil {
		return 0, err
	}

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodGet, "registrar/extensions/count_by_customer", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return 0, err
	}

	var res extensionCountByCustomerResponse
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return 0, errParse
	}

	return res.Count, nil
}

// RegistrarV1ExtensionGetByDirectHash sends a request to registrar-manager
// to get the extension corresponding to the given direct hash.
// It resolves hash → extension in a single RPC call.
func (r *requestHandler) RegistrarV1ExtensionGetByDirectHash(ctx context.Context, hash string) (*rmextension.Extension, error) {
	uri := fmt.Sprintf("/v1/extensions/by-direct-hash/%s", url.PathEscape(hash))

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

// RegistrarV1ExtensionDirectGetByHash sends a request to registrar-manager
// to get the extension direct by hash.
func (r *requestHandler) RegistrarV1ExtensionDirectGetByHash(ctx context.Context, hash string) (*rmextensiondirect.ExtensionDirect, error) {
	uri := fmt.Sprintf("/v1/extension-directs?hash=%s", url.QueryEscape(hash))

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodGet, "registrar/extension-direct", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res rmextensiondirect.ExtensionDirect
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
