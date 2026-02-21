package requesthandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-registrar-manager/models/astcontact"

	"github.com/pkg/errors"
)

// RegistrarV1ContactList sends the /v1/contacts GET request to registrar-manager
func (r *requestHandler) RegistrarV1ContactList(ctx context.Context, filters map[string]any) ([]astcontact.AstContact, error) {

	uri := "/v1/contacts"

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodGet, "flow/actions", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []astcontact.AstContact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// RegistrarV1ContactRefresh refreshes the /v1/contacts by sending the PUT request to registrar-manager
func (r *requestHandler) RegistrarV1ContactRefresh(ctx context.Context, filters map[string]any) error {

	uri := "/v1/contacts"

	m, err := json.Marshal(filters)
	if err != nil {
		return errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodPut, "call/channels/health", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}
