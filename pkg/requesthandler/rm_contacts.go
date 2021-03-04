package requesthandler

import (
	"encoding/json"
	"fmt"
	"net/url"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler/models/rmastcontact"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// RMV1ContactsGet sends the /v1/contacts GET request to registrar-manager
func (r *requestHandler) RMV1ContactsGet(endpoint string) ([]*rmastcontact.AstContact, error) {

	uri := fmt.Sprintf("/v1/contacts?endpoint=%s", url.QueryEscape(endpoint))

	res, err := r.sendRequestRegistrar(uri, rabbitmqhandler.RequestMethodGet, resourceFlowsActions, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 299 {
		return nil, fmt.Errorf("could not find action")
	}

	var tmpContacts []rmastcontact.AstContact
	if err := json.Unmarshal([]byte(res.Data), &tmpContacts); err != nil {
		return nil, err
	}

	var contacts []*rmastcontact.AstContact
	for _, c := range tmpContacts {
		contacts = append(contacts, &c)
	}

	return contacts, nil
}

// RMV1ContactsPut sends the /v1/contacts PUT request to registrar-manager
func (r *requestHandler) RMV1ContactsPut(endpoint string) error {

	uri := fmt.Sprintf("/v1/contacts?endpoint=%s", url.QueryEscape(endpoint))

	res, err := r.sendRequestRegistrar(uri, rabbitmqhandler.RequestMethodPut, resourceCallChannelsHealth, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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
