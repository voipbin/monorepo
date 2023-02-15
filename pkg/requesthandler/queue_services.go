package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	qmqueuecall "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	qmservice "gitlab.com/voipbin/bin-manager/queue-manager.git/models/service"
	qmrequest "gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// QueueV1ServiceTypeQueuecallStart sends a request to queue-manager
// to starts a queuecall service.
// it returns created service if it succeed.
func (r *requestHandler) QueueV1ServiceTypeQueuecallStart(ctx context.Context, queueID uuid.UUID, activeflowID uuid.UUID, referenceType qmqueuecall.ReferenceType, referenceID uuid.UUID, exitActionID uuid.UUID) (*qmservice.Service, error) {
	uri := "/v1/services/type/queuecall"

	data := &qmrequest.V1DataServicesTypeQueuecallPost{
		QueueID:       queueID,
		ActiveflowID:  activeflowID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		ExitActionID:  exitActionID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestQueue(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceQueueServiceTypeQueuecall, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res qmservice.Service
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
