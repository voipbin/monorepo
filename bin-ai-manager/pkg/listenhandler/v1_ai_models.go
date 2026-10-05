package listenhandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-common-handler/models/sock"
)

// processV1AIModelsGet handles GET /v1/ai_models request.
// It returns the customer-facing model catalog. No pagination and no database access.
func (h *listenHandler) processV1AIModelsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1AIModelsGet",
	})

	data, err := json.Marshal(ai.CatalogView())
	if err != nil {
		log.Errorf("Could not marshal the response message. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
