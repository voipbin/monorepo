package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	qmqueuecallreference "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecallreference"
)

// QMV1QueuecallGet sends a request to queue-manager
// to get the queuecall.
// it returns an queuecall if it succeed.
func (r *requestHandler) QMV1QueuecallReferenceGet(ctx context.Context, referenceID uuid.UUID) (*qmqueuecallreference.QueuecallReference, error) {
	uri := fmt.Sprintf("/v1/queuecallreferences/%s", referenceID)

	tmp, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodGet, resourceQMQueuecallreferences, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res qmqueuecallreference.QueuecallReference
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
