package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	cmrequest "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// CMV1ConfbridgeCreate sends the request for confbridge create.
// conferenceID: conference id
func (r *requestHandler) CMV1ConfbridgeCreate(ctx context.Context, confbridgeType cmconfbridge.Type) (*cmconfbridge.Confbridge, error) {
	uri := "/v1/confbridges"

	m, err := json.Marshal(cmrequest.V1DataConfbridgesPost{
		Type: confbridgeType,
	})
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestCM(uri, rabbitmqhandler.RequestMethodPost, resourceCMConfbridges, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("no response found")
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var cb cmconfbridge.Confbridge
	if errUnmarshal := json.Unmarshal([]byte(res.Data), &cb); errUnmarshal != nil {
		return nil, errUnmarshal
	}

	return &cb, nil
}

// CMV1ConfbridgeDelete sends the request for confbridge delete.
// conferenceID: conference id
func (r *requestHandler) CMV1ConfbridgeDelete(ctx context.Context, conferenceID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/confbridges/%s", conferenceID)

	res, err := r.sendRequestCM(uri, rabbitmqhandler.RequestMethodDelete, resourceCMConfbridges, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// CMV1ConfbridgeCallKick sends the kick request to the confbridge.
// conferenceID: conference id
// callID: call id
func (r *requestHandler) CMV1ConfbridgeCallKick(ctx context.Context, conferenceID uuid.UUID, callID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/confbridges/%s/calls/%s", conferenceID, callID)

	res, err := r.sendRequestCM(uri, rabbitmqhandler.RequestMethodDelete, resourceCMConfbridges, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// CMV1ConfbridgeCallAdd sends the call join request to the confbridge.
// conferenceID: conference id
// callID: call id
func (r *requestHandler) CMV1ConfbridgeCallAdd(ctx context.Context, conferenceID uuid.UUID, callID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/confbridges/%s/calls/%s", conferenceID, callID)

	res, err := r.sendRequestCM(uri, rabbitmqhandler.RequestMethodPost, resourceCMConfbridges, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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
