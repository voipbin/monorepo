package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// processV1AccountsIDAllowanceGet handles GET /v1/accounts/<account-id>/allowance request
func (h *listenHandler) processV1AccountsIDAllowanceGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsIDAllowanceGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	accountID := uuid.FromStringOrNil(uriItems[3])

	a, err := h.allowanceHandler.GetCurrentCycle(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get current allowance. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(a)
	if err != nil {
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
