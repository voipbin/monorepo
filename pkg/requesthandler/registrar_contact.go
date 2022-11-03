package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// RegistrarV1ContactGets sends the /v1/contacts GET request to registrar-manager
func (r *requestHandler) RegistrarV1ContactGets(ctx context.Context, endpoint string) ([]*astcontact.AstContact, error) {

	uri := fmt.Sprintf("/v1/contacts?endpoint=%s", url.QueryEscape(endpoint))

	tmp, err := r.sendRequestRegistrar(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceFlowActions, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get contact. status: %d", tmp.StatusCode)
	}

	var tmpContacts []astcontact.AstContact
	if err := json.Unmarshal([]byte(tmp.Data), &tmpContacts); err != nil {
		return nil, err
	}

	var res []*astcontact.AstContact
	for _, c := range tmpContacts {
		res = append(res, &c)
	}

	return res, nil
}

// RegistrarV1ContactUpdate sends the /v1/contacts PUT request to registrar-manager
func (r *requestHandler) RegistrarV1ContactUpdate(ctx context.Context, endpoint string) error {

	uri := fmt.Sprintf("/v1/contacts?endpoint=%s", url.QueryEscape(endpoint))

	tmp, err := r.sendRequestRegistrar(ctx, uri, rabbitmqhandler.RequestMethodPut, resourceCallChannelsHealth, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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
