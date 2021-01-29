package requesthandler

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/cmcall"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/request"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// CMCallCreate sends a request to call-manager
// to creating a call.
// it returns created call if it succeed.
func (r *requestHandler) CMCallCreate(userID uint64, flowID uuid.UUID, source, destination cmcall.Address) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls")

	data := &request.V1DataCallsIDPost{
		UserID:      userID,
		FlowID:      flowID,
		Source:      source,
		Destination: destination,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestCall(uri, rabbitmqhandler.RequestMethodPost, resourceCallCall, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

	res, err := r.sendRequestCall(uri, rabbitmqhandler.RequestMethodGet, resourceCallCall, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// CMCallGets sends a request to call-manager
// to getting a list of call info.
// it returns detail list of call info if it succeed.
func (r *requestHandler) CMCallGets(userID uint64, pageToken string, pageSize uint64) ([]cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls?page_token=%s&page_size=%d&user_id=%d", url.QueryEscape(pageToken), pageSize, userID)

	res, err := r.sendRequestCall(uri, rabbitmqhandler.RequestMethodGet, resourceCallCall, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var calls []cmcall.Call
	if err := json.Unmarshal([]byte(res.Data), &calls); err != nil {
		return nil, err
	}

	return calls, nil
}

// CMCallDelete sends a request to call-manager
// to hangup the call.
// it returns 200 if it succeed.
func (r *requestHandler) CMCallDelete(callID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/calls/%s", callID)

	res, err := r.sendRequestCall(uri, rabbitmqhandler.RequestMethodDelete, resourceCallCall, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case res == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}

	return nil
}
