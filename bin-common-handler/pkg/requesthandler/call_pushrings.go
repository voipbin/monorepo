package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	cmpushring "monorepo/bin-call-manager/models/pushring"
	cmrequest "monorepo/bin-call-manager/pkg/listenhandler/models/request"
	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
)

// requestTimeoutCallPushRingStart is the timeout of the push ring start. The call-manager answers inside 4 s by its
// own deadline, so a timeout here means the answer is unknown.
const requestTimeoutCallPushRingStart int = 5000

// CallV1PushRingStart sends the request to start a push ring session for the call to the extension.
// It returns started false when no session was started; the call then continues as it did before.
func (r *requestHandler) CallV1PushRingStart(
	ctx context.Context,
	customerID uuid.UUID,
	destination commonaddress.Address,
	confbridgeID uuid.UUID,
	masterCallID uuid.UUID,
	flowID uuid.UUID,
	source commonaddress.Address,
) (*cmpushring.StartResult, error) {
	uri := "/v1/push-rings"

	m, err := json.Marshal(cmrequest.V1DataPushRingsPost{
		CustomerID:   customerID,
		Destination:  destination,
		ConfbridgeID: confbridgeID,
		MasterCallID: masterCallID,
		FlowID:       flowID,
		Source:       source,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodPost, "call/push-rings", requestTimeoutCallPushRingStart, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmpushring.StartResult
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CallV1PushRingExpire sends the request to end the ring window of the push ring session.
//
// delay: milliseconds
func (r *requestHandler) CallV1PushRingExpire(ctx context.Context, sessionID uuid.UUID, delay int) error {
	uri := fmt.Sprintf("/v1/push-rings/%s/expire", sessionID)

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodPost, "call/push-rings/<push-ring-id>/expire", requestTimeoutDefault, delay, ContentTypeJSON, nil)
	if err != nil {
		return err
	}

	return parseResponse(tmp, nil)
}

// CallV1PushRingClose sends the request to close the push ring session without hanging up its master call.
func (r *requestHandler) CallV1PushRingClose(ctx context.Context, sessionID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/push-rings/%s/close", sessionID)

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodPost, "call/push-rings/<push-ring-id>/close", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return err
	}

	return parseResponse(tmp, nil)
}
