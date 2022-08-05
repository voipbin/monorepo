package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// RMV1ContactGets sends the /v1/contacts GET request to registrar-manager
func (r *requestHandler) RMV1ContactGets(ctx context.Context, endpoint string) ([]*astcontact.AstContact, error) {

	uri := fmt.Sprintf("/v1/contacts?endpoint=%s", url.QueryEscape(endpoint))

	tmp, err := r.sendRequestRM(uri, rabbitmqhandler.RequestMethodGet, resourceFlowsActions, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// RMV1ContactUpdate sends the /v1/contacts PUT request to registrar-manager
func (r *requestHandler) RMV1ContactUpdate(ctx context.Context, endpoint string) error {

	uri := fmt.Sprintf("/v1/contacts?endpoint=%s", url.QueryEscape(endpoint))

	res, err := r.sendRequestRM(uri, rabbitmqhandler.RequestMethodPut, resourceCMChannelsHealth, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case res == nil:
		return nil
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}

	return nil
}
