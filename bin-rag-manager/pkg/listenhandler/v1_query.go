package listenhandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
)

func (h *listenHandler) processV1QueryPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1QueryPost",
	})

	var req struct {
		RagID uuid.UUID `json:"rag_id"`
		Query string    `json:"query"`
		TopK  int       `json:"top_k"`
	}
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal request. err: %v", err)
		return simpleResponse(400), nil
	}

	if req.RagID == uuid.Nil {
		log.Errorf("Missing rag_id in query request.")
		return simpleResponse(400), nil
	}

	if req.Query == "" {
		log.Errorf("Missing query in query request.")
		return simpleResponse(400), nil
	}

	res, err := h.ragHandler.QueryRag(ctx, req.RagID, req.Query, req.TopK)
	if err != nil {
		log.Errorf("Could not query rag. err: %v", err)
		return errorResponse(err), nil
	}

	return jsonResponse(200, res), nil
}
