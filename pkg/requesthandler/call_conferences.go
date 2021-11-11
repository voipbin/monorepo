package requesthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// ConferenceConferenceTerminate sends the request for conference terminating
// conferenceID: conference id
// delay: millisecond
func (r *requestHandler) CallConferenceTerminate(ctx context.Context, conferenceID uuid.UUID, delay int) error {
	uri := fmt.Sprintf("/v1/conferences/%s", conferenceID)

	res, err := r.sendRequestCall(uri, rabbitmqhandler.RequestMethodDelete, resourceCallChannelsHealth, requestTimeoutDefault, delay, ContentTypeJSON, nil)
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
