package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	cmrequest "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/request"
	cmresponse "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/response"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// CallV1CallHealth sends the request for call health-check
func (r *requestHandler) CallV1CallHealth(ctx context.Context, id uuid.UUID, delay, retryCount int) error {
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

	res, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCallsHealth, requestTimeoutDefault, delay, ContentTypeJSON, m)
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

// CallV1CallActionTimeout sends the request for call's action timeout.
//
// delay: millisecond
func (r *requestHandler) CallV1CallActionTimeout(ctx context.Context, id uuid.UUID, delay int, a *action.Action) error {
	uri := fmt.Sprintf("/v1/calls/%s/action-timeout", id)

	m, err := json.Marshal(cmrequest.V1DataCallsIDActionTimeoutPost{
		ActionID:   a.ID,
		ActionType: a.Type,
		TMExecute:  a.TMExecute,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCallsActionTimeout, requestTimeoutDefault, delay, ContentTypeJSON, m)
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

// CallV1CallActionNext sends the request for call's action next.
//
// delay: millisecond
func (r *requestHandler) CallV1CallActionNext(ctx context.Context, callID uuid.UUID, force bool) error {
	uri := fmt.Sprintf("/v1/calls/%s/action-next", callID)

	m, err := json.Marshal(cmrequest.V1DataCallsIDActionNextPost{
		Force: force,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCallsActionNext, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// CallV1CallCreate sends a request to call-manager
// to creating a call.
// it returns created call if it succeed.
func (r *requestHandler) CallV1CallsCreate(ctx context.Context, customerID, flowID, masterCallID uuid.UUID, source *address.Address, destinations []address.Address) ([]cmcall.Call, error) {
	uri := "/v1/calls"

	data := &cmrequest.V1DataCallsPost{
		CustomerID:   customerID,
		FlowID:       flowID,
		MasterCallID: masterCallID,
		Source:       *source,
		Destinations: destinations,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCalls, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []cmcall.Call
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// CallV1CallCreateWithID sends a request to call-manager
// to creating a call with the given id.
// it returns created call if it succeed.
func (r *requestHandler) CallV1CallCreateWithID(ctx context.Context, id, customerID, flowID, activeflowID, masterCallID uuid.UUID, source, destination *address.Address) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls/%s", id.String())

	data := &cmrequest.V1DataCallsIDPost{
		CustomerID:   customerID,
		FlowID:       flowID,
		ActiveflosID: activeflowID,
		MasterCallID: masterCallID,
		Source:       *source,
		Destination:  *destination,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCalls, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// CallV1CallGet sends a request to call-manager
// to getting a call.
// it returns given call id's call if it succeed.
func (r *requestHandler) CallV1CallGet(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls/%s", callID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceCallCalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmcall.Call
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallV1CallGets sends a request to call-manager
// to getting a list of call info.
// it returns detail list of call info if it succeed.
func (r *requestHandler) CallV1CallGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceCallCalls, 30000, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []cmcall.Call
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// CallV1CallDelete sends a request to call-manager
// to delete the call.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallDelete(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls/%s", callID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceCallCalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmcall.Call
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallV1CallHangup sends a request to call-manager
// to hangup the call.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallHangup(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls/%s/hangup", callID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmcall.Call
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallV1CallAddChainedCall sends a request to call-manager
// to add the chained call to the call.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallAddChainedCall(ctx context.Context, callID uuid.UUID, chainedCallID uuid.UUID) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls/%s/chained-call-ids", callID)

	data := &cmrequest.V1DataCallsIDChainedCallIDsPost{
		ChainedCallID: chainedCallID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCalls, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// CallV1CallRemoveChainedCall sends a request to call-manager
// to remove the chained call to the call.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallRemoveChainedCall(ctx context.Context, callID uuid.UUID, chainedCallID uuid.UUID) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls/%s/chained-call-ids/%s", callID, chainedCallID)

	res, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceCallCalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// CallV1CallExternalMediaStart sends a request to call-manager
// to start the external media.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallExternalMediaStart(
	ctx context.Context,
	callID uuid.UUID,
	externalHost string, // external host:port
	encapsulation string, // rtp
	transport string, // udp
	connectionType string, // client,server
	format string, // ulaw
	direction string, // in,out,both
) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls/%s/external-media", callID)

	reqData := &cmrequest.V1DataCallsIDExternalMediaPost{
		ExternalHost:   externalHost,
		Encapsulation:  encapsulation,
		Transport:      transport,
		ConnectionType: connectionType,
		Format:         format,
		Direction:      direction,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCalls, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmcall.Call
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallV1CallExternalMediaStop sends a request to call-manager
// to stop the external media.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallExternalMediaStop(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls/%s/external-media", callID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceCallCallsExternalMedia, requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmcall.Call
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallV1CallGetDigits sends a request to call-manager
// to get received digits of the call.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallGetDigits(ctx context.Context, callID uuid.UUID) (string, error) {
	uri := fmt.Sprintf("/v1/calls/%s/digits", callID)

	res, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceCallCalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return "", err
	case res == nil:
		// not found
		return "", fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return "", fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData cmresponse.V1ResponseCallsIDDigitsGet
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return "", err
	}

	return resData.Digits, nil
}

// CallV1CallSendDigits sends a request to call-manager
// to send the digits to the call.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallSendDigits(ctx context.Context, callID uuid.UUID, digits string) error {
	uri := fmt.Sprintf("/v1/calls/%s/digits", callID)

	reqData := &cmrequest.V1DataCallsIDDigitsPost{
		Digits: digits,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return err
	}

	res, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCalls, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// CallV1CallRecordingStart sends a request to call-manager
// to starts the given call's recording.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallRecordingStart(ctx context.Context, callID uuid.UUID, format cmrecording.Format, endOfSilence int, endOfKey string, duration int) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls/%s/recording_start", callID)

	reqData := &cmrequest.V1DataCallsIDRecordingStartPost{
		Format:       format,
		EndOfSilence: endOfSilence,
		EndOfKey:     endOfKey,
		Duration:     duration,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCallsRecordingStart, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmcall.Call
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallV1CallRecordingStop sends a request to call-manager
// to starts the given call's recording.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallRecordingStop(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls/%s/recording_stop", callID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCallsRecordingStop, requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmcall.Call
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
