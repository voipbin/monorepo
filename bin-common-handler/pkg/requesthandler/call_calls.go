package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	cmrecording "monorepo/bin-call-manager/models/recording"
	cmrequest "monorepo/bin-call-manager/pkg/listenhandler/models/request"
	cmresponse "monorepo/bin-call-manager/pkg/listenhandler/models/response"

	"monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

// CallV1CallHealth sends the request for call health-check
//
// delay: milliseconds
func (r *requestHandler) CallV1CallHealth(ctx context.Context, id uuid.UUID, delay, retryCount int) error {
	uri := fmt.Sprintf("/v1/calls/%s/health-check", id)

	m, err := json.Marshal(cmrequest.V1DataCallsIDHealthPost{
		RetryCount: retryCount,
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

	res, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCallsCallIDActionTimeout, requestTimeoutDefault, delay, ContentTypeJSON, m)
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

	res, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCallsCallIDActionNext, requestTimeoutDefault, 0, ContentTypeJSON, m)
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
// to creating a calls and groupcalls depending on the destination's type.
// it returns created calls and groupcalls if it succeed.
func (r *requestHandler) CallV1CallsCreate(
	ctx context.Context,
	customerID uuid.UUID,
	flowID uuid.UUID,
	masterCallID uuid.UUID,
	source *commonaddress.Address,
	destinations []commonaddress.Address,
	ealryExecution bool,
	connect bool,
) ([]*cmcall.Call, []*cmgroupcall.Groupcall, error) {
	uri := "/v1/calls"

	m, err := json.Marshal(cmrequest.V1DataCallsPost{
		CustomerID:     customerID,
		FlowID:         flowID,
		MasterCallID:   masterCallID,
		Source:         *source,
		Destinations:   destinations,
		EarlyExecution: ealryExecution,
		Connect:        connect,
	})
	if err != nil {
		return nil, nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCalls, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, nil, err
	case tmp == nil:
		// not found
		return nil, nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	res := &cmresponse.V1ResponseCallsPost{}
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, nil, err
	}

	return res.Calls, res.Groupcalls, nil
}

// CallV1CallCreateWithID sends a request to call-manager
// to creating a call with the given id.
// it returns created call if it succeed.
func (r *requestHandler) CallV1CallCreateWithID(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	flowID uuid.UUID,
	activeflowID uuid.UUID,
	masterCallID uuid.UUID,
	source *commonaddress.Address,
	destination *commonaddress.Address,
	groupcallID uuid.UUID,
	earlyExecution bool,
	connect bool,
) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls/%s", id.String())

	m, err := json.Marshal(cmrequest.V1DataCallsIDPost{
		CustomerID:     customerID,
		FlowID:         flowID,
		ActiveflosID:   activeflowID,
		MasterCallID:   masterCallID,
		Source:         *source,
		Destination:    *destination,
		GroupcallID:    groupcallID,
		EarlyExecution: earlyExecution,
		Connect:        connect,
	})
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

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceCallCalls, requestTimeoutDefault, 0, ContentTypeNone, nil)
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
func (r *requestHandler) CallV1CallGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = parseFilters(uri, filters)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceCallCalls, 30000, 0, ContentTypeNone, nil)
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

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceCallCalls, requestTimeoutDefault, 0, ContentTypeNone, nil)
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

	m, err := json.Marshal(cmrequest.V1DataCallsIDChainedCallIDsPost{
		ChainedCallID: chainedCallID,
	})
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

	res, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceCallCalls, requestTimeoutDefault, 0, ContentTypeNone, nil)
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

	m, err := json.Marshal(cmrequest.V1DataCallsIDExternalMediaPost{
		ExternalHost:   externalHost,
		Encapsulation:  encapsulation,
		Transport:      transport,
		ConnectionType: connectionType,
		Format:         format,
		Direction:      direction,
	})
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

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceCallCallsCallIDExternalMedia, requestTimeoutDefault, 0, ContentTypeNone, nil)
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

	m, err := json.Marshal(cmrequest.V1DataCallsIDDigitsPost{
		Digits: digits,
	})
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

	m, err := json.Marshal(cmrequest.V1DataCallsIDRecordingStartPost{
		Format:       format,
		EndOfSilence: endOfSilence,
		EndOfKey:     endOfKey,
		Duration:     duration,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCallsCallIDRecordingStart, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCallsCallIDRecordingStop, requestTimeoutDefault, 0, ContentTypeNone, nil)
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

// CallV1CallUpdateConfbridgeID sends a request to call-manager
// to updates the given call's confbridge id.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallUpdateConfbridgeID(ctx context.Context, callID uuid.UUID, confbirdgeID uuid.UUID) (*cmcall.Call, error) {
	uri := fmt.Sprintf("/v1/calls/%s/confbridge_id", callID)

	m, err := json.Marshal(cmrequest.V1DataCallsIDConfbridgeIDPut{
		ConfbridgeID: confbirdgeID,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPut, resourceCallCallsCallIDConfbridgeID, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// CallV1CallTalk sends a request to call-manager
// to talk to the call directly.
// it returns error if something went wrong.
// rqeuestTimeout: timeout in milliseconds
func (r *requestHandler) CallV1CallTalk(ctx context.Context, callID uuid.UUID, text string, gender string, language string, rqeuestTimeout int) error {
	uri := fmt.Sprintf("/v1/calls/%s/talk", callID)

	m, err := json.Marshal(cmrequest.V1DataCallsIDTalkPost{
		Text:     text,
		Gender:   gender,
		Language: language,
	})
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCallsCallIDTalk, rqeuestTimeout, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// CallV1CallPlay sends a request to call-manager
// to play the given media urls.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallPlay(ctx context.Context, callID uuid.UUID, mediaURLs []string) error {
	uri := fmt.Sprintf("/v1/calls/%s/play", callID)

	m, err := json.Marshal(cmrequest.V1DataCallsIDPlayPost{
		MediaURLs: mediaURLs,
	})
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCallsCallIDPlay, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// CallV1CallMediaStop sends a request to call-manager
// to stop the media playing(play, talk).
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallMediaStop(ctx context.Context, callID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/calls/%s/media_stop", callID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCallsCallIDPlay, requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// CallV1CallHoldOn sends a request to call-manager
// to hold the call.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallHoldOn(ctx context.Context, callID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/calls/%s/hold", callID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCallsCallIDHold, requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// CallV1CallHoldOff sends a request to call-manager
// to unhold the call.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallHoldOff(ctx context.Context, callID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/calls/%s/hold", callID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceCallCallsCallIDHold, requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// CallV1CallMuteOn sends a request to call-manager
// to mute the call.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallMuteOn(ctx context.Context, callID uuid.UUID, direction cmcall.MuteDirection) error {
	uri := fmt.Sprintf("/v1/calls/%s/mute", callID)

	m, err := json.Marshal(cmrequest.V1DataCallsIDMutePost{
		Direction: direction,
	})
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCallsCallIDMute, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// CallV1CallMuteOff sends a request to call-manager
// to unmute the call.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallMuteOff(ctx context.Context, callID uuid.UUID, direction cmcall.MuteDirection) error {
	uri := fmt.Sprintf("/v1/calls/%s/mute", callID)

	m, err := json.Marshal(cmrequest.V1DataCallsIDMuteDelete{
		Direction: direction,
	})
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceCallCallsCallIDMute, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// CallV1CallMusicOnHoldOn sends a request to call-manager
// to music on hold the call.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallMusicOnHoldOn(ctx context.Context, callID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/calls/%s/moh", callID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCallsCallIDMOH, requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// CallV1CallMusicOnHoldOff sends a request to call-manager
// to music on hold off the call.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallMusicOnHoldOff(ctx context.Context, callID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/calls/%s/moh", callID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceCallCallsCallIDMOH, requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// CallV1CallSilenceOn sends a request to call-manager
// to music on hold the call.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallSilenceOn(ctx context.Context, callID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/calls/%s/silence", callID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallCallsCallIDSilence, requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// CallV1CallSilenceOff sends a request to call-manager
// to music on hold off the call.
// it returns error if something went wrong.
func (r *requestHandler) CallV1CallSilenceOff(ctx context.Context, callID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/calls/%s/silence", callID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceCallCallsCallIDSilence, requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}
