package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmrecording "monorepo/bin-call-manager/models/recording"
	cmrequest "monorepo/bin-call-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

// CallV1ConfbridgeCreate sends the request for confbridge create.
func (r *requestHandler) CallV1ConfbridgeCreate(ctx context.Context, customerID uuid.UUID, confbridgeType cmconfbridge.Type) (*cmconfbridge.Confbridge, error) {
	uri := "/v1/confbridges"

	m, err := json.Marshal(cmrequest.V1DataConfbridgesPost{
		CustomerID: customerID,
		Type:       confbridgeType,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, "call/confbridges", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("no response found")
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmconfbridge.Confbridge
	if errUnmarshal := json.Unmarshal([]byte(tmp.Data), &res); errUnmarshal != nil {
		return nil, errUnmarshal
	}

	return &res, nil
}

// CallV1ConfbridgeDelete sends the request for confbridge delete.
func (r *requestHandler) CallV1ConfbridgeDelete(ctx context.Context, confbridgeID uuid.UUID) (*cmconfbridge.Confbridge, error) {
	uri := fmt.Sprintf("/v1/confbridges/%s", confbridgeID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodDelete, "call/confbridges", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("no response found")
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmconfbridge.Confbridge
	if errUnmarshal := json.Unmarshal([]byte(tmp.Data), &res); errUnmarshal != nil {
		return nil, errUnmarshal
	}

	return &res, nil
}

// CallV1ConfbridgeGet sends a request to call-manager
// to getting a confbridge.
// it returns given call id's call if it succeed.
func (r *requestHandler) CallV1ConfbridgeGet(ctx context.Context, confbridgeID uuid.UUID) (*cmconfbridge.Confbridge, error) {
	uri := fmt.Sprintf("/v1/confbridges/%s", confbridgeID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodGet, "call/calls", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmconfbridge.Confbridge
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallV1ConfbridgeCallKick sends the kick request to the confbridge.
func (r *requestHandler) CallV1ConfbridgeCallKick(ctx context.Context, confbridgeID uuid.UUID, callID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/confbridges/%s/calls/%s", confbridgeID, callID)

	res, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodDelete, "call/confbridges", requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// CallV1ConfbridgeCallAdd sends the call join request to the confbridge.
func (r *requestHandler) CallV1ConfbridgeCallAdd(ctx context.Context, confbridgeID uuid.UUID, callID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/confbridges/%s/calls/%s", confbridgeID, callID)

	res, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, "call/confbridges", requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// CallV1ConfbridgeExternalMediaStart sends a request to call-manager
// to start the external media.
// it returns error if something went wrong.
func (r *requestHandler) CallV1ConfbridgeExternalMediaStart(
	ctx context.Context,
	confbridgeID uuid.UUID,
	externalHost string, // external host:port
	encapsulation string, // rtp
	transport string, // udp
	connectionType string, // client,server
	format string, // ulaw
	direction string, // in,out,both
) (*cmconfbridge.Confbridge, error) {
	uri := fmt.Sprintf("/v1/confbridges/%s/external-media", confbridgeID)

	reqData := &cmrequest.V1DataConfbridgesIDExternalMediaPost{
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

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, "call/confbridges/<confbridge-id>/external-media", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmconfbridge.Confbridge
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallV1ConfbridgeExternalMediaStop sends a request to call-manager
// to stop the external media.
// it returns error if something went wrong.
func (r *requestHandler) CallV1ConfbridgeExternalMediaStop(ctx context.Context, confbridgeID uuid.UUID) (*cmconfbridge.Confbridge, error) {
	uri := fmt.Sprintf("/v1/confbridges/%s/external-media", confbridgeID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodDelete, "call/confbridges/<confbridge-id>/external-media", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmconfbridge.Confbridge
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallV1ConfbridgeRecordingStart sends a request to call-manager
// to starts the given confbridge's recording.
// it returns error if something went wrong.
func (r *requestHandler) CallV1ConfbridgeRecordingStart(ctx context.Context, confbridgeID uuid.UUID, format cmrecording.Format, endOfSilence int, endOfKey string, duration int) (*cmconfbridge.Confbridge, error) {
	uri := fmt.Sprintf("/v1/confbridges/%s/recording_start", confbridgeID)

	reqData := &cmrequest.V1DataConfbridgesIDRecordingStartPost{
		Format:       format,
		EndOfSilence: endOfSilence,
		EndOfKey:     endOfKey,
		Duration:     duration,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, "call/confbridges/<confbridge-id>/recording-start", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmconfbridge.Confbridge
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallV1ConfbridgeRecordingStop sends a request to call-manager
// to starts the given confbridge's recording.
// it returns error if something went wrong.
func (r *requestHandler) CallV1ConfbridgeRecordingStop(ctx context.Context, confbridgeID uuid.UUID) (*cmconfbridge.Confbridge, error) {
	uri := fmt.Sprintf("/v1/confbridges/%s/recording_stop", confbridgeID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, "call/confbridges/<confbridge-id>/recording-stop", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmconfbridge.Confbridge
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallV1ConfbridgeFlagAdd sends a request to call-manager
// to add the flag to the given confbridge.
// it returns error if something went wrong.
func (r *requestHandler) CallV1ConfbridgeFlagAdd(ctx context.Context, confbridgeID uuid.UUID, flag cmconfbridge.Flag) (*cmconfbridge.Confbridge, error) {
	uri := fmt.Sprintf("/v1/confbridges/%s/flags", confbridgeID)

	reqData := &cmrequest.V1DataConfbridgesIDFlagsPost{
		Flag: flag,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, "call/confbridges/<confbridge-id>/recording-stop", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmconfbridge.Confbridge
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallV1ConfbridgeFlagRemove sends a request to call-manager
// to remove the flag from the given confbridge.
// it returns error if something went wrong.
func (r *requestHandler) CallV1ConfbridgeFlagRemove(ctx context.Context, confbridgeID uuid.UUID, flag cmconfbridge.Flag) (*cmconfbridge.Confbridge, error) {
	uri := fmt.Sprintf("/v1/confbridges/%s/flags", confbridgeID)

	reqData := &cmrequest.V1DataConfbridgesIDFlagsDelete{
		Flag: flag,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodDelete, "call/confbridges/<confbridge-id>/recording-stop", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmconfbridge.Confbridge
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallV1ConfbridgeTerminate sends a request to call-manager
// to terminate the confbridge.
// it returns error if something went wrong.
func (r *requestHandler) CallV1ConfbridgeTerminate(ctx context.Context, confbridgeID uuid.UUID) (*cmconfbridge.Confbridge, error) {
	uri := fmt.Sprintf("/v1/confbridges/%s/terminate", confbridgeID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, "call/confbridges/<confbridge-id>/terminate", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmconfbridge.Confbridge
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallV1ConfbridgeRing sends a request to call-manager
// to ring the confbridge.
// it returns error if something went wrong.
func (r *requestHandler) CallV1ConfbridgeRing(ctx context.Context, confbridgeID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/confbridges/%s/ring", confbridgeID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, "call/confbridges/<confbridge-id>/ring", requestTimeoutDefault, 0, ContentTypeNone, nil)
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

// CallV1ConfbridgeAnswer sends a request to call-manager
// to answer the confbridge.
// it returns error if something went wrong.
func (r *requestHandler) CallV1ConfbridgeAnswer(ctx context.Context, confbridgeID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/confbridges/%s/answer", confbridgeID)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, "call/confbridges/<confbridge-id>/answer", requestTimeoutDefault, 0, ContentTypeNone, nil)
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
