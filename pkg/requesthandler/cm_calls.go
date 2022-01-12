package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmrequest "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/request"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// CMV1CallHealth sends the request for call health-check
func (r *requestHandler) CMV1CallHealth(ctx context.Context, id uuid.UUID, delay, retryCount int) error {
	uri := fmt.Sprintf("/v1/calls/%s/health-check", id)

	type Data struct {
		RetryCount int `json:"retry_count"`
		Delay      int `json:"delay"`
	}

	m, err := json.Marshal(Data{
		retryCount,
		delay,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestCM(uri, rabbitmqhandler.RequestMethodPost, resourceCMCallsHealth, requestTimeoutDefault, delay, ContentTypeJSON, m)
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

// CMV1CallActionTimeout sends the request for call's action timeout.
//
// delay: millisecond
func (r *requestHandler) CMV1CallActionTimeout(ctx context.Context, id uuid.UUID, delay int, a *action.Action) error {
	uri := fmt.Sprintf("/v1/calls/%s/action-timeout", id)

	m, err := json.Marshal(cmrequest.V1DataCallsIDActionTimeoutPost{
		ActionID:   a.ID,
		ActionType: a.Type,
		TMExecute:  a.TMExecute,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestCM(uri, rabbitmqhandler.RequestMethodPost, resourceCMCallsActionTimeout, requestTimeoutDefault, delay, ContentTypeJSON, m)
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

// CMV1CallActionNext sends the request for call's action next.
//
// delay: millisecond
func (r *requestHandler) CMV1CallActionNext(ctx context.Context, callID uuid.UUID, force bool) error {
	uri := fmt.Sprintf("/v1/calls/%s/action-next", callID)

	m, err := json.Marshal(cmrequest.V1DataCallsIDActionNextPost{
		Force: force,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestCM(uri, rabbitmqhandler.RequestMethodPost, resourceCMCallsActionNext, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// CMV1CallCreate sends a request to call-manager
// to creating a call.
// it returns created call if it succeed.
func (r *requestHandler) CMV1CallCreate(ctx context.Context, userID uint64, flowID uuid.UUID, source, destination *cmaddress.Address) (*cmcall.Call, error) {
	uri := "/v1/calls"

	data := &cmrequest.V1DataCallsPost{
		UserID:      userID,
		FlowID:      flowID,
		Source:      *source,
		Destination: *destination,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestCM(uri, rabbitmqhandler.RequestMethodPost, resourceCMCall, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// CMV1CallCreateWithID sends a request to call-manager
// to creating a call with the given id.
// it returns created call if it succeed.
func (r *requestHandler) CMV1CallCreateWithID(ctx context.Context, id uuid.UUID, userID uint64, flowID uuid.UUID, source, destination *cmaddress.Address) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls/%s", id.String())

	data := &cmrequest.V1DataCallsIDPost{
		UserID:      userID,
		FlowID:      flowID,
		Source:      *source,
		Destination: *destination,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestCM(uri, rabbitmqhandler.RequestMethodPost, resourceCMCall, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// CMV1CallGet sends a request to call-manager
// to creating a call.
// it returns created call if it succeed.
func (r *requestHandler) CMV1CallGet(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls/%s", callID)

	res, err := r.sendRequestCM(uri, rabbitmqhandler.RequestMethodGet, resourceCMCall, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// CMV1CallGets sends a request to call-manager
// to getting a list of call info.
// it returns detail list of call info if it succeed.
func (r *requestHandler) CMV1CallGets(ctx context.Context, userID uint64, pageToken string, pageSize uint64) ([]cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls?page_token=%s&page_size=%d&user_id=%d", url.QueryEscape(pageToken), pageSize, userID)

	res, err := r.sendRequestCM(uri, rabbitmqhandler.RequestMethodGet, resourceCMCall, 30, 0, ContentTypeJSON, nil)
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

// CMV1CallHangup sends a request to call-manager
// to hangup the call.
// it returns error if something went wrong.
func (r *requestHandler) CMV1CallHangup(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls/%s", callID)

	res, err := r.sendRequestCM(uri, rabbitmqhandler.RequestMethodDelete, resourceCMCall, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// CMV1CallAddChainedCall sends a request to call-manager
// to add the chained call to the call.
// it returns error if something went wrong.
func (r *requestHandler) CMV1CallAddChainedCall(ctx context.Context, callID uuid.UUID, chainedCallID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/calls/%s/chained-call-ids", callID)

	data := &cmrequest.V1DataCallsIDChainedCallIDsPost{
		ChainedCallID: chainedCallID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return err
	}

	res, err := r.sendRequestCM(uri, rabbitmqhandler.RequestMethodPost, resourceCMCall, requestTimeoutDefault, 0, ContentTypeJSON, m)
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
