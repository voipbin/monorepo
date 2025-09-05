package requesthandler

import (
	"context"
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

	var res []astcontact.AstContact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// RegistrarV1ContactRefresh refreshes the /v1/contacts by sending the PUT request to registrar-manager
func (r *requestHandler) RegistrarV1ContactRefresh(ctx context.Context, customerID uuid.UUID, extension string) error {

	uri := fmt.Sprintf("/v1/contacts?customer_id=%s&extension=%s", customerID, url.QueryEscape(extension))

	tmp, err := r.sendRequestRegistrar(ctx, uri, sock.RequestMethodPut, "call/channels/health", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}
