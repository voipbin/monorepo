package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"
)

// processV1PingGet handles GET /v1/ping by returning the pod's PingResult.
// This is the per-pod liveness probe used by ai-manager preflight.
func (h *listenHandler) processV1PingGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1PingGet",
		"request": m,
	})

	res, err := h.pipecatcallHandler.Ping(ctx)
	if err != nil {
		log.Debugf("Could not get ping result. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		log.Debugf("Could not marshal ping result. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
