package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// processV1FlowsIDDirectHashRegeneratePost handles POST /v1/flows/<flow-id>/direct-hash-regenerate request
func (h *listenHandler) processV1FlowsIDDirectHashRegeneratePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1FlowsIDDirectHashRegeneratePost",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/flows/a6f4eae8-8a74-11ea-af75-3f1e61b9a236/direct-hash-regenerate"
	tmpVals := strings.Split(u.Path, "/")
	if len(tmpVals) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(tmpVals[3])
	log = log.WithField("flow_id", id)
	log.Debug("Received request.")

	tmp, err := h.flowHandler.DirectHashRegenerate(ctx, id)
	if err != nil {
		log.Errorf("Could not regenerate direct hash. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal response. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
