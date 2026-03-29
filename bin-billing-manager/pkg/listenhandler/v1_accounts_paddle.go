package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// paddlePortalSessionResponse is the response for the portal session endpoint.
type paddlePortalSessionResponse struct {
	URL string `json:"url"`
}

// processV1AccountsIDPaddlePortalSessionPost handles POST /v1/accounts/<id>/paddle_portal_session
func (h *listenHandler) processV1AccountsIDPaddlePortalSessionPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1AccountsIDPaddlePortalSessionPost",
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		return simpleResponse(400), nil
	}

	url, err := h.accountHandler.PaddleCreatePortalSession(ctx, id)
	if err != nil {
		log.Errorf("Could not create portal session: %v", err)
		return simpleResponse(400), nil
	}

	resp := paddlePortalSessionResponse{URL: url}
	data, err := json.Marshal(resp)
	if err != nil {
		log.Errorf("Could not marshal response: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
