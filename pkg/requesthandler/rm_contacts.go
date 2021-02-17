package requesthandler

import (
	"encoding/json"
	"fmt"
	"net/url"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler/models/rmastcontact"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func (r *requestHandler) RMV1ContactsGet(endpoint string) ([]*rmastcontact.AstContact, error) {

	uri := fmt.Sprintf("/v1/contacts?endpoint=%s", url.QueryEscape(endpoint))

	res, err := r.sendRequestRegistrar(uri, rabbitmqhandler.RequestMethodGet, resourceFlowsActions, requestTimeoutDefault, ContentTypeJSON, nil)
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
