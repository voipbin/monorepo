package requesthandler

import (
	"fmt"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// CMCallsIDDelete sends the request for call hangup.
// callID: call id
// delay: millisecond
func (r *requestHandler) CMCallsIDDelete(callID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/calls/%s", callID)

	res, err := r.sendRequestCall(uri, rabbitmqhandler.RequestMethodDelete, resourceCMCalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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
