package requesthandler

import (
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmrequest "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/request"
	cmresponse "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/response"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

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

// CMCallExternalMedia sends a request to call-manager
// to creating a external media.
// it returns created external media info if it succeed.
func (r *requestHandler) CMCallExternalMedia(
	callID uuid.UUID,
	externalHost string,
	encapsulation string,
	transport string,
	connectionType string,
	format string,
	direction string,
) (addrIP string, addrPort int, errRet error) {
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
		return "", 0, err
	}

	res, err := r.sendRequestCall(uri, rabbitmqhandler.RequestMethodPost, resourceCallCall, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return "", 0, err
	case res == nil:
		// not found
		return "", 0, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return "", 0, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData cmresponse.V1ResponseCallsIDExternalMediaPost
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return "", 0, err
	}

	return resData.MediaAddrIP, resData.MediaAddrPort, nil
}
