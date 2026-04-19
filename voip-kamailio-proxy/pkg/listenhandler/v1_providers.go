package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/voip-kamailio-proxy/pkg/listenhandler/request"
	"monorepo/voip-kamailio-proxy/pkg/siphandler"

	"github.com/sirupsen/logrus"
)

// processV1ProvidersHealthPost handles POST /v1/providers/health.
// It sends a SIP OPTIONS to the given hostname and returns the health result.
func (h *listenHandler) processV1ProvidersHealthPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ProvidersHealthPost",
		"request": m,
	})

	var req request.V1DataProvidersHealthPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	if req.Hostname == "" {
		log.Debugf("Empty hostname in health check request")
		return simpleResponse(400), nil
	}

	result, err := siphandler.SendOptionsCheck(ctx, req.Hostname, h.sipTimeout)
	if err != nil {
		log.Errorf("Could not send SIP OPTIONS. hostname: %s, err: %v", req.Hostname, err)
		return simpleResponse(500), nil
	}
	log.WithField("result", result).Debugf("SIP OPTIONS health check result. hostname: %s", req.Hostname)

	res := &request.V1ResponseProvidersHealthPost{
		Status:     "unhealthy",
		ResultCode: result.ResponseCode,
	}
	if result.Healthy {
		res.Status = "healthy"
	}

	data, err := json.Marshal(res)
	if err != nil {
		log.Errorf("Could not marshal health result. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
