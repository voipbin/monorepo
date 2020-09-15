package requesthandler

import (
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/rabbitmq"
)

// ConferenceConferenceTerminate sends the request for conference terminating
// conferenceID: conference id
// delay: millisecond
func (r *requestHandler) CallConferenceTerminate(conferenceID uuid.UUID, reason string, delay int) error {
	uri := fmt.Sprintf("/v1/conferences/%s", conferenceID)

	type Data struct {
		Reason string `json:"reason"`
	}

	m, err := json.Marshal(Data{
		reason,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestCall(uri, rabbitmq.RequestMethodDelete, resourceCallChannelsHealth, requestTimeoutDefault, delay, ContentTypeJSON, m)
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
