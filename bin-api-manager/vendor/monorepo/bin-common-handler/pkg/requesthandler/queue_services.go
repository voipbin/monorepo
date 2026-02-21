package requesthandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/service"
	"monorepo/bin-common-handler/models/sock"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"
	qmrequest "monorepo/bin-queue-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

// QueueV1ServiceTypeQueuecallStart sends a request to queue-manager
// to starts a queuecall service.
// it returns created service if it succeed.
func (r *requestHandler) QueueV1ServiceTypeQueuecallStart(ctx context.Context, queueID uuid.UUID, activeflowID uuid.UUID, referenceType qmqueuecall.ReferenceType, referenceID uuid.UUID) (*service.Service, error) {
	uri := "/v1/services/type/queuecall"

	data := &qmrequest.V1DataServicesTypeQueuecallPost{
		QueueID:       queueID,
		ActiveflowID:  activeflowID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPost, "queue/services/type/queuecall", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res service.Service
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
