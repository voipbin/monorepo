package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/pkg/listenhandler/models/response"
)

// processV1FailedEventsRetryPost handles POST /v1/failed_events/retry request
func (h *listenHandler) processV1FailedEventsRetryPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1FailedEventsRetryPost",
		"request": m,
	})

	retried, succeeded, exhausted, err := h.failedEventHandler.RetryPending(ctx)
	if err != nil {
		log.Errorf("Could not retry pending failed events. err: %v", err)
		return errorResponse(err), nil
	}

	tmp := &response.V1ResponseFailedEventsRetry{
		Retried:   retried,
		Succeeded: succeeded,
		Exhausted: exhausted,
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal failed event retry response. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
