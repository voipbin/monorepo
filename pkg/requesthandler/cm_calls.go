package requesthandler

import (
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager/pkg/rabbitmq/models"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler/models/cmcall"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler/models/request"
)

// CMCallCreate sends a request to call-manager
// to creating a call.
// it returns created call if it succeed.
func (r *requestHandler) CMCallCreate(userID uint64, flowID uuid.UUID, source, destination cmcall.Address) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls")

	data := &request.V1DataCallsIDPost{
		FlowID:      flowID,
		Source:      source,
		Destination: destination,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestCall(uri, models.RequestMethodPost, resourceCallCall, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var c cmcall.Call
	if err := json.Unmarshal([]byte(res.Data), &c); err != nil {
		return nil, err
	}

	return &c, nil
}

// CMCallGet sends a request to call-manager
// to creating a call.
// it returns created call if it succeed.
func (r *requestHandler) CMCallGet(callID uuid.UUID) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls/%s", callID)

	res, err := r.sendRequestCall(uri, models.RequestMethodPost, resourceCallCall, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var c cmcall.Call
	if err := json.Unmarshal([]byte(res.Data), &c); err != nil {
		return nil, err
	}

	return &c, nil
}
