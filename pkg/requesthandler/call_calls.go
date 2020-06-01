package requesthandler

import (
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/request"
)

// CallCallsHealth sends the request for call health-check
func (r *requestHandler) CallCallHealth(id uuid.UUID, delay, retryCount int) error {
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

	res, err := r.sendRequestCall(uri, rabbitmq.RequestMethodPost, resourceCallCallsHealth, requestTimeoutDefault, delay, ContentTypeJSON, string(m))
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

// CallCallActionTimeout sends the request for call's action timeout.
//
// delay: millisecond
func (r *requestHandler) CallCallActionTimeout(id uuid.UUID, delay int, a *action.Action) error {
	uri := fmt.Sprintf("/v1/calls/%s/action-timeout", id)

	m, err := json.Marshal(request.V1DataCallsIDActionTimeout{
		ActionID:   a.ID,
		ActionType: a.Type,
		TMExecute:  a.TMExecute,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestCall(uri, rabbitmq.RequestMethodPost, resourceCallCallsActionTimeout, requestTimeoutDefault, delay, ContentTypeJSON, string(m))
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

// CallCallActionNext sends the request for call's action next.
//
// delay: millisecond
func (r *requestHandler) CallCallActionNext(id uuid.UUID) error {
	uri := fmt.Sprintf("/v1/calls/%s/action-next", id)

	res, err := r.sendRequestCall(uri, rabbitmq.RequestMethodPost, resourceCallCallsActionNext, requestTimeoutDefault, 0, ContentTypeJSON, "")
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
