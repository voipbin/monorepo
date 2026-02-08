package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-rag-manager/pkg/raghandler"

	"github.com/sirupsen/logrus"
)

// processV1RagQueryPost handles POST /v1/rags/query
func (h *listenHandler) processV1RagQueryPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "processV1RagQueryPost")

	var req raghandler.QueryRequest
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Infof("Could not unmarshal request data. err: %v", err)
		return simpleResponse(400), nil
	}

	if req.Query == "" {
		log.Infof("Empty query")
		return simpleResponse(400), nil
	}

	result, err := h.ragHandler.Query(ctx, &req)
	if err != nil {
		log.Errorf("Could not process query. err: %v", err)
		return simpleResponse(500), nil
	}
	log.Debugf("Query processed successfully. sources: %d", len(result.Sources))

	return jsonResponse(200, result), nil
}
