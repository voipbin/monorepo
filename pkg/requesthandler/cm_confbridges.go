package requesthandler

import (
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/request"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// CMConfbridgesPost sends the request for confbridge create.
// conferenceID: conference id
func (r *requestHandler) CMConfbridgesPost(conferenceID uuid.UUID) (*confbridge.Confbridge, error) {
	uri := "/v1/confbridges"

	m, err := json.Marshal(request.V1DataConfbridgesPost{
		ConferenceID: conferenceID,
	})
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestCall(uri, rabbitmqhandler.RequestMethodPost, resourceCMConfbridges, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("no response found")
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var cb confbridge.Confbridge
	if errUnmarshal := json.Unmarshal([]byte(res.Data), &cb); errUnmarshal != nil {
		return nil, errUnmarshal
	}

	return &cb, nil
}

// CMConfbridgesIDDelete sends the request for confbridge delete.
// conferenceID: conference id
func (r *requestHandler) CMConfbridgesIDDelete(conferenceID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/confbridges/%s", conferenceID)

	res, err := r.sendRequestCall(uri, rabbitmqhandler.RequestMethodDelete, resourceCMConfbridges, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case res == nil:
		return fmt.Errorf("no response found")
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}

	return nil
}

// CMConfbridgesIDCallsIDDelete sends the kick request to the confbridge.
// conferenceID: conference id
// callID: call id
func (r *requestHandler) CMConfbridgesIDCallsIDDelete(conferenceID uuid.UUID, callID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/confbridges/%s/calls/%s", conferenceID, callID)

	res, err := r.sendRequestCall(uri, rabbitmqhandler.RequestMethodDelete, resourceCMConfbridges, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case res == nil:
		return fmt.Errorf("no response found")
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}

	return nil
}

// CMConfbridgesIDCallsIDPost sends the call join request to the confbridge.
// conferenceID: conference id
// callID: call id
func (r *requestHandler) CMConfbridgesIDCallsIDPost(conferenceID uuid.UUID, callID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/confbridges/%s/calls/%s", conferenceID, callID)

	res, err := r.sendRequestCall(uri, rabbitmqhandler.RequestMethodPost, resourceCMConfbridges, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case res == nil:
		return fmt.Errorf("no response found")
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}

	return nil
}
