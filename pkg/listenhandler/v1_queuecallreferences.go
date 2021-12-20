package listenhandler

import (
	"context"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processV1QueuecallreferencesIDDelete handles Delete /v1/queuecallreferences/<reference-id> request
func (h *listenHandler) processV1QueuecallreferencesIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "processV1QueuecallreferencesIDDelete",
			"queuecall_id": id,
		})
	log.Debug("Executing processV1QueuecallreferencesIDDelete.")

	if err := h.queuecallHandler.KickByReferenceID(ctx, id); err != nil {
		log.Errorf("Could not leave the queuecall from the queue. err: %v", err)
		return simpleResponse(500), nil
	}
	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}
