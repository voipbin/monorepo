package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-registrar-manager/models/astcontact"

	"github.com/gofrs/uuid"
)

// RegistrarV1ContactGets sends the /v1/contacts GET request to registrar-manager
func (r *requestHandler) RegistrarV1ContactGets(ctx context.Context, customerID uuid.UUID, extension string) ([]astcontact.AstContact, error) {

	uri := fmt.Sprintf("/v1/contacts?customer_id=%s&extension=%s", customerID, url.QueryEscape(extension))

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodGet, "flow/actions", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get contact. status: %d", tmp.StatusCode)
	}

	var res []astcontact.AstContact
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// RegistrarV1ContactRefresh refreshes the /v1/contacts by sending the PUT request to registrar-manager
func (r *requestHandler) RegistrarV1ContactRefresh(ctx context.Context, customerID uuid.UUID, extension string) error {

	uri := fmt.Sprintf("/v1/contacts?customer_id=%s&extension=%s", customerID, url.QueryEscape(extension))

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodPut, "call/channels/health", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		return nil
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}
