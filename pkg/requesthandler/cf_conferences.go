package requesthandler

import (
	"fmt"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// CFConferencesIDDelete sends the request for confbridge delete.
// conferenceID: conference id
// delay: millisecond
func (r *requestHandler) CFConferencesIDDelete(conferenceID uuid.UUID, delay int) error {
	uri := fmt.Sprintf("/v1/confbridges/%s", conferenceID)

	res, err := r.sendRequestCall(uri, rabbitmqhandler.RequestMethodDelete, resourceCFConferences, requestTimeoutDefault, delay, ContentTypeJSON, nil)
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
